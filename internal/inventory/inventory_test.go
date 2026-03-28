package inventory

import (
	"testing"

	"vps-monitor/internal/config"
)

func TestRenderNodeMergesSharedSettingsAndOverrides(t *testing.T) {
	t.Parallel()

	falseValue := false
	inv := &Inventory{
		Cluster: InventoryCluster{
			InternalTokenEnv: "MONITOR_INTERNAL_TOKEN",
		},
		Shared: SharedConfig{
			Network: config.NetworkConfig{
				ListenAddr:      ":8443",
				PublicHTTPSPort: 443,
				TLSCertFile:     "/etc/ssl/shared.pem",
				TLSKeyFile:      "/etc/ssl/shared.key",
			},
			Checks: config.ChecksConfig{
				Services: []string{"ssh", "nginx"},
				TCPPorts: []int{80, 443},
				HTTPChecks: []config.HTTPCheck{
					{Name: "health", Scheme: "https", Path: "/healthz", Port: 443, ExpectStatus: 200, Timeout: "3s"},
				},
				DockerChecks: []string{"app"},
			},
			Thresholds: config.Thresholds{
				CPUWarn:  80,
				CPUCrit:  92,
				MemWarn:  85,
				MemCrit:  95,
				DiskWarn: 80,
				DiskCrit: 92,
			},
			Storage: config.StorageConfig{
				DataDir:       "/var/lib/vps-monitor",
				SQLitePath:    "/var/lib/vps-monitor/monitor.db",
				RaftDir:       "/var/lib/vps-monitor/raft",
				RetentionDays: 30,
			},
		},
		Nodes: []Node{
			{
				NodeID:      "node-a",
				DisplayName: "Tokyo-1",
				APIAddr:     "10.0.0.11:8443",
				RaftAddr:    "10.0.0.11:7000",
				PublicIPv4:  "203.0.113.11",
				Priority:    300,
			},
			{
				NodeID:           "node-c",
				DisplayName:      "Frankfurt-1",
				APIAddr:          "10.0.0.13:8443",
				RaftAddr:         "10.0.0.13:7000",
				RaftBindAddr:     "0.0.0.0:7000",
				PublicIPv4:       "203.0.113.13",
				Priority:         100,
				IngressCandidate: &falseValue,
				Network: &config.NetworkConfig{
					ListenAddr: ":9443",
				},
				Checks: &config.ChecksConfig{
					Services: []string{"ssh"},
				},
				Storage: &config.StorageConfig{
					DataDir: "/srv/vps-monitor",
				},
			},
		},
	}

	cfg, err := inv.RenderNode("node-c")
	if err != nil {
		t.Fatalf("RenderNode() error = %v", err)
	}

	if cfg.Cluster.NodeID != "node-c" {
		t.Fatalf("Cluster.NodeID = %q, want %q", cfg.Cluster.NodeID, "node-c")
	}
	if cfg.Cluster.RaftBindAddr != "0.0.0.0:7000" {
		t.Fatalf("Cluster.RaftBindAddr = %q, want %q", cfg.Cluster.RaftBindAddr, "0.0.0.0:7000")
	}
	if cfg.Cluster.InternalTokenEnv != "MONITOR_INTERNAL_TOKEN" {
		t.Fatalf("Cluster.InternalTokenEnv = %q, want %q", cfg.Cluster.InternalTokenEnv, "MONITOR_INTERNAL_TOKEN")
	}
	if len(cfg.Cluster.Peers) != 2 {
		t.Fatalf("len(Cluster.Peers) = %d, want 2", len(cfg.Cluster.Peers))
	}
	if got := cfg.Cluster.Peers[1].DisplayName; got != "Frankfurt-1" {
		t.Fatalf("peer display_name = %q, want %q", got, "Frankfurt-1")
	}
	if got := cfg.Cluster.Peers[1].IsIngressCandidate(); got {
		t.Fatalf("peer ingress candidate = %v, want false", got)
	}
	if cfg.Network.ListenAddr != ":9443" {
		t.Fatalf("Network.ListenAddr = %q, want %q", cfg.Network.ListenAddr, ":9443")
	}
	if cfg.Network.PublicIPv4 != "203.0.113.13" {
		t.Fatalf("Network.PublicIPv4 = %q, want %q", cfg.Network.PublicIPv4, "203.0.113.13")
	}
	if cfg.Network.TLSCertFile != "/etc/ssl/shared.pem" {
		t.Fatalf("Network.TLSCertFile = %q, want %q", cfg.Network.TLSCertFile, "/etc/ssl/shared.pem")
	}
	if len(cfg.Checks.Services) != 1 || cfg.Checks.Services[0] != "ssh" {
		t.Fatalf("Checks.Services = %#v, want [ssh]", cfg.Checks.Services)
	}
	if len(cfg.Checks.TCPPorts) != 2 {
		t.Fatalf("Checks.TCPPorts = %#v, want shared ports", cfg.Checks.TCPPorts)
	}
	if cfg.Storage.DataDir != "/srv/vps-monitor" {
		t.Fatalf("Storage.DataDir = %q, want %q", cfg.Storage.DataDir, "/srv/vps-monitor")
	}
	if cfg.Storage.SQLitePath != "/var/lib/vps-monitor/monitor.db" {
		t.Fatalf("Storage.SQLitePath = %q, want shared path", cfg.Storage.SQLitePath)
	}
}

