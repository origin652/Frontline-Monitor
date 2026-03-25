package model

import (
	"fmt"
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
)

type MonitorCheck struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"`
	Name          string    `json:"name"`
	Enabled       bool      `json:"enabled"`
	SortOrder     int       `json:"sort_order"`
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
	c.Path = strings.TrimSpace(c.Path)
	c.Timeout = strings.TrimSpace(c.Timeout)
	c.Label = strings.TrimSpace(c.Label)
	if c.Scheme == "" {
		c.Scheme = "http"
	}
	if c.Type == MonitorCheckTypeHTTP && c.HostMode == "" {
		c.HostMode = MonitorCheckHostModePeer
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
