package monitor

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"vps-monitor/internal/cluster"
	"vps-monitor/internal/config"
	"vps-monitor/internal/model"
)

type Prober struct {
	cfg     *config.Config
	cluster *cluster.Manager
	source  ProbeRuntimeSource
	sink    ObservationSink
	logger  *slog.Logger
	client  *http.Client
}

type ProbeRuntimeSource interface {
	MonitorCheckSource
	ListNodeStates(ctx context.Context) ([]model.NodeState, error)
}

func NewProber(cfg *config.Config, clusterManager *cluster.Manager, source ProbeRuntimeSource, sink ObservationSink, logger *slog.Logger) *Prober {
	return &Prober{
		cfg:     cfg,
		cluster: clusterManager,
		source:  source,
		sink:    sink,
		logger:  logger,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (p *Prober) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	p.probeOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.probeOnce(ctx)
		}
	}
}

func (p *Prober) probeOnce(ctx context.Context) {
	members, err := p.cluster.ActiveMembers(ctx)
	if err != nil {
		p.logger.Error("list cluster members for prober failed", "error", err)
		return
	}

	states := []model.NodeState{}
	if p.source != nil {
		states, err = p.source.ListNodeStates(ctx)
		if err != nil {
			p.logger.Warn("load node states for prober failed, falling back to empty observer snapshot", "error", err)
		}
	}

	assignments := BuildObserverAssignments(members, states, p.cfg.ProbeObserversPerTarget())
	targets := ProbeTargetsForObserver(p.cfg.Cluster.NodeID, members, assignments)
	if len(targets) == 0 {
		p.logger.Debug("skip prober tick", "reason", errNoPeers)
		return
	}
	runtimeChecks := loadMonitorChecks(ctx, p.source, p.cfg, p.logger)

	for _, target := range targets {
		probe := p.probePeer(ctx, target, runtimeChecks)
		if err := p.sink.SubmitProbe(ctx, probe); err != nil {
			p.logger.Error("submit probe failed", "target", target.NodeID, "error", err)
		}
	}
}

func (p *Prober) probePeer(ctx context.Context, peer model.ClusterMember, checks []model.MonitorCheck) model.ProbeObservation {
	now := time.Now().UTC()
	ports := make([]model.PortResult, 0, len(checks)+2)
	sshLatency, sshOK := p.readSSHBanner(peer.PublicIPv4)
	port22 := p.probeTCP(peer.PublicIPv4, 22)
	port443 := p.probeTCP(peer.PublicIPv4, p.cfg.Network.PublicHTTPSPort)
	ports = append(ports, port22, port443)
	for _, check := range checks {
		if !check.Enabled || !check.RunsAgainstPeer() || !check.AppliesToNode(peer.NodeID) {
			continue
		}
		if check.Type == model.MonitorCheckTypeTCP {
			if check.Port == 22 || check.Port == p.cfg.Network.PublicHTTPSPort {
				continue
			}
			ports = append(ports, p.probeTCP(peer.PublicIPv4, check.Port))
		}
	}

	httpChecks := make([]model.HTTPCheckResult, 0, len(checks))
	httpOK := false
	for _, check := range checks {
		if !check.Enabled || check.Type != model.MonitorCheckTypeHTTP || !check.RunsAgainstPeer() || !check.AppliesToNode(peer.NodeID) {
			continue
		}
		result := p.probeHTTP(ctx, peer.PublicIPv4, check)
		httpChecks = append(httpChecks, result)
		if result.OK {
			httpOK = true
		}
	}

	return model.ProbeObservation{
		SourceNodeID: p.cfg.Cluster.NodeID,
		TargetNodeID: peer.NodeID,
		CollectedAt:  now,
		TCP22OK:      sshOK || port22.Open,
		TCP443OK:     port443.Open,
		HTTPOK:       httpOK,
		SSHBannerMS:  sshLatency,
		Ports:        ports,
		HTTPChecks:   httpChecks,
	}
}

func (p *Prober) probeTCP(host string, port int) model.PortResult {
	address := fmt.Sprintf("%s:%d", host, port)
	start := time.Now()
	conn, err := net.DialTimeout("tcp", address, 3*time.Second)
	if err != nil {
		return model.PortResult{
			Port:      port,
			Open:      false,
			LatencyMS: time.Since(start).Milliseconds(),
			CheckedAt: time.Now().UTC(),
		}
	}
	_ = conn.Close()
	return model.PortResult{
		Port:      port,
		Open:      true,
		LatencyMS: time.Since(start).Milliseconds(),
		CheckedAt: time.Now().UTC(),
	}
}

func (p *Prober) readSSHBanner(host string) (int64, bool) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, "22"), 3*time.Second)
	if err != nil {
		return time.Since(start).Milliseconds(), false
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return time.Since(start).Milliseconds(), false
	}
	return time.Since(start).Milliseconds(), len(line) > 0
}

func (p *Prober) probeHTTP(ctx context.Context, host string, check model.MonitorCheck) model.HTTPCheckResult {
	targetURL := fmt.Sprintf("%s://%s:%d%s", defaultScheme(check.Scheme), host, check.Port, check.Path)
	reqCtx, cancel := context.WithTimeout(ctx, p.cfg.HTTPCheckTimeout(config.HTTPCheck{
		Scheme:       check.Scheme,
		Path:         check.Path,
		Port:         check.Port,
		ExpectStatus: check.ExpectStatus,
		Timeout:      check.Timeout,
	}))
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, targetURL, nil)
	if err != nil {
		return model.HTTPCheckResult{Name: check.Name, URL: targetURL, CheckedAt: time.Now().UTC()}
	}
	start := time.Now()
	resp, err := p.client.Do(req)
	if err != nil {
		return model.HTTPCheckResult{
			Name:      check.Name,
			URL:       targetURL,
			OK:        false,
			LatencyMS: time.Since(start).Milliseconds(),
			CheckedAt: time.Now().UTC(),
		}
	}
	defer resp.Body.Close()
	return model.HTTPCheckResult{
		Name:       check.Name,
		URL:        targetURL,
		OK:         resp.StatusCode == expectedStatus(check.ExpectStatus),
		StatusCode: resp.StatusCode,
		LatencyMS:  time.Since(start).Milliseconds(),
		CheckedAt:  time.Now().UTC(),
	}
}
