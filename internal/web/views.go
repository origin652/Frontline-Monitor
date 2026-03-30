package web

import (
	"context"
	"time"

	"vps-monitor/internal/model"
)

func (s *Server) snapshot(ctx context.Context, isAdmin bool) (*model.ClusterSnapshot, error) {
	resolver, err := s.newNodeNameResolver(ctx)
	if err != nil {
		return nil, err
	}
	nodes, err := s.nodeStates(ctx, isAdmin, resolver)
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
	s.decorateIncidents(resolver, incidents)
	s.decorateEvents(resolver, events)
	s.decorateIngress(resolver, ingress)
	leaderID := s.cluster.LeaderID()
	snapshot := &model.ClusterSnapshot{
		GeneratedAt: time.Now().UTC(),
		NodeID:      s.cfg.Cluster.NodeID,
		NodeName:    resolver.DisplayName(s.cfg.Cluster.NodeID),
		LeaderID:    leaderID,
		LeaderName:  resolver.DisplayName(leaderID),
		Ingress:     *ingress,
		Nodes:       nodes,
		Incidents:   incidents,
		Events:      events,
	}
	if !isAdmin {
		s.redactSnapshot(snapshot)
	}
	return snapshot, nil
}

func (s *Server) nodeStates(ctx context.Context, isAdmin bool, resolver nodeNameResolver) ([]model.NodeState, error) {
	states, err := s.store.ListNodeStates(ctx)
	if err != nil {
		return nil, err
	}
	stateMap := map[string]model.NodeState{}
	for _, state := range states {
		stateMap[state.NodeID] = state
	}

	members, err := s.cluster.OrderedMembers(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]model.NodeState, 0, len(members))
	for _, member := range members {
		if state, ok := stateMap[member.NodeID]; ok {
			s.decorateNodeState(resolver, &state)
			out = append(out, state)
			continue
		}
		state := model.NodeState{
			NodeID:          member.NodeID,
			NodeName:        resolver.DisplayName(member.NodeID),
			Status:          model.StatusUnknown,
			Reason:          "awaiting cluster data",
			RuleKey:         "telemetry",
			LastEvaluatedAt: time.Now().UTC(),
		}
		out = append(out, state)
	}
	if !isAdmin {
		for i := range out {
			s.redactNodeState(&out[i])
		}
	}
	return out, nil
}

func (s *Server) nodeDetail(ctx context.Context, nodeID string, isAdmin bool, resolver nodeNameResolver) (model.NodeDetail, error) {
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
	s.decorateNodeState(resolver, state)
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
	s.decorateProbes(resolver, probes)
	s.decorateIncidents(resolver, incidents)
	detail := model.NodeDetail{
		State:     *state,
		Heartbeat: heartbeat,
		Probes:    probes,
		Incidents: incidents,
		History:   history,
	}
	if !isAdmin {
		s.redactNodeDetail(&detail)
	}
	return detail, nil
}

func (s *Server) enabledAlertChannels(ctx context.Context) ([]string, error) {
	if s.alertResolver == nil {
		return nil, nil
	}
	return s.alertResolver.EnabledChannels(ctx)
}
