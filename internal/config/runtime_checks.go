package config

import (
	"fmt"

	"vps-monitor/internal/model"
)

func RuntimeMonitorChecksFromConfig(cfg *Config) []model.MonitorCheck {
	if cfg == nil {
		return nil
	}

	checks := make([]model.MonitorCheck, 0, len(cfg.Checks.Services)+len(cfg.Checks.DockerChecks)+len(cfg.Checks.HTTPChecks)+len(cfg.Checks.TCPPorts))
	sortOrder := 10

	for i, service := range cfg.Checks.Services {
		checks = append(checks, model.MonitorCheck{
			ID:          fmt.Sprintf("seed-systemd-%d", i),
			Type:        model.MonitorCheckTypeSystemd,
			Name:        service,
			Enabled:     true,
			SortOrder:   sortOrder,
			ServiceName: service,
		})
		sortOrder += 10
	}

	for i, container := range cfg.Checks.DockerChecks {
		checks = append(checks, model.MonitorCheck{
			ID:            fmt.Sprintf("seed-docker-%d", i),
			Type:          model.MonitorCheckTypeDocker,
			Name:          container,
			Enabled:       true,
			SortOrder:     sortOrder,
			ContainerName: container,
		})
		sortOrder += 10
	}

	for i, check := range cfg.Checks.HTTPChecks {
		name := check.Name
		if name == "" {
			name = fmt.Sprintf("http-%d", i+1)
		}
		checks = append(checks, model.MonitorCheck{
			ID:           fmt.Sprintf("seed-http-%d", i),
			Type:         model.MonitorCheckTypeHTTP,
			Name:         name,
			Enabled:      true,
			SortOrder:    sortOrder,
			Scheme:       check.Scheme,
			HostMode:     model.MonitorCheckHostModePeer,
			Port:         check.Port,
			Path:         check.Path,
			ExpectStatus: check.ExpectStatus,
			Timeout:      check.Timeout,
		}.Normalize())
		sortOrder += 10
	}

	for i, port := range cfg.Checks.TCPPorts {
		checks = append(checks, model.MonitorCheck{
			ID:        fmt.Sprintf("seed-tcp-%d", i),
			Type:      model.MonitorCheckTypeTCP,
			Name:      fmt.Sprintf("tcp-%d", port),
			Enabled:   true,
			SortOrder: sortOrder,
			Port:      port,
			Label:     fmt.Sprintf("tcp-%d", port),
		}.Normalize())
		sortOrder += 10
	}

	return checks
}
