package monitor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"

	"vps-monitor/internal/config"
	"vps-monitor/internal/model"
)

type ObservationSink interface {
	SubmitHeartbeat(ctx context.Context, hb model.NodeHeartbeat) error
	SubmitProbe(ctx context.Context, probe model.ProbeObservation) error
}

type Collector struct {
	cfg    *config.Config
	checks MonitorCheckSource
	sink   ObservationSink
	logger *slog.Logger
	client *http.Client
}

func NewCollector(cfg *config.Config, checks MonitorCheckSource, sink ObservationSink, logger *slog.Logger) *Collector {
	return &Collector{
		cfg:    cfg,
		checks: checks,
		sink:   sink,
		logger: logger,
		client: &http.Client{Timeout: 4 * time.Second},
	}
}

func (c *Collector) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	c.collectOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.collectOnce(ctx)
		}
	}
}

func (c *Collector) collectOnce(ctx context.Context) {
	hb, err := c.collect(ctx)
	if err != nil {
		c.logger.Error("collector tick failed", "error", err)
		return
	}
	if err := c.sink.SubmitHeartbeat(ctx, hb); err != nil {
		c.logger.Error("submit heartbeat failed", "error", err)
	}
}

func (c *Collector) collect(ctx context.Context) (model.NodeHeartbeat, error) {
	percentages, err := cpu.Percent(0, false)
	if err != nil {
		return model.NodeHeartbeat{}, fmt.Errorf("cpu percent: %w", err)
	}
	vm, err := mem.VirtualMemory()
	if err != nil {
		return model.NodeHeartbeat{}, fmt.Errorf("virtual memory: %w", err)
	}
	usage, err := disk.Usage(diskUsagePath())
	if err != nil {
		return model.NodeHeartbeat{}, fmt.Errorf("disk usage: %w", err)
	}
	load1 := 0.0
	loadAvg, err := load.Avg()
	if err == nil {
		load1 = loadAvg.Load1
	}
	hostInfo, err := host.InfoWithContext(ctx)
	if err != nil {
		return model.NodeHeartbeat{}, fmt.Errorf("host info: %w", err)
	}

	runtimeChecks := loadMonitorChecks(ctx, c.checks, c.cfg, c.logger)
	services := make([]model.ServiceCheck, 0, len(runtimeChecks))
	var dockerChecks []model.DockerCheck
	var httpChecks []model.HTTPCheckResult
	for _, check := range runtimeChecks {
		if !check.Enabled || !check.RunsLocally() {
			continue
		}
		switch check.Type {
		case model.MonitorCheckTypeSystemd:
			services = append(services, c.checkService(ctx, check))
		case model.MonitorCheckTypeDocker:
			service, docker := c.checkDocker(ctx, check)
			services = append(services, service)
			dockerChecks = append(dockerChecks, docker)
		case model.MonitorCheckTypeHTTP:
			service, httpResult := c.runLocalHTTPCheck(ctx, check)
			services = append(services, service)
			httpChecks = append(httpChecks, httpResult)
		}
	}

	return model.NodeHeartbeat{
		NodeID:          c.cfg.Cluster.NodeID,
		CollectedAt:     time.Now().UTC(),
		CPUPct:          firstOrZero(percentages),
		MemPct:          vm.UsedPercent,
		DiskPct:         usage.UsedPercent,
		Load1:           load1,
		UptimeS:         hostInfo.Uptime,
		Services:        services,
		DockerChecks:    dockerChecks,
		LocalHTTPChecks: httpChecks,
	}, nil
}

func (c *Collector) checkService(ctx context.Context, check model.MonitorCheck) model.ServiceCheck {
	now := time.Now().UTC()
	command := exec.CommandContext(ctx, "systemctl", "is-active", check.ServiceName)
	output, err := command.CombinedOutput()
	if err != nil {
		return model.ServiceCheck{
			ID:        check.ID,
			Type:      check.Type,
			Name:      check.Name,
			Target:    check.ServiceName,
			Status:    "inactive",
			Detail:    strings.TrimSpace(string(output)),
			UpdatedAt: now,
		}
	}
	state := strings.TrimSpace(string(output))
	if state == "" {
		state = "unknown"
	}
	return model.ServiceCheck{
		ID:        check.ID,
		Type:      check.Type,
		Name:      check.Name,
		Target:    check.ServiceName,
		Status:    state,
		Detail:    state,
		UpdatedAt: now,
	}
}

