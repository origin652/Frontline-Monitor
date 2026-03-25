package engine

import (
	"testing"

	"vps-monitor/internal/config"
	"vps-monitor/internal/model"
)

func TestSelectIngressTargetPeerSkipsNonCandidates(t *testing.T) {
	t.Parallel()

	falseValue := false
	cfg := &config.Config{
		Cluster: config.ClusterConfig{
			NodeID: "node-a",
			Peers: []config.ClusterPeer{
				{
					NodeID:     "node-a",
					RaftAddr:   "10.0.0.11:7000",
					APIAddr:    "10.0.0.11:8443",
					PublicIPv4: "203.0.113.11",
					Priority:   100,
				},
				{
					NodeID:     "node-b",
					RaftAddr:   "10.0.0.12:7000",
					APIAddr:    "10.0.0.12:8443",
					PublicIPv4: "203.0.113.12",
					Priority:   200,
				},
				{
					NodeID:           "node-c",
					RaftAddr:         "10.0.0.13:7000",
					APIAddr:          "10.0.0.13:8443",
					PublicIPv4:       "203.0.113.13",
					Priority:         300,
					IngressCandidate: &falseValue,
				},
			},
		},
	}

	states := []model.NodeState{
		{
			NodeID:  "node-a",
			Status:  model.StatusHealthy,
			RuleKey: "telemetry",
			LastProbeSummary: model.ProbeSummary{
				Reachable: false,
			},
		},
		{
			NodeID:  "node-b",
			Status:  model.StatusHealthy,
			RuleKey: "telemetry",
			LastProbeSummary: model.ProbeSummary{
				Reachable: true,
			},
		},
		{
			NodeID:  "node-c",
			Status:  model.StatusHealthy,
			RuleKey: "telemetry",
			LastProbeSummary: model.ProbeSummary{
				Reachable: true,
			},
		},
	}

	target, ok := selectIngressTargetPeer(cfg, states, nil)
	if !ok {
		t.Fatal("selectIngressTargetPeer() = no match, want node-b")
	}
	if target.NodeID != "node-b" {
		t.Fatalf("selectIngressTargetPeer() = %q, want %q", target.NodeID, "node-b")
	}
}
