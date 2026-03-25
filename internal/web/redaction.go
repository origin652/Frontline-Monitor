package web

import "vps-monitor/internal/model"

func (s *Server) redactIngress(ingress *model.IngressState) {
	if ingress == nil {
		return
	}
	ingress.DesiredIP = ""
}

func (s *Server) redactEvents(events []model.Event) []model.Event {
	out := make([]model.Event, len(events))
	for i, event := range events {
		event.Meta = nil
		out[i] = event
	}
	return out
}

func (s *Server) redactNodeState(state *model.NodeState) {
	if state == nil {
		return
	}
}

func (s *Server) redactProbe(probe *model.ProbeObservation) {
	if probe == nil {
		return
	}
	probe.Ports = nil
	probe.HTTPChecks = nil
}

func (s *Server) redactNodeDetail(detail *model.NodeDetail) {
	if detail == nil {
		return
	}
	s.redactNodeState(&detail.State)
	if detail.Heartbeat != nil {
		detail.Heartbeat.DockerChecks = nil
		detail.Heartbeat.LocalHTTPChecks = nil
	}
	for i := range detail.Probes {
		s.redactProbe(&detail.Probes[i])
	}
}

func (s *Server) redactSnapshot(snapshot *model.ClusterSnapshot) {
	if snapshot == nil {
		return
	}
	s.redactIngress(&snapshot.Ingress)
	for i := range snapshot.Nodes {
		s.redactNodeState(&snapshot.Nodes[i])
	}
	snapshot.Events = s.redactEvents(snapshot.Events)
}
