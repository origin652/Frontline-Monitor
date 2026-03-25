package model

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

const (
	MonitorCheckTypeSystemd = "systemd"
	MonitorCheckTypeDocker  = "docker"
	MonitorCheckTypeHTTP    = "http"
	MonitorCheckTypeTCP     = "tcp"

	MonitorCheckHostModeLocal = "local"
	MonitorCheckHostModePeer  = "peer"
	MonitorCheckScopeAll      = "all"
	MonitorCheckScopeInclude  = "include_nodes"
	MonitorCheckScopeExclude  = "exclude_nodes"
)

type MonitorCheck struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"`
	Name          string    `json:"name"`
	Enabled       bool      `json:"enabled"`
	SortOrder     int       `json:"sort_order"`
	ScopeMode     string    `json:"scope_mode,omitempty"`
	NodeIDs       []string  `json:"node_ids,omitempty"`
	ServiceName   string    `json:"service_name,omitempty"`
	ContainerName string    `json:"container_name,omitempty"`
	Scheme        string    `json:"scheme,omitempty"`
	HostMode      string    `json:"host_mode,omitempty"`
	Port          int       `json:"port,omitempty"`
	Path          string    `json:"path,omitempty"`
	ExpectStatus  int       `json:"expect_status,omitempty"`
	Timeout       string    `json:"timeout,omitempty"`
	Label         string    `json:"label,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (c MonitorCheck) Normalize() MonitorCheck {
	c.Type = strings.TrimSpace(strings.ToLower(c.Type))
	c.Name = strings.TrimSpace(c.Name)
	c.ServiceName = strings.TrimSpace(c.ServiceName)
	c.ContainerName = strings.TrimSpace(c.ContainerName)
	c.Scheme = strings.TrimSpace(strings.ToLower(c.Scheme))
	c.HostMode = strings.TrimSpace(strings.ToLower(c.HostMode))
	c.ScopeMode = strings.TrimSpace(strings.ToLower(c.ScopeMode))
	c.Path = strings.TrimSpace(c.Path)
	c.Timeout = strings.TrimSpace(c.Timeout)
	c.Label = strings.TrimSpace(c.Label)
	c.NodeIDs = normalizeMonitorCheckNodeIDs(c.NodeIDs)
	if c.Scheme == "" {
		c.Scheme = "http"
	}
	if c.Type == MonitorCheckTypeHTTP && c.HostMode == "" {
		c.HostMode = MonitorCheckHostModePeer
	}
	if c.ScopeMode == "" {
		c.ScopeMode = MonitorCheckScopeAll
	}
	if c.Path == "" && c.Type == MonitorCheckTypeHTTP {
		c.Path = "/"
	}
	return c
}

func (c MonitorCheck) Validate() error {
	c = c.Normalize()
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	switch c.ScopeMode {
	case MonitorCheckScopeAll:
	case MonitorCheckScopeInclude, MonitorCheckScopeExclude:
		if len(c.NodeIDs) == 0 {
			return fmt.Errorf("node_ids is required when scope_mode is %s", c.ScopeMode)
		}
	default:
		return fmt.Errorf("unsupported scope_mode %q", c.ScopeMode)
	}
	switch c.Type {
	case MonitorCheckTypeSystemd:
		if c.ServiceName == "" {
			return fmt.Errorf("service_name is required")
		}
	case MonitorCheckTypeDocker:
		if c.ContainerName == "" {
			return fmt.Errorf("container_name is required")
		}
	case MonitorCheckTypeHTTP:
		if c.HostMode != MonitorCheckHostModeLocal && c.HostMode != MonitorCheckHostModePeer {
			return fmt.Errorf("host_mode must be local or peer")
		}
		if c.Port <= 0 || c.Port > 65535 {
			return fmt.Errorf("port must be between 1 and 65535")
		}
	case MonitorCheckTypeTCP:
		if c.Port <= 0 || c.Port > 65535 {
			return fmt.Errorf("port must be between 1 and 65535")
		}
	default:
		return fmt.Errorf("unsupported check type %q", c.Type)
	}
	return nil
}

func (c MonitorCheck) RunsLocally() bool {
	c = c.Normalize()
	return c.Type == MonitorCheckTypeSystemd ||
		c.Type == MonitorCheckTypeDocker ||
		(c.Type == MonitorCheckTypeHTTP && c.HostMode == MonitorCheckHostModeLocal)
}

func (c MonitorCheck) RunsAgainstPeer() bool {
	c = c.Normalize()
	return c.Type == MonitorCheckTypeTCP ||
		(c.Type == MonitorCheckTypeHTTP && c.HostMode == MonitorCheckHostModePeer)
}

func (c MonitorCheck) AppliesToNode(nodeID string) bool {
	c = c.Normalize()
	nodeID = strings.TrimSpace(nodeID)
	switch c.ScopeMode {
	case MonitorCheckScopeInclude:
		return slices.Contains(c.NodeIDs, nodeID)
	case MonitorCheckScopeExclude:
		return !slices.Contains(c.NodeIDs, nodeID)
	default:
		return true
	}
}

func (c MonitorCheck) TargetLabel() string {
	c = c.Normalize()
	switch c.Type {
	case MonitorCheckTypeSystemd:
		return c.ServiceName
	case MonitorCheckTypeDocker:
		return c.ContainerName
	case MonitorCheckTypeHTTP:
		return fmt.Sprintf("%s:%d%s", c.Scheme, c.Port, c.Path)
	case MonitorCheckTypeTCP:
		if c.Label != "" {
			return fmt.Sprintf("%s:%d", c.Label, c.Port)
		}
		return fmt.Sprintf(":%d", c.Port)
	default:
		return ""
	}
}

func normalizeMonitorCheckNodeIDs(nodeIDs []string) []string {
	if len(nodeIDs) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(nodeIDs))
	normalized := make([]string, 0, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		trimmed := strings.TrimSpace(nodeID)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	slices.Sort(normalized)
	return normalized
}
