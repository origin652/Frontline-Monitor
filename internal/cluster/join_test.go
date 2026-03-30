package cluster_test

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"vps-monitor/internal/cluster"
	"vps-monitor/internal/config"
	"vps-monitor/internal/notify"
	"vps-monitor/internal/store"
	"vps-monitor/internal/web"
)

func TestAutoJoinDynamicMembership(t *testing.T) {
	t.Setenv("MONITOR_INTERNAL_TOKEN", "secret")

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	leaderAPIListener := mustListenTCP(t)
	defer leaderAPIListener.Close()
	leaderAPIAddr := leaderAPIListener.Addr().String()
	leaderRaftAddr := freeTCPAddr(t)
	leaderCfg := dynamicTestConfig(t, "node-a", leaderAPIAddr, leaderRaftAddr)
	leaderCfg.Cluster.DisplayName = "Tokyo-1"
	leaderCfg.Cluster.Bootstrap = true
	leaderStore, leaderManager := newTestManager(t, leaderCfg, logger)
	defer leaderStore.Close()
	defer shutdownManager(t, leaderManager)
	waitForLeader(t, leaderManager)

	leaderServer, err := web.New(leaderCfg, leaderStore, leaderManager, cluster.NewSubmitter(leaderManager, leaderCfg), notify.NewResolver(leaderCfg, leaderStore, logger), logger)
	if err != nil {
		t.Fatalf("new leader web server: %v", err)
	}
	httpServer := &http.Server{
		Handler:           leaderServer.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		_ = httpServer.Serve(leaderAPIListener)
	}()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(ctx)
	}()

	joinerAPIAddr := freeTCPAddr(t)
	joinerRaftAddr := freeTCPAddr(t)
	joinerCfg := dynamicTestConfig(t, "node-b", joinerAPIAddr, joinerRaftAddr)
	joinerCfg.Cluster.DisplayName = "Singapore-1"
	joinerCfg.Cluster.JoinSeeds = []string{leaderAPIAddr}
	joinerStore, joinerManager := newTestManager(t, joinerCfg, logger)
	defer joinerStore.Close()
	defer shutdownManager(t, joinerManager)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := joinerManager.AutoJoin(ctx); err != nil {
		t.Fatalf("AutoJoin() error = %v", err)
	}

	waitForCondition(t, 15*time.Second, func() bool {
		roleMap, err := leaderManager.CurrentRoleMap()
		if err != nil {
			return false
		}
		return roleMap["node-b"] == "voter"
	})

	waitForCondition(t, 15*time.Second, func() bool {
		members, err := joinerStore.ListClusterMembers(context.Background())
		if err != nil {
			return false
		}
		return len(members) == 2
	})
}

func dynamicTestConfig(t *testing.T, nodeID, apiAddr, raftAddr string) *config.Config {
	t.Helper()
	dir := t.TempDir()
	return &config.Config{
		Cluster: config.ClusterConfig{
			NodeID:   nodeID,
			APIAddr:  apiAddr,
			RaftAddr: raftAddr,
			Priority: 100,
		},
		Network: config.NetworkConfig{
			ListenAddr:      apiAddr,
			PublicIPv4:      "203.0.113.10",
			PublicHTTPSPort: 8443,
		},
		Storage: config.StorageConfig{
			DataDir:       dir,
			SQLitePath:    filepath.Join(dir, "monitor.db"),
			RaftDir:       filepath.Join(dir, "raft"),
			RetentionDays: 30,
		},
	}
}

func newTestManager(t *testing.T, cfg *config.Config, logger *slog.Logger) (*store.Store, *cluster.Manager) {
	t.Helper()
	st, err := store.Open(cfg.Storage.SQLitePath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	manager, err := cluster.NewManager(cfg, st, logger)
	if err != nil {
		st.Close()
		t.Fatalf("new manager: %v", err)
	}
	return st, manager
}

func shutdownManager(t *testing.T, manager *cluster.Manager) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = manager.Shutdown(ctx)
}

func waitForLeader(t *testing.T, manager *cluster.Manager) {
	t.Helper()
	waitForCondition(t, 5*time.Second, manager.IsLeader)
}

func waitForCondition(t *testing.T, timeout time.Duration, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("condition not satisfied before timeout")
}

func freeTCPAddr(t *testing.T) string {
	t.Helper()
	ln := mustListenTCP(t)
	defer ln.Close()
	return ln.Addr().String()
}

func mustListenTCP(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen free addr: %v", err)
	}
	return ln
}
