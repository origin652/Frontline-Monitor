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

func TestValidateDynamicClusterRequiresAPIAddrTokenAndJoin(t *testing.T) {
	t.Setenv("MONITOR_INTERNAL_TOKEN", "secret")
	cfg := &Config{
		Cluster: ClusterConfig{
			NodeID:      "node-a",
			APIAddr:     "10.0.0.11:8443",
			RaftAddr:    "10.0.0.11:7000",
			JoinSeeds:   []string{"10.0.0.12:8443"},
			Role:        "voter",
			Bootstrap:   false,
			Priority:    100,
			DisplayName: "Tokyo-1",
		},
		Network: NetworkConfig{
			ListenAddr:      ":8443",
			PublicIPv4:      "203.0.113.11",
			PublicHTTPSPort: 443,
		},
		Storage: StorageConfig{
			DataDir:    "/tmp/data",
			SQLitePath: "/tmp/data/monitor.db",
			RaftDir:    "/tmp/data/raft",
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateDynamicClusterRejectsMissingJoinTarget(t *testing.T) {
	t.Setenv("MONITOR_INTERNAL_TOKEN", "secret")
	cfg := &Config{
		Cluster: ClusterConfig{
			NodeID:   "node-a",
			APIAddr:  "10.0.0.11:8443",
			RaftAddr: "10.0.0.11:7000",
			Role:     "voter",
		},
		Network: NetworkConfig{
			ListenAddr:      ":8443",
			PublicIPv4:      "203.0.113.11",
			PublicHTTPSPort: 443,
		},
		Storage: StorageConfig{
			DataDir:    "/tmp/data",
			SQLitePath: "/tmp/data/monitor.db",
			RaftDir:    "/tmp/data/raft",
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want join validation failure")
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
