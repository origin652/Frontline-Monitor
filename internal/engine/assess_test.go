package engine

import (
	"testing"
	"time"

	"vps-monitor/internal/config"
	"vps-monitor/internal/model"
)

func TestAssessNodeHeartbeatStaleWithReachableObserverStaysDegraded(t *testing.T) {
	t.Parallel()

	engine := newTestEngine()
	now := time.Now().UTC()
	heartbeat := &model.NodeHeartbeat{
		CollectedAt: now.Add(-50 * time.Second),
	}
	probes := []model.ProbeObservation{
		{SourceNodeID: "node-b", TCP443OK: true},
	}

	state := engine.assessNode("node-a", heartbeat, probes, nil, now, 3, false)
	if state.Status != model.StatusDegraded {
		t.Fatalf("status = %q, want %q", state.Status, model.StatusDegraded)
	}
	if state.Reason != "node is reachable but agent heartbeat is stale" {
		t.Fatalf("reason = %q", state.Reason)
	}
	if state.LastProbeSummary.ExpectedPeers != 3 || state.LastProbeSummary.SuccessfulPeers != 1 || state.LastProbeSummary.TotalPeers != 1 {
		t.Fatalf("summary = %+v", state.LastProbeSummary)
	}
}

func TestAssessNodeHeartbeatStaleWithInsufficientObserverEvidence(t *testing.T) {
	t.Parallel()

	engine := newTestEngine()
	now := time.Now().UTC()
	heartbeat := &model.NodeHeartbeat{
		CollectedAt: now.Add(-50 * time.Second),
	}
	probes := []model.ProbeObservation{
		{SourceNodeID: "node-b"},
	}

	state := engine.assessNode("node-a", heartbeat, probes, nil, now, 3, false)
	if state.Status != model.StatusDegraded {
		t.Fatalf("status = %q, want %q", state.Status, model.StatusDegraded)
	}
	if state.Reason != "heartbeat stale with insufficient observer evidence" {
		t.Fatalf("reason = %q", state.Reason)
	}
	if state.LastProbeSummary.ExpectedPeers != 3 || state.LastProbeSummary.SuccessfulPeers != 0 || state.LastProbeSummary.TotalPeers != 1 {
		t.Fatalf("summary = %+v", state.LastProbeSummary)
	}
}

func TestAssessNodeHeartbeatStaleWithEnoughFailedObserversBecomesCritical(t *testing.T) {
	t.Parallel()

	engine := newTestEngine()
	now := time.Now().UTC()
	heartbeat := &model.NodeHeartbeat{
		CollectedAt: now.Add(-50 * time.Second),
	}
	probes := []model.ProbeObservation{
		{SourceNodeID: "node-b"},
		{SourceNodeID: "node-c"},
	}

	state := engine.assessNode("node-a", heartbeat, probes, nil, now, 3, false)
	if state.Status != model.StatusCritical {
		t.Fatalf("status = %q, want %q", state.Status, model.StatusCritical)
	}
	if state.LastProbeSummary.ExpectedPeers != 3 || state.LastProbeSummary.TotalPeers != 2 {
		t.Fatalf("summary = %+v", state.LastProbeSummary)
	}
}

func TestAssessNodeFreshHeartbeatWithSuccessfulObserverIsHealthy(t *testing.T) {
	t.Parallel()

	engine := newTestEngine()
	now := time.Now().UTC()
	heartbeat := &model.NodeHeartbeat{
		CollectedAt: now.Add(-10 * time.Second),
	}
	probes := []model.ProbeObservation{
		{SourceNodeID: "node-b", TCP443OK: true},
	}

	state := engine.assessNode("node-a", heartbeat, probes, nil, now, 3, false)
	if state.Status != model.StatusHealthy {
		t.Fatalf("status = %q, want %q", state.Status, model.StatusHealthy)
	}
	if state.Reason != "fresh heartbeat and observer reachability confirmed" {
		t.Fatalf("reason = %q", state.Reason)
	}
	if state.LastProbeSummary.ExpectedPeers != 3 || state.LastProbeSummary.SuccessfulPeers != 1 || state.LastProbeSummary.TotalPeers != 1 {
		t.Fatalf("summary = %+v", state.LastProbeSummary)
	}
}

func newTestEngine() *Engine {
	return &Engine{
		cfg: &config.Config{
			Thresholds: config.Thresholds{
				CPUWarn:  80,
				CPUCrit:  92,
				MemWarn:  85,
				MemCrit:  95,
				DiskWarn: 80,
				DiskCrit: 92,
			},
		},
	}
}
