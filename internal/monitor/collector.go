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
	sink   ObservationSink
	logger *slog.Logger
	client *http.Client
}

func NewCollector(cfg *config.Config, sink ObservationSink, logger *slog.Logger) *Collector {
	return &Collector{
		cfg:    cfg,
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

	services := make([]model.ServiceCheck, 0, len(c.cfg.Checks.Services))
	for _, service := range c.cfg.Checks.Services {
		services = append(services, c.checkService(ctx, service))
	}

	dockerChecks := make([]model.DockerCheck, 0, len(c.cfg.Checks.DockerChecks))
	for _, container := range c.cfg.Checks.DockerChecks {
		dockerChecks = append(dockerChecks, c.checkDocker(ctx, container))
	}

	httpChecks := make([]model.HTTPCheckResult, 0, len(c.cfg.Checks.HTTPChecks))
	for _, check := range c.cfg.Checks.HTTPChecks {
		httpChecks = append(httpChecks, c.runLocalHTTPCheck(ctx, check))
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

func (c *Collector) checkService(ctx context.Context, service string) model.ServiceCheck {
	now := time.Now().UTC()
	command := exec.CommandContext(ctx, "systemctl", "is-active", service)
	output, err := command.CombinedOutput()
	if err != nil {
		return model.ServiceCheck{
			Name:      service,
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
		Name:      service,
		Status:    state,
		Detail:    state,
		UpdatedAt: now,
	}
}

func (c *Collector) checkDocker(ctx context.Context, container string) model.DockerCheck {
	now := time.Now().UTC()
	command := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.Status}}", container)
	output, err := command.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(output))
		if detail == "" {
			detail = err.Error()
		}
		return model.DockerCheck{
			Name:      container,
			Status:    "unknown",
			Detail:    detail,
			UpdatedAt: now,
		}
	}
	state := strings.TrimSpace(string(output))
	if state == "" {
		state = "unknown"
	}
	return model.DockerCheck{
		Name:      container,
		Status:    state,
		Detail:    state,
		UpdatedAt: now,
	}
}

func (c *Collector) runLocalHTTPCheck(ctx context.Context, check config.HTTPCheck) model.HTTPCheckResult {
	now := time.Now().UTC()
	targetURL := fmt.Sprintf("%s://127.0.0.1:%d%s", defaultScheme(check.Scheme), check.Port, check.Path)
	checkCtx, cancel := context.WithTimeout(ctx, c.cfg.HTTPCheckTimeout(check))
	defer cancel()

	req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, targetURL, nil)
	if err != nil {
		return model.HTTPCheckResult{Name: check.Name, URL: targetURL, CheckedAt: now}
	}
	start := time.Now()
	resp, err := c.client.Do(req)
	if err != nil {
		return model.HTTPCheckResult{
			Name:      check.Name,
			URL:       targetURL,
			OK:        false,
			CheckedAt: now,
		}
	}
	defer resp.Body.Close()
	return model.HTTPCheckResult{
		Name:       check.Name,
		URL:        targetURL,
		OK:         resp.StatusCode == expectedStatus(check.ExpectStatus),
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
