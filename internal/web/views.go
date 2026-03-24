package web

import (
	"context"
	"time"

	"vps-monitor/internal/model"
)

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

func (s *Server) enabledAlertChannels() []string {
	var channels []string
	for _, notifier := range s.notifiers {
		channels = append(channels, notifier.Name())
	}
	return channels
}
