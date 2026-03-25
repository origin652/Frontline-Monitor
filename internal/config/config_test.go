package config

import "testing"

func TestRaftBindAddrFallback(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Cluster: ClusterConfig{
			RaftAddr: "203.0.113.10:7000",
		},
	}
	if got := cfg.RaftBindAddr(); got != "203.0.113.10:7000" {
		t.Fatalf("RaftBindAddr() fallback = %q, want %q", got, "203.0.113.10:7000")
	}

	cfg.Cluster.RaftBindAddr = "0.0.0.0:7000"
	if got := cfg.RaftBindAddr(); got != "0.0.0.0:7000" {
		t.Fatalf("RaftBindAddr() explicit = %q, want %q", got, "0.0.0.0:7000")
	}
}

func TestClusterPeerIsIngressCandidate(t *testing.T) {
	t.Parallel()

	falseValue := false
	trueValue := true
	tests := []struct {
		name string
		peer ClusterPeer
		want bool
	}{
		{
			name: "default true when omitted",
			peer: ClusterPeer{},
			want: true,
		},
		{
			name: "explicit false",
			peer: ClusterPeer{IngressCandidate: &falseValue},
			want: false,
		},
		{
			name: "explicit true",
			peer: ClusterPeer{IngressCandidate: &trueValue},
			want: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.peer.IsIngressCandidate(); got != tt.want {
				t.Fatalf("IsIngressCandidate() = %v, want %v", got, tt.want)
			}
		})
	}
}
