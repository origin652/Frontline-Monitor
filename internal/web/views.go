package web

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"vps-monitor/internal/model"
)

type baseView struct {
	Title     string
	Page      string
	NodeID    string
	LeaderID  string
	Generated time.Time
}

type overviewView struct {
	Base        baseView
	Ingress     model.IngressState
	Nodes       []nodeCardView
	Incidents   []model.Incident
	Events      []model.Event
	ProbeMatrix []probeMatrixRow
}

type nodeCardView struct {
	State       model.NodeState
	PeerIP      string
	Trend       []model.MetricPoint
	ServicesBad int
}

type nodeView struct {
	Base        baseView
	Ingress     model.IngressState
	Detail      model.NodeDetail
	CPUHistory  []model.MetricPoint
	MemHistory  []model.MetricPoint
	DiskHistory []model.MetricPoint
}

type eventsView struct {
	Base              baseView
	Ingress           model.IngressState
	Incidents         []model.Incident
	Events            []model.Event
	TestChannels      []string
	TestRequiresToken bool
}

type probeMatrixRow struct {
	Source string
	Cells  []probeMatrixCell
}

type probeMatrixCell struct {
	Target string
	Status string
	Label  string
}

func (s *Server) buildOverviewView(ctx context.Context) (*overviewView, error) {
	snapshot, err := s.snapshot(ctx)
	if err != nil {
		return nil, err
	}

	nodes := make([]nodeCardView, 0, len(snapshot.Nodes))
	for _, state := range snapshot.Nodes {
		peer, _ := s.cfg.PeerByID(state.NodeID)
		trend, _ := s.store.History(ctx, state.NodeID, "cpu_pct", time.Now().UTC().Add(-24*time.Hour), time.Now().UTC())
		nodes = append(nodes, nodeCardView{
			State:       state,
			PeerIP:      peer.PublicIPv4,
			Trend:       trend,
			ServicesBad: len(countFailingServices(state.Services)),
		})
	}
	return &overviewView{
		Base: baseView{
			Title:     "Cluster Overview",
			Page:      "overview",
			NodeID:    snapshot.NodeID,
			LeaderID:  snapshot.LeaderID,
			Generated: snapshot.GeneratedAt,
		},
		Ingress:     snapshot.Ingress,
		Nodes:       nodes,
		Incidents:   snapshot.Incidents,
		Events:      snapshot.Events,
		ProbeMatrix: s.buildProbeMatrix(snapshot.Nodes),
	}, nil
}