func TestRenderNodeAppliesInventoryDefaults(t *testing.T) {
	t.Parallel()

	inv := &Inventory{
		Nodes: []Node{
			{
				NodeID:     "node-a",
				APIAddr:    "10.0.0.11:8443",
				RaftAddr:   "10.0.0.11:7000",
				PublicIPv4: "203.0.113.11",
			},
		},
	}
	*inv = *defaultInventory()
	inv.Nodes = []Node{
		{
			NodeID:     "node-a",
			APIAddr:    "10.0.0.11:8443",
			RaftAddr:   "10.0.0.11:7000",
			PublicIPv4: "203.0.113.11",
		},
	}

	cfg, err := inv.RenderNode("node-a")
	if err != nil {
		t.Fatalf("RenderNode() error = %v", err)
	}

	if cfg.Network.ListenAddr != ":8443" {
		t.Fatalf("Network.ListenAddr = %q, want %q", cfg.Network.ListenAddr, ":8443")
	}
	if cfg.Network.PublicHTTPSPort != 443 {
		t.Fatalf("Network.PublicHTTPSPort = %d, want 443", cfg.Network.PublicHTTPSPort)
	}
	if cfg.Storage.SQLitePath != "/var/lib/vps-monitor/monitor.db" {
		t.Fatalf("Storage.SQLitePath = %q, want default path", cfg.Storage.SQLitePath)
	}
}

func TestRenderNodeDynamicMembership(t *testing.T) {
	t.Parallel()

	falseValue := false
	inv := &Inventory{
		Cluster: InventoryCluster{
			Mode:             "dynamic",
			InternalTokenEnv: "MONITOR_INTERNAL_TOKEN",
		},
		Shared: defaultInventory().Shared,
		Nodes: []Node{
			{
				NodeID:      "node-a",
				DisplayName: "Tokyo-1",
				APIAddr:     "10.0.0.11:8443",
				RaftAddr:    "10.0.0.11:7000",
				PublicIPv4:  "203.0.113.11",
				Priority:    300,
			},
			{
				NodeID:           "node-b",
				DisplayName:      "Singapore-1",
				APIAddr:          "10.0.0.12:8443",
				RaftAddr:         "10.0.0.12:7000",
				PublicIPv4:       "203.0.113.12",
				Priority:         200,
				Role:             "nonvoter",
				IngressCandidate: &falseValue,
			},
		},
	}

	bootstrapCfg, err := inv.RenderNode("node-a")
	if err != nil {
		t.Fatalf("RenderNode(node-a) error = %v", err)
	}
	if !bootstrapCfg.UsesDynamicMembership() {
		t.Fatal("bootstrap config should use dynamic membership")
	}
	if !bootstrapCfg.Cluster.Bootstrap {
		t.Fatal("bootstrap node should set cluster.bootstrap=true")
	}
	if len(bootstrapCfg.Cluster.Peers) != 0 {
		t.Fatalf("bootstrap config peers = %#v, want none", bootstrapCfg.Cluster.Peers)
	}
	if bootstrapCfg.Cluster.APIAddr != "10.0.0.11:8443" {
		t.Fatalf("bootstrap APIAddr = %q, want %q", bootstrapCfg.Cluster.APIAddr, "10.0.0.11:8443")
	}

	joinCfg, err := inv.RenderNode("node-b")
	if err != nil {
		t.Fatalf("RenderNode(node-b) error = %v", err)
	}
	if joinCfg.Cluster.Bootstrap {
		t.Fatal("joiner should not set cluster.bootstrap=true")
	}
	if got := joinCfg.NormalizedRole(); got != "nonvoter" {
		t.Fatalf("joiner NormalizedRole() = %q, want %q", got, "nonvoter")
	}
	if got := joinCfg.Cluster.DisplayName; got != "Singapore-1" {
		t.Fatalf("joiner display_name = %q, want %q", got, "Singapore-1")
	}
	if got := joinCfg.Cluster.IngressCandidate; got == nil || *got != false {
		t.Fatalf("joiner ingress_candidate = %#v, want false", got)
	}
	if len(joinCfg.Cluster.JoinSeeds) != 1 || joinCfg.Cluster.JoinSeeds[0] != "10.0.0.11:8443" {
		t.Fatalf("joiner join_seeds = %#v, want [10.0.0.11:8443]", joinCfg.Cluster.JoinSeeds)
	}
}