func (c *Collector) checkDocker(ctx context.Context, check model.MonitorCheck) (model.ServiceCheck, model.DockerCheck) {
	now := time.Now().UTC()
	command := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.Status}}", check.ContainerName)
	output, err := command.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(output))
		if detail == "" {
			detail = err.Error()
		}
		return model.ServiceCheck{
				ID:        check.ID,
				Type:      check.Type,
				Name:      check.Name,
				Target:    check.ContainerName,
				Status:    "unknown",
				Detail:    detail,
				UpdatedAt: now,
			}, model.DockerCheck{
				Name:      check.ContainerName,
				Status:    "unknown",
				Detail:    detail,
				UpdatedAt: now,
			}
	}
	state := strings.TrimSpace(string(output))
	if state == "" {
		state = "unknown"
	}
	return model.ServiceCheck{
			ID:        check.ID,
			Type:      check.Type,
			Name:      check.Name,
			Target:    check.ContainerName,
			Status:    state,
			Detail:    state,
			UpdatedAt: now,
		}, model.DockerCheck{
			Name:      check.ContainerName,
			Status:    state,
			Detail:    state,
			UpdatedAt: now,
		}
}

func (c *Collector) runLocalHTTPCheck(ctx context.Context, check model.MonitorCheck) (model.ServiceCheck, model.HTTPCheckResult) {
	now := time.Now().UTC()
	targetURL := fmt.Sprintf("%s://127.0.0.1:%d%s", defaultScheme(check.Scheme), check.Port, check.Path)
	checkCtx, cancel := context.WithTimeout(ctx, c.cfg.HTTPCheckTimeout(config.HTTPCheck{
		Scheme:       check.Scheme,
		Path:         check.Path,
		Port:         check.Port,
		ExpectStatus: check.ExpectStatus,
		Timeout:      check.Timeout,
	}))
	defer cancel()

	req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, targetURL, nil)
	if err != nil {
		return model.ServiceCheck{
			ID:        check.ID,
			Type:      check.Type,
			Name:      check.Name,
			Target:    targetURL,
			Status:    "unknown",
			Detail:    err.Error(),
			UpdatedAt: now,
		}, model.HTTPCheckResult{Name: check.Name, URL: targetURL, CheckedAt: now}
	}
	start := time.Now()
	resp, err := c.client.Do(req)
	if err != nil {
		return model.ServiceCheck{
				ID:        check.ID,
				Type:      check.Type,
				Name:      check.Name,
				Target:    targetURL,
				Status:    "failed",
				Detail:    err.Error(),
				UpdatedAt: now,
			}, model.HTTPCheckResult{
				Name:      check.Name,
				URL:       targetURL,
				OK:        false,
				CheckedAt: now,
			}
	}
	defer resp.Body.Close()
	ok := resp.StatusCode == expectedStatus(check.ExpectStatus)
	status := "healthy"
	detail := fmt.Sprintf("status %d", resp.StatusCode)
	if !ok {
		status = "failed"
	}
	return model.ServiceCheck{
			ID:        check.ID,
			Type:      check.Type,
			Name:      check.Name,
			Target:    targetURL,
			Status:    status,
			Detail:    detail,
			UpdatedAt: now,
		}, model.HTTPCheckResult{
			Name:       check.Name,
			URL:        targetURL,
			OK:         ok,
			StatusCode: resp.StatusCode,
			LatencyMS:  time.Since(start).Milliseconds(),
			CheckedAt:  now,
		}
}

func firstOrZero(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	return values[0]
}

func expectedStatus(status int) int {
	if status == 0 {
		return http.StatusOK
	}
	return status
}

func defaultScheme(scheme string) string {
	if scheme == "" {
		return "http"
	}
	return scheme
}

func diskUsagePath() string {
	if runtime.GOOS == "windows" {
		return "C:\\"
	}
	return "/"
}

var errNoPeers = errors.New("no remote peers configured")