func (s *Server) buildNodeView(ctx context.Context, nodeID string) (*nodeView, error) {
	detail, err := s.nodeDetail(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	ingress, _ := s.store.GetIngressState(ctx)
	if ingress == nil {
		ingress = &model.IngressState{}
	}
	cpuHistory, _ := s.store.History(ctx, nodeID, "cpu_pct", time.Now().UTC().Add(-24*time.Hour), time.Now().UTC())
	memHistory, _ := s.store.History(ctx, nodeID, "mem_pct", time.Now().UTC().Add(-24*time.Hour), time.Now().UTC())
	diskHistory, _ := s.store.History(ctx, nodeID, "disk_pct", time.Now().UTC().Add(-24*time.Hour), time.Now().UTC())
	return &nodeView{
		Base: baseView{
			Title:     fmt.Sprintf("Node %s", nodeID),
			Page:      "node",
			NodeID:    s.cfg.Cluster.NodeID,
			LeaderID:  s.cluster.LeaderID(),
			Generated: time.Now().UTC(),
		},
		Ingress:     *ingress,
		Detail:      detail,
		CPUHistory:  cpuHistory,
		MemHistory:  memHistory,
		DiskHistory: diskHistory,
	}, nil
}

func (s *Server) buildEventsView(ctx context.Context) (*eventsView, error) {
	incidents, err := s.store.ListIncidents(ctx, "", 30)
	if err != nil {
		return nil, err
	}
	events, err := s.store.ListEvents(ctx, 80)
	if err != nil {
		return nil, err
	}
	ingress, _ := s.store.GetIngressState(ctx)
	if ingress == nil {
		ingress = &model.IngressState{}
	}
	return &eventsView{
		Base: baseView{
			Title:     "Events",
			Page:      "events",
			NodeID:    s.cfg.Cluster.NodeID,
			LeaderID:  s.cluster.LeaderID(),
			Generated: time.Now().UTC(),
		},
		Ingress:           *ingress,
		Incidents:         incidents,
		Events:            events,
		TestChannels:      s.enabledAlertChannels(),
		TestRequiresToken: strings.TrimSpace(os.Getenv("MONITOR_TEST_ALERT_TOKEN")) != "",
	}, nil
}

func (s *Server) snapshot(ctx context.Context) (*model.ClusterSnapshot, error) {
	nodes, err := s.nodeStates(ctx)
	if err != nil {
		return nil, err
	}
	incidents, err := s.store.ListIncidents(ctx, model.IncidentStatusActive, 8)
	if err != nil {
		return nil, err
	}
	events, err := s.store.ListEvents(ctx, 10)
	if err != nil {
		return nil, err
	}
	ingress, err := s.store.GetIngressState(ctx)
	if err != nil {
		return nil, err
	}
	if ingress == nil {
		ingress = &model.IngressState{}
	}
	return &model.ClusterSnapshot{
		GeneratedAt: time.Now().UTC(),
		NodeID:      s.cfg.Cluster.NodeID,
		LeaderID:    s.cluster.LeaderID(),
		Ingress:     *ingress,
		Nodes:       nodes,
		Incidents:   incidents,
		Events:      events,
	}, nil
}

func (s *Server) nodeStates(ctx context.Context) ([]model.NodeState, error) {
	states, err := s.store.ListNodeStates(ctx)
	if err != nil {
		return nil, err
	}
	stateMap := map[string]model.NodeState{}
	for _, state := range states {
		stateMap[state.NodeID] = state
	}

	out := make([]model.NodeState, 0, len(s.cfg.OrderedPeers()))
	for _, peer := range s.cfg.OrderedPeers() {
		if state, ok := stateMap[peer.NodeID]; ok {
			out = append(out, state)
			continue
		}
		out = append(out, model.NodeState{
			NodeID:          peer.NodeID,
			Status:          model.StatusUnknown,
			Reason:          "awaiting cluster data",
			RuleKey:         "telemetry",
			LastEvaluatedAt: time.Now().UTC(),
		})
	}
	return out, nil
}

func (s *Server) nodeDetail(ctx context.Context, nodeID string) (model.NodeDetail, error) {
	state, err := s.store.GetNodeState(ctx, nodeID)
	if err != nil {
		return model.NodeDetail{}, err
	}
	if state == nil {
		state = &model.NodeState{
			NodeID:          nodeID,
			Status:          model.StatusUnknown,
			Reason:          "awaiting cluster data",
			RuleKey:         "telemetry",
			LastEvaluatedAt: time.Now().UTC(),
		}
	}
	heartbeat, err := s.store.LatestHeartbeat(ctx, nodeID)
	if err != nil {
		return model.NodeDetail{}, err
	}
	probes, err := s.store.RecentProbesForTarget(ctx, nodeID, time.Now().UTC().Add(-2*time.Hour), 20)
	if err != nil {
		return model.NodeDetail{}, err
	}
	incidents, err := s.store.ListIncidentsForNode(ctx, nodeID, 20)
	if err != nil {
		return model.NodeDetail{}, err
	}
	history, err := s.store.History(ctx, nodeID, "cpu_pct", time.Now().UTC().Add(-24*time.Hour), time.Now().UTC())
	if err != nil {
		return model.NodeDetail{}, err
	}
	return model.NodeDetail{
		State:     *state,
		Heartbeat: heartbeat,
		Probes:    probes,
		Incidents: incidents,
		History:   history,
	}, nil
}

func (s *Server) buildProbeMatrix(nodes []model.NodeState) []probeMatrixRow {
	rows := make([]probeMatrixRow, 0, len(s.cfg.Cluster.Peers))
	for _, source := range s.cfg.OrderedPeers() {
		row := probeMatrixRow{Source: source.NodeID}
		for _, target := range s.cfg.OrderedPeers() {
			if source.NodeID == target.NodeID {
				row.Cells = append(row.Cells, probeMatrixCell{Target: target.NodeID, Status: "self", Label: "SELF"})
				continue
			}
			state := "unknown"
			label := "WAIT"
			for _, node := range nodes {
				if node.NodeID != target.NodeID {
					continue
				}
				if node.LastProbeSummary.Reachable {
					state = "healthy"
					label = "OPEN"
				} else if node.Status == model.StatusCritical {
					state = "critical"
					label = "DROP"
				} else if node.Status == model.StatusDegraded {
					state = "degraded"
					label = "THIN"
				}
			}
			row.Cells = append(row.Cells, probeMatrixCell{Target: target.NodeID, Status: state, Label: label})
		}
		rows = append(rows, row)
	}
	return rows
}

func countFailingServices(services []model.ServiceCheck) []string {
	var out []string
	for _, service := range services {
		if service.Status != "active" && service.Status != "running" && service.Status != "healthy" {
			out = append(out, service.Name)
		}
	}
	return out
}

func (s *Server) enabledAlertChannels() []string {
	var channels []string
	for _, notifier := range s.notifiers {
		channels = append(channels, notifier.Name())
	}
	return channels
}
