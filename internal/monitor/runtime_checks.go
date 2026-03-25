package monitor

import (
	"context"
	"log/slog"

	"vps-monitor/internal/config"
	"vps-monitor/internal/model"
)

type MonitorCheckSource interface {
	ListMonitorChecks(ctx context.Context) ([]model.MonitorCheck, error)
}

func loadMonitorChecks(ctx context.Context, source MonitorCheckSource, cfg *config.Config, logger *slog.Logger) []model.MonitorCheck {
	if source != nil {
		checks, err := source.ListMonitorChecks(ctx)
		if err == nil && len(checks) > 0 {
			return checks
		}
		if err != nil && logger != nil {
			logger.Warn("load runtime monitor checks failed, falling back to config", "error", err)
		}
	}
	return config.RuntimeMonitorChecksFromConfig(cfg)
}
