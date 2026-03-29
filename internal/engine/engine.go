package engine

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"vps-monitor/internal/cloudflare"
	"vps-monitor/internal/cluster"
	"vps-monitor/internal/config"
	"vps-monitor/internal/model"
	"vps-monitor/internal/monitor"
	"vps-monitor/internal/notify"
	"vps-monitor/internal/store"
)

type Engine struct {
	cfg       *config.Config
	store     *store.Store
	cluster   *cluster.Manager
	cf        *cloudflare.Client
	notifiers []notify.Notifier
	logger    *slog.Logger
}

func New(cfg *config.Config, st *store.Store, cl *cluster.Manager, cf *cloudflare.Client, notifiers []notify.Notifier, logger *slog.Logger) *Engine {
	return &Engine{
		cfg:       cfg,
		store:     st,
		cluster:   cl,
		cf:        cf,
		notifiers: notifiers,
		logger:    logger,
	}
}

func (e *Engine) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	e.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.tick(ctx)
		}
	}
}

func (e *Engine) tick(ctx context.Context) {
	if !e.cluster.IsLeader() {
		return
	}
	now := time.Now().UTC()
	if err := e.cluster.EnsureMemberDirectorySeeded(ctx, now); err != nil {
		e.logger.Error("seed cluster members failed", "error", err)
	}
	if err := e.ensureMonitorChecksSeeded(ctx, now); err != nil {
		e.logger.Error("seed runtime monitor checks failed", "error", err)
	}
	members, err := e.cluster.ActiveMembers(ctx)
	if err != nil {
		e.logger.Error("list cluster members failed", "error", err)
		return
	}
	states, err := e.store.ListNodeStates(ctx)
	if err != nil {
		e.logger.Error("list node states failed", "error", err)
		return
	}
	assignments := monitor.BuildObserverAssignments(members, states, e.cfg.ProbeObserversPerTarget())
	singleNodeMode := len(members) <= 1
	for _, member := range members {
		if err := e.evaluateNode(ctx, member.NodeID, assignments[member.NodeID], singleNodeMode, now); err != nil {
			e.logger.Error("evaluate node failed", "node_id", member.NodeID, "error", err)
		}
	}
	if err := e.syncIngress(ctx, now); err != nil {
		e.logger.Error("sync ingress failed", "error", err)
	}
	if err := e.store.DeleteExpiredAdminSessions(ctx, now); err != nil {
		e.logger.Error("prune expired admin sessions failed", "error", err)
	}
	if err := e.store.PruneOldData(ctx, e.cfg.Storage.RetentionDays); err != nil {
		e.logger.Error("prune old data failed", "error", err)
	}
}

