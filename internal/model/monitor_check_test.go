package model

import "testing"

func TestMonitorCheckNormalizeDefaultsScope(t *testing.T) {
	t.Parallel()

	check := MonitorCheck{
		Type:    MonitorCheckTypeSystemd,
		Name:    "sshd",
		NodeIDs: []string{" node-b ", "node-a", "node-b", ""},
	}
	normalized := check.Normalize()

	if normalized.ScopeMode != MonitorCheckScopeAll {
		t.Fatalf("ScopeMode = %q, want %q", normalized.ScopeMode, MonitorCheckScopeAll)
	}
	if len(normalized.NodeIDs) != 2 || normalized.NodeIDs[0] != "node-a" || normalized.NodeIDs[1] != "node-b" {
		t.Fatalf("NodeIDs = %#v, want [node-a node-b]", normalized.NodeIDs)
	}
}

func TestMonitorCheckValidateScope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		check MonitorCheck
		ok    bool
	}{
		{
			name: "all scope without nodes",
			check: MonitorCheck{
				Type:        MonitorCheckTypeSystemd,
				Name:        "sshd",
				ServiceName: "ssh",
			},
			ok: true,
		},
		{
			name: "include scope requires nodes",
			check: MonitorCheck{
				Type:        MonitorCheckTypeSystemd,
				Name:        "sshd",
				ServiceName: "ssh",
				ScopeMode:   MonitorCheckScopeInclude,
			},
			ok: false,
		},
		{
			name: "exclude scope with nodes",
			check: MonitorCheck{
				Type:        MonitorCheckTypeSystemd,
				Name:        "sshd",
				ServiceName: "ssh",
				ScopeMode:   MonitorCheckScopeExclude,
				NodeIDs:     []string{"node-c"},
			},
			ok: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.check.Validate()
			if tt.ok && err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if !tt.ok && err == nil {
				t.Fatal("Validate() error = nil, want error")
			}
		})
	}
}

func TestMonitorCheckAppliesToNode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		check  MonitorCheck
		nodeID string
		want   bool
	}{
		{
			name: "all scope",
			check: MonitorCheck{
				Type:        MonitorCheckTypeSystemd,
				Name:        "sshd",
				ServiceName: "ssh",
			},
			nodeID: "node-a",
			want:   true,
		},
		{
			name: "include scope hit",
			check: MonitorCheck{
				Type:        MonitorCheckTypeSystemd,
				Name:        "sshd",
				ServiceName: "ssh",
				ScopeMode:   MonitorCheckScopeInclude,
				NodeIDs:     []string{"node-a"},
			},
			nodeID: "node-a",
			want:   true,
		},
		{
			name: "include scope miss",
			check: MonitorCheck{
				Type:        MonitorCheckTypeSystemd,
				Name:        "sshd",
				ServiceName: "ssh",
				ScopeMode:   MonitorCheckScopeInclude,
				NodeIDs:     []string{"node-a"},
			},
			nodeID: "node-b",
			want:   false,
		},
		{
			name: "exclude scope hit",
			check: MonitorCheck{
				Type:        MonitorCheckTypeSystemd,
				Name:        "sshd",
				ServiceName: "ssh",
				ScopeMode:   MonitorCheckScopeExclude,
				NodeIDs:     []string{"node-b"},
			},
			nodeID: "node-a",
			want:   true,
		},
		{
			name: "exclude scope miss",
			check: MonitorCheck{
				Type:        MonitorCheckTypeSystemd,
				Name:        "sshd",
				ServiceName: "ssh",
				ScopeMode:   MonitorCheckScopeExclude,
				NodeIDs:     []string{"node-b"},
			},
			nodeID: "node-b",
			want:   false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.check.AppliesToNode(tt.nodeID); got != tt.want {
				t.Fatalf("AppliesToNode(%q) = %v, want %v", tt.nodeID, got, tt.want)
			}
		})
	}
}
