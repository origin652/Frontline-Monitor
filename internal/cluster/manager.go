package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"

	"vps-monitor/internal/config"
	"vps-monitor/internal/store"
)

type Manager struct {
	cfg               *config.Config
	store             *store.Store
	raft              *raft.Raft
	logger            *slog.Logger
	leaderTransitions chan bool
	isLeader          atomic.Bool
	closers           []io.Closer
}

func NewManager(cfg *config.Config, st *store.Store, logger *slog.Logger) (*Manager, error) {
	if err := os.MkdirAll(cfg.Storage.RaftDir, 0o755); err != nil {
		return nil, fmt.Errorf("create raft dir: %w", err)
	}

	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = raft.ServerID(cfg.Cluster.NodeID)
	notifyCh := make(chan bool, 8)
	raftConfig.NotifyCh = notifyCh
	raftConfig.SnapshotInterval = 30 * time.Minute
	raftConfig.SnapshotThreshold = 64

	logStore, err := raftboltdb.NewBoltStore(filepath.Join(cfg.Storage.RaftDir, "raft-log.db"))
	if err != nil {
		return nil, fmt.Errorf("open raft log store: %w", err)
	}
	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(cfg.Storage.RaftDir, "raft-stable.db"))
	if err != nil {
		return nil, fmt.Errorf("open raft stable store: %w", err)
	}
	snapshotStore, err := raft.NewFileSnapshotStore(filepath.Join(cfg.Storage.RaftDir, "snapshots"), 3, io.Discard)
	if err != nil {
		return nil, fmt.Errorf("create raft snapshot store: %w", err)
	}

	addr, err := net.ResolveTCPAddr("tcp", cfg.Cluster.RaftAddr)
	if err != nil {
		return nil, fmt.Errorf("resolve raft addr: %w", err)
	}
	transport, err := raft.NewTCPTransport(cfg.RaftBindAddr(), addr, 3, 10*time.Second, io.Discard)
	if err != nil {
		return nil, fmt.Errorf("create raft transport: %w", err)
	}

	fsm := NewFSM(st)
	nodeRaft, err := raft.NewRaft(raftConfig, fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return nil, fmt.Errorf("create raft instance: %w", err)
	}

	manager := &Manager{
		cfg:               cfg,
		store:             st,
		raft:              nodeRaft,
		logger:            logger,
		leaderTransitions: notifyCh,
		closers:           []io.Closer{logStore, stableStore, transport},
	}
	manager.watchLeadership()

	hasState, err := raft.HasExistingState(logStore, stableStore, snapshotStore)
	if err != nil {
		return nil, fmt.Errorf("check raft state: %w", err)
	}
	if !hasState && manager.shouldBootstrap() {
		servers := make([]raft.Server, 0, len(cfg.Cluster.Peers))
		for _, peer := range cfg.Cluster.Peers {
			servers = append(servers, raft.Server{
				ID:      raft.ServerID(peer.NodeID),
				Address: raft.ServerAddress(peer.RaftAddr),
			})
		}
		if err := nodeRaft.BootstrapCluster(raft.Configuration{Servers: servers}).Error(); err != nil && err != raft.ErrCantBootstrap {
			return nil, fmt.Errorf("bootstrap raft cluster: %w", err)
		}
	}

	return manager, nil
}

func (m *Manager) watchLeadership() {
	go func() {
		for leader := range m.leaderTransitions {
			m.isLeader.Store(leader)
			if leader {
				m.logger.Info("leadership acquired", "node_id", m.cfg.Cluster.NodeID)
			} else {
				m.logger.Info("leadership lost", "node_id", m.cfg.Cluster.NodeID)
			}
		}
	}()
}

func (m *Manager) shouldBootstrap() bool {
	if len(m.cfg.Cluster.Peers) == 0 {
		return true
	}
	return m.cfg.Cluster.Peers[0].NodeID == m.cfg.Cluster.NodeID
}

func (m *Manager) IsLeader() bool {
	return m.isLeader.Load()
}

func (m *Manager) LeaderID() string {
	leaderAddr := string(m.raft.Leader())
	if leaderAddr == "" {
		return ""
	}
	for _, peer := range m.cfg.Cluster.Peers {
		if peer.RaftAddr == leaderAddr {
			return peer.NodeID
		}
	}
	return ""
}

func (m *Manager) LeaderAPIAddr() string {
	return m.cfg.LeaderAPIAddr(m.LeaderID())
}

func (m *Manager) Apply(ctx context.Context, cmdType string, payload any) (any, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal command payload: %w", err)
	}
	return m.ApplyRaw(ctx, cmdType, payloadBytes)
}

func (m *Manager) ApplyRaw(ctx context.Context, cmdType string, payload json.RawMessage) (any, error) {
	if !m.IsLeader() {
		return nil, raft.ErrNotLeader
	}
	data, err := json.Marshal(commandEnvelope{Type: cmdType, Payload: payload})
	if err != nil {
		return nil, fmt.Errorf("marshal command envelope: %w", err)
	}
	timeout := 8 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
		if timeout <= 0 {
			timeout = 2 * time.Second
		}
	}
	future := m.raft.Apply(data, timeout)
	if err := future.Error(); err != nil {
		return nil, err
	}
	if result, ok := future.Response().(CommandResult); ok && result.Error != "" {
		return result, fmt.Errorf(result.Error)
	}
	return future.Response(), nil
}

func (m *Manager) Shutdown(ctx context.Context) error {
	future := m.raft.Shutdown()
	done := make(chan error, 1)
	go func() {
		done <- future.Error()
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		var closeErr error
		for _, closer := range m.closers {
			if closer == nil {
				continue
			}
			if err := closer.Close(); err != nil && closeErr == nil {
				closeErr = err
			}
		}
		if err != nil {
			return err
		}
		return closeErr
	}
}