func (e *Engine) ensureMonitorChecksSeeded(ctx context.Context, now time.Time) error {
	count, err := e.store.CountMonitorChecks(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	for _, check := range config.RuntimeMonitorChecksFromConfig(e.cfg) {
		seed := check.Normalize()
		if seed.ID == "" {
			seed.ID = uuid.NewString()
		}
		seed.CreatedAt = now
		seed.UpdatedAt = now
		if err := seed.Validate(); err != nil {
			return err
		}
		if _, err := e.cluster.Apply(ctx, cluster.CommandMonitorCheck, seed); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) evaluateNode(ctx context.Context, nodeID string, expectedObservers []model.ClusterMember, singleNodeMode bool, now time.Time) error {
	prev, err := e.store.GetNodeState(ctx, nodeID)
	if err != nil {
		return err
	}
	heartbeat, err := e.store.LatestHeartbeat(ctx, nodeID)
	if err != nil {
		return err
	}
	probes, err := e.store.RecentProbesForTarget(ctx, nodeID, now.Add(-45*time.Second), probeSampleLimit(len(expectedObservers), e.cfg.LoopInterval()))
	if err != nil {
		return err
	}
	probes = filterProbesForObservers(probes, expectedObservers)
	next := e.assessNode(nodeID, heartbeat, probes, prev, now, len(expectedObservers), singleNodeMode)
	if prev == nil || stateChanged(*prev, next) {
		if _, err := e.cluster.Apply(ctx, cluster.CommandNodeState, next); err != nil {
			return err
		}
		if err := e.recordStateChangeEvent(ctx, prev, next); err != nil {
			return err
		}
	}
	return e.reconcileIncidents(ctx, next, prev, now)
}

func (e *Engine) assessNode(nodeID string, heartbeat *model.NodeHeartbeat, probes []model.ProbeObservation, prev *model.NodeState, now time.Time, expectedPeers int, singleNodeMode bool) model.NodeState {
	next := model.NodeState{
		NodeID:          nodeID,
		Status:          model.StatusUnknown,
		Reason:          "waiting for observations",
		RuleKey:         "telemetry",
		LastEvaluatedAt: now,
	}
	if prev != nil {
		next.BadStreak = prev.BadStreak
		next.GoodStreak = prev.GoodStreak
	}

	if heartbeat != nil {
		next.LastHeartbeatAt = heartbeat.CollectedAt
		next.CPUPct = heartbeat.CPUPct
		next.MemPct = heartbeat.MemPct
		next.DiskPct = heartbeat.DiskPct
		next.Load1 = heartbeat.Load1
		next.UptimeS = heartbeat.UptimeS
		next.Services = heartbeat.Services
	}

	successSources := map[string]struct{}{}
	allSources := map[string]struct{}{}
	var evidence []string
	for _, probe := range probes {
		allSources[probe.SourceNodeID] = struct{}{}
		if probe.TCP443OK || probe.TCP22OK || probe.HTTPOK {
			successSources[probe.SourceNodeID] = struct{}{}
		}
	}

	lastSources := make([]string, 0, len(allSources))
	for source := range allSources {
		lastSources = append(lastSources, source)
	}
	slices.Sort(lastSources)
	next.LastProbeSummary = model.ProbeSummary{
		SuccessfulPeers: len(successSources),
		TotalPeers:      len(allSources),
		ExpectedPeers:   expectedPeers,
		Reachable:       len(successSources) > 0,
		LastSources:     lastSources,
	}

	if heartbeat == nil {
		next.Status = model.StatusUnknown
		next.Reason = "no heartbeat yet"
		next.RuleKey = "telemetry"
		evidence = append(evidence, "leader has not received any heartbeat from this node yet")
		return updateStreaks(next, prev, evidence)
	}

	heartbeatAge := now.Sub(heartbeat.CollectedAt)
	next.ReplicatedFresh = heartbeatAge <= 45*time.Second
	evidence = append(evidence, fmt.Sprintf("heartbeat age %s", heartbeatAge.Round(time.Second)))
	if expectedPeers > 0 {
		evidence = append(evidence, fmt.Sprintf("%d/%d expected observers reported this node", len(allSources), expectedPeers))
		evidence = append(evidence, fmt.Sprintf("%d/%d expected observers confirmed reachability", len(successSources), expectedPeers))
	} else if len(allSources) > 0 {
		evidence = append(evidence, fmt.Sprintf("%d/%d peers still reach node", len(successSources), len(allSources)))
	} else {
		evidence = append(evidence, "no fresh observer probes in evaluation window")
	}

	serviceFailures := failingServices(heartbeat.Services)
	negativeEvidenceThreshold := requiredNegativeEvidenceReports(expectedPeers)
	switch {
	case heartbeatAge > 45*time.Second && len(successSources) > 0:
		next.Status = model.StatusDegraded
		next.Reason = "node is reachable but agent heartbeat is stale"
		next.RuleKey = "agent-stale"
	case heartbeatAge > 45*time.Second && negativeEvidenceThreshold > 0 && len(allSources) >= negativeEvidenceThreshold && len(successSources) == 0:
		next.Status = model.StatusCritical
		next.Reason = "heartbeat stale and no peers can reach the node"
		next.RuleKey = "availability"
	case heartbeatAge > 45*time.Second:
		next.Status = model.StatusDegraded
		next.Reason = "heartbeat stale with insufficient observer evidence"
		next.RuleKey = "agent-stale"
	case heartbeat.DiskPct >= e.cfg.Thresholds.DiskCrit || heartbeat.MemPct >= e.cfg.Thresholds.MemCrit:
		next.Status = model.StatusCritical
		next.Reason = "critical resource threshold exceeded"
		next.RuleKey = "resource-critical"
	case len(serviceFailures) > 0:
		next.Status = model.StatusDegraded
		next.Reason = "required service check failed"
		next.RuleKey = "service"
		evidence = append(evidence, "failed services: "+strings.Join(serviceFailures, ", "))
	case heartbeat.CPUPct >= e.cfg.Thresholds.CPUCrit || heartbeat.MemPct >= e.cfg.Thresholds.MemWarn || heartbeat.DiskPct >= e.cfg.Thresholds.DiskWarn:
		next.Status = model.StatusDegraded
		next.Reason = "resource threshold exceeded"
		next.RuleKey = "resource"
	case singleNodeMode:
		next.Status = model.StatusHealthy
		next.Reason = "single-node local mode with fresh heartbeat"
		next.RuleKey = "healthy-local"
	case len(allSources) == 0:
		next.Status = model.StatusDegraded
		next.Reason = "missing observer visibility confirmation"
		next.RuleKey = "visibility"
	case len(successSources) == 0:
		next.Status = model.StatusDegraded
		next.Reason = "observers report the public surface unreachable"
		next.RuleKey = "visibility"
	default:
		next.Status = model.StatusHealthy
		next.Reason = "fresh heartbeat and observer reachability confirmed"
		next.RuleKey = "healthy"
	}

	if next.Status == model.StatusCritical && len(serviceFailures) > 0 {
		evidence = append(evidence, "service failures coincided with the critical state")
	}
	return updateStreaks(next, prev, evidence)
}

func updateStreaks(next model.NodeState, prev *model.NodeState, evidence []string) model.NodeState {
	next.PrimaryEvidence = evidence
	if next.Status == model.StatusHealthy {
		if prev != nil && prev.Status == model.StatusHealthy {
			next.GoodStreak = prev.GoodStreak + 1
		} else {
			next.GoodStreak = 1
		}
		next.BadStreak = 0
		return next
	}
	if prev != nil && prev.Status == next.Status && prev.RuleKey == next.RuleKey {
		next.BadStreak = prev.BadStreak + 1
	} else {
		next.BadStreak = 1
	}
	next.GoodStreak = 0
	return next
}

func stateChanged(prev model.NodeState, next model.NodeState) bool {
	return prev.Status != next.Status ||
		prev.Reason != next.Reason ||
		prev.BadStreak != next.BadStreak ||
		prev.GoodStreak != next.GoodStreak ||
		prev.CPUPct != next.CPUPct ||
		prev.MemPct != next.MemPct ||
		prev.DiskPct != next.DiskPct ||
		prev.ReplicatedFresh != next.ReplicatedFresh ||
		prev.RuleKey != next.RuleKey ||
		probeSummaryChanged(prev.LastProbeSummary, next.LastProbeSummary)
}

func failingServices(services []model.ServiceCheck) []string {
	var names []string
	for _, service := range services {
		if service.Status != "active" && service.Status != "running" && service.Status != "healthy" {
			names = append(names, service.Name)
		}
	}
	return names
}

func probeSummaryChanged(prev model.ProbeSummary, next model.ProbeSummary) bool {
	if prev.SuccessfulPeers != next.SuccessfulPeers ||
		prev.TotalPeers != next.TotalPeers ||
		prev.ExpectedPeers != next.ExpectedPeers ||
		prev.Reachable != next.Reachable ||
		len(prev.LastSources) != len(next.LastSources) {
		return true
	}
	for i := range prev.LastSources {
		if prev.LastSources[i] != next.LastSources[i] {
			return true
		}
	}
	return false
}

func filterProbesForObservers(probes []model.ProbeObservation, expectedObservers []model.ClusterMember) []model.ProbeObservation {
	if len(expectedObservers) == 0 {
		return nil
	}
	allowed := make(map[string]struct{}, len(expectedObservers))
	for _, observer := range expectedObservers {
		allowed[observer.NodeID] = struct{}{}
	}
	filtered := make([]model.ProbeObservation, 0, len(probes))
	for _, probe := range probes {
		if _, ok := allowed[probe.SourceNodeID]; !ok {
			continue
		}
		filtered = append(filtered, probe)
	}
	return filtered
}

func requiredNegativeEvidenceReports(expectedPeers int) int {
	switch {
	case expectedPeers <= 0:
		return 0
	case expectedPeers == 1:
		return 1
	default:
		return 2
	}
}

func probeSampleLimit(expectedPeers int, interval time.Duration) int {
	if interval <= 0 {
		interval = 15 * time.Second
	}
	samplesPerObserver := int((45 * time.Second) / interval)
	if (45*time.Second)%interval != 0 {
		samplesPerObserver++
	}
	samplesPerObserver++
	if samplesPerObserver < 1 {
		samplesPerObserver = 1
	}
	if expectedPeers < 1 {
		expectedPeers = 1
	}
	limit := expectedPeers * samplesPerObserver
	if limit < 12 {
		return 12
	}
	return limit
}

func (e *Engine) recordStateChangeEvent(ctx context.Context, prev *model.NodeState, next model.NodeState) error {
	if prev != nil && prev.Status == next.Status && prev.Reason == next.Reason {
		return nil
	}
	event := model.Event{
		ID:        uuid.NewString(),
		Kind:      "state_change",
		Severity:  next.Status,
		NodeID:    next.NodeID,
		Title:     fmt.Sprintf("%s shifted to %s", next.NodeID, strings.ToUpper(next.Status)),
		Body:      next.Reason,
		CreatedAt: next.LastEvaluatedAt,
		Meta: map[string]any{
			"rule_key": next.RuleKey,
		},
	}
	_, err := e.cluster.Apply(ctx, cluster.CommandEvent, event)
	return err
}

func (e *Engine) reconcileIncidents(ctx context.Context, next model.NodeState, prev *model.NodeState, now time.Time) error {
	incidents, err := e.store.ListIncidentsForNode(ctx, next.NodeID, 16)
	if err != nil {
		return err
	}
	var active []*model.Incident
	for i := range incidents {
		if incidents[i].Status == model.IncidentStatusActive {
			active = append(active, &incidents[i])
		}
	}

	if next.Status == model.StatusHealthy {
		if next.GoodStreak < 2 {
			return nil
		}
		for _, inc := range active {
			resolvedAt := now
			inc.Status = model.IncidentStatusResolved
			inc.ResolvedAt = &resolvedAt
			if _, err := e.cluster.Apply(ctx, cluster.CommandIncident, *inc); err != nil {
				return err
			}
			if err := e.sendAlert(ctx, "resolved", *inc); err != nil {
				e.logger.Error("send resolved alert failed", "incident_id", inc.ID, "error", err)
			}
			if err := e.addIncidentEvent(ctx, "incident_resolved", *inc, "Incident resolved"); err != nil {
				return err
			}
		}
		return nil
	}

	for _, inc := range active {
		if inc.RuleKey != next.RuleKey {
			resolvedAt := now
			inc.Status = model.IncidentStatusResolved
			inc.ResolvedAt = &resolvedAt
			if _, err := e.cluster.Apply(ctx, cluster.CommandIncident, *inc); err != nil {
				return err
			}
		}
	}

	activeIncident, err := e.store.ActiveIncidentByRule(ctx, next.NodeID, next.RuleKey)
	if err != nil {
		return err
	}
	if activeIncident == nil && next.BadStreak >= 2 {
		incident := model.Incident{
			ID:       next.NodeID + ":" + next.RuleKey,
			NodeID:   next.NodeID,
			RuleKey:  next.RuleKey,
			Severity: next.Status,
			Status:   model.IncidentStatusActive,
			Summary:  next.Reason,
			Detail:   strings.Join(next.PrimaryEvidence, " | "),
			OpenedAt: now,
		}
		if _, err := e.cluster.Apply(ctx, cluster.CommandIncident, incident); err != nil {
			return err
		}
		if err := e.sendAlert(ctx, "opened", incident); err != nil {
			e.logger.Error("send opened alert failed", "incident_id", incident.ID, "error", err)
		}
		return e.addIncidentEvent(ctx, "incident_opened", incident, "Incident opened")
	}

	if activeIncident != nil && shouldRepeat(activeIncident.LastNotifiedAt, now) {
		if err := e.sendAlert(ctx, "reminder", *activeIncident); err != nil {
			e.logger.Error("send reminder alert failed", "incident_id", activeIncident.ID, "error", err)
		}
	}
	return nil
}

func shouldRepeat(lastNotified *time.Time, now time.Time) bool {
	if lastNotified == nil {
		return true
	}
	return now.Sub(*lastNotified) >= 30*time.Minute
}

func (e *Engine) sendAlert(ctx context.Context, action string, incident model.Incident) error {
	for _, notifier := range e.notifiers {
		deliveryKey := incident.ID + ":" + notifier.Name() + ":" + deliverySlot(action, incident, time.Now().UTC())
		response, err := e.claimAndSend(ctx, notifier, action, incident, deliveryKey)
		if err != nil {
			e.logger.Error("alert delivery failed", "channel", notifier.Name(), "incident_id", incident.ID, "error", err)
			continue
		}
		e.logger.Info("alert delivered", "channel", notifier.Name(), "incident_id", incident.ID, "response", response)
	}
	return nil
}

func deliverySlot(action string, incident model.Incident, now time.Time) string {
	switch action {
	case "opened":
		return "opened"
	case "resolved":
		return "resolved"
	default:
		return "reminder-" + now.Truncate(30*time.Minute).Format("200601021504")
	}
}

func (e *Engine) claimAndSend(ctx context.Context, notifier notify.Notifier, action string, incident model.Incident, deliveryKey string) (string, error) {
	result, err := e.cluster.Apply(ctx, cluster.CommandAlertClaim, model.AlertClaim{
		IncidentID:  incident.ID,
		Channel:     notifier.Name(),
		DeliveryKey: deliveryKey,
		CreatedAt:   time.Now().UTC(),
	})
	if err != nil {
		return "", err
	}
	claimResult, _ := result.(cluster.CommandResult)
	if !claimResult.Claimed {
		return "deduped", nil
	}
	response, sendErr := notifier.Send(ctx, action, incident)
	status := "sent"
	if sendErr != nil {
		status = "failed"
		response = sendErr.Error()
	}
	if _, err := e.cluster.Apply(ctx, cluster.CommandAlertCompletion, model.AlertCompletion{
		IncidentID:  incident.ID,
		DeliveryKey: deliveryKey,
		Status:      status,
		Response:    response,
		SentAt:      time.Now().UTC(),
	}); err != nil {
		return response, err
	}
	return response, sendErr
}

func (e *Engine) addIncidentEvent(ctx context.Context, kind string, incident model.Incident, title string) error {
	event := model.Event{
		ID:        uuid.NewString(),
		Kind:      kind,
		Severity:  incident.Severity,
		NodeID:    incident.NodeID,
		Title:     title,
		Body:      incident.Summary,
		CreatedAt: time.Now().UTC(),
		Meta: map[string]any{
			"rule_key": incident.RuleKey,
		},
	}
	_, err := e.cluster.Apply(ctx, cluster.CommandEvent, event)
	return err
}

func (e *Engine) syncIngress(ctx context.Context, now time.Time) error {
	current, err := e.store.GetIngressState(ctx)
	if err != nil {
		return err
	}
	states, err := e.store.ListNodeStates(ctx)
	if err != nil {
		return err
	}

	members, err := e.cluster.ActiveMembers(ctx)
	if err != nil {
		return err
	}
	targetPeer, ok := selectIngressTargetPeer(e.cfg.Cluster.NodeID, members, states, current)
	if !ok {
		return nil
	}
	if current != nil && current.ActiveNodeID == targetPeer.NodeID && current.DesiredIP == targetPeer.PublicIPv4 && current.DNSSynced {
		return nil
	}

	ingressState := model.IngressState{
		ActiveNodeID: targetPeer.NodeID,
		DesiredIP:    targetPeer.PublicIPv4,
		DNSSynced:    false,
		DNSSyncedAt:  now,
		UpdatedAt:    now,
	}
	if _, err := e.cluster.Apply(ctx, cluster.CommandIngress, ingressState); err != nil {
		return err
	}

	if !e.cf.Enabled() {
		ingressState.DNSSynced = true
		ingressState.DNSSyncedAt = now
		_, err := e.cluster.Apply(ctx, cluster.CommandIngress, ingressState)
		return err
	}

	var syncErr error
	for _, delay := range cloudflare.BackoffSchedule() {
		if syncErr = e.cf.UpdateARecord(ctx, targetPeer.PublicIPv4); syncErr == nil {
			break
		}
		time.Sleep(delay)
	}
	if syncErr != nil {
		ingressState.LastDNSError = syncErr.Error()
		_, _ = e.cluster.Apply(ctx, cluster.CommandIngress, ingressState)
		_, _ = e.cluster.Apply(ctx, cluster.CommandEvent, model.Event{
			ID:        uuid.NewString(),
			Kind:      "dns_sync_failed",
			Severity:  model.StatusCritical,
			NodeID:    targetPeer.NodeID,
			Title:     "DNS sync failed",
			Body:      syncErr.Error(),
			CreatedAt: time.Now().UTC(),
			Meta: map[string]any{
				"desired_ip": targetPeer.PublicIPv4,
			},
		})
		return syncErr
	}

	ingressState.DNSSynced = true
	ingressState.DNSSyncedAt = time.Now().UTC()
	ingressState.LastDNSError = ""
	if _, err := e.cluster.Apply(ctx, cluster.CommandIngress, ingressState); err != nil {
		return err
	}
	_, err = e.cluster.Apply(ctx, cluster.CommandEvent, model.Event{
		ID:        uuid.NewString(),
		Kind:      "dns_sync",
		Severity:  model.StatusHealthy,
		NodeID:    targetPeer.NodeID,
		Title:     "Ingress DNS moved",
		Body:      fmt.Sprintf("monitor hostname now points at %s", targetPeer.NodeID),
		CreatedAt: time.Now().UTC(),
		Meta: map[string]any{
			"desired_ip": targetPeer.PublicIPv4,
		},
	})
	return err
}

type ingressTargetCandidate struct {
	state model.NodeState
	peer  model.ClusterMember
}

func selectIngressTargetPeer(selfNodeID string, members []model.ClusterMember, states []model.NodeState, current *model.IngressState) (model.ClusterMember, bool) {
	memberMap := make(map[string]model.ClusterMember, len(members))
	for _, member := range members {
		memberMap[member.NodeID] = member
	}
	eligible := make([]ingressTargetCandidate, 0, len(states))
	for _, state := range states {
		if !isIngressStateEligible(selfNodeID, state) {
			continue
		}
		peer, ok := memberMap[state.NodeID]
		if !ok || !peer.IsIngressCandidate() {
			continue
		}
		eligible = append(eligible, ingressTargetCandidate{
			state: state,
			peer:  peer,
		})
	}
	if len(eligible) == 0 {
		return model.ClusterMember{}, false
	}

	slices.SortFunc(eligible, func(a, b ingressTargetCandidate) int {
		if a.peer.Priority != b.peer.Priority {
			return b.peer.Priority - a.peer.Priority
		}
		if current != nil && a.state.NodeID == current.ActiveNodeID {
			return -1
		}
		if current != nil && b.state.NodeID == current.ActiveNodeID {
			return 1
		}
		return strings.Compare(a.state.NodeID, b.state.NodeID)
	})

	return eligible[0].peer, true
}

func isIngressStateEligible(selfNodeID string, state model.NodeState) bool {
	if state.Status != model.StatusHealthy && (state.Status != model.StatusDegraded || state.RuleKey == "availability") {
		return false
	}
	return state.LastProbeSummary.Reachable || state.NodeID == selfNodeID
}
