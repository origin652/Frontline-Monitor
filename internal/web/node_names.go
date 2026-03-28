package web

import (
	"context"
	"strings"

	"vps-monitor/internal/model"
)

type nodeNameResolver struct {
	defaultNames map[string]string
	overrides    map[string]model.NodeDisplayName
}

func (s *Server) newNodeNameResolver(ctx context.Context) (nodeNameResolver, error) {
	items, err := s.store.ListNodeDisplayNames(ctx)
	if err != nil {
		return nodeNameResolver{}, err
	}
	members, err := s.cluster.ListMembers(ctx)
	if err != nil {
		return nodeNameResolver{}, err
	}
	overrides := make(map[string]model.NodeDisplayName, len(items))
	for _, item := range items {
		overrides[item.NodeID] = item
	}
	defaultNames := make(map[string]string, len(members)+1)
	for _, member := range members {
		defaultNames[member.NodeID] = strings.TrimSpace(member.DisplayName)
	}
	if _, ok := defaultNames[s.cfg.Cluster.NodeID]; !ok {
		defaultNames[s.cfg.Cluster.NodeID] = s.cfg.DefaultDisplayName()
	}
	return nodeNameResolver{
		defaultNames: defaultNames,
		overrides:    overrides,
	}, nil
}

func (r nodeNameResolver) DisplayName(nodeID string) string {
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return ""
	}
	if item, ok := r.overrides[nodeID]; ok && strings.TrimSpace(item.DisplayName) != "" {
		return strings.TrimSpace(item.DisplayName)
	}
	if displayName := strings.TrimSpace(r.defaultNames[nodeID]); displayName != "" {
		return displayName
	}
	return nodeID
}

func (r nodeNameResolver) Override(nodeID string) (model.NodeDisplayName, bool) {
	item, ok := r.overrides[strings.TrimSpace(nodeID)]
	return item, ok
}

func (r nodeNameResolver) ConfigDisplayName(nodeID string) string {
	return strings.TrimSpace(r.defaultNames[strings.TrimSpace(nodeID)])
}

func (s *Server) decorateIngress(resolver nodeNameResolver, ingress *model.IngressState) {
	if ingress == nil {
		return
	}
	ingress.ActiveNodeName = resolver.DisplayName(ingress.ActiveNodeID)
}

func (s *Server) decorateNodeState(resolver nodeNameResolver, state *model.NodeState) {
	if state == nil {
		return
	}
	state.NodeName = resolver.DisplayName(state.NodeID)
}

func (s *Server) decorateIncidents(resolver nodeNameResolver, incidents []model.Incident) {
	for i := range incidents {
		incidents[i].NodeName = resolver.DisplayName(incidents[i].NodeID)
	}
}

func (s *Server) decorateEvents(resolver nodeNameResolver, events []model.Event) {
	for i := range events {
		events[i].NodeName = resolver.DisplayName(events[i].NodeID)
	}
}

func (s *Server) decorateProbes(resolver nodeNameResolver, probes []model.ProbeObservation) {
	for i := range probes {
		probes[i].SourceNodeName = resolver.DisplayName(probes[i].SourceNodeID)
		probes[i].TargetNodeName = resolver.DisplayName(probes[i].TargetNodeID)
	}
}
