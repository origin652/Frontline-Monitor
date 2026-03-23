package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"vps-monitor/internal/model"
)

const timeLayout = time.RFC3339Nano

type Store struct {
	db *sql.DB
}

type SnapshotData struct {
	MetricSamples   []model.NodeHeartbeat    `json:"metric_samples"`
	ProbeSamples    []model.ProbeObservation `json:"probe_samples"`
	NodeStates      []model.NodeState        `json:"node_states"`
	Incidents       []model.Incident         `json:"incidents"`
	AlertDeliveries []model.AlertDelivery    `json:"alert_deliveries"`
	Events          []model.Event            `json:"events"`
	Ingress         *model.IngressState      `json:"ingress,omitempty"`
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create sqlite dir: %w", err)
	}

	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)", filepath.ToSlash(path))
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)

	store := &Store{db: db}
	if err := store.init(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) init(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS metric_samples (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			node_id TEXT NOT NULL,
			collected_at TEXT NOT NULL,
			cpu_pct REAL NOT NULL,
			mem_pct REAL NOT NULL,
			disk_pct REAL NOT NULL,
			load1 REAL NOT NULL,
			uptime_s INTEGER NOT NULL,
			services_json TEXT NOT NULL,
			docker_json TEXT NOT NULL,
			http_json TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_metric_samples_node_collected ON metric_samples(node_id, collected_at DESC)`,
		`CREATE TABLE IF NOT EXISTS probe_samples (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_node_id TEXT NOT NULL,
			target_node_id TEXT NOT NULL,
			collected_at TEXT NOT NULL,
			tcp_22_ok INTEGER NOT NULL,
			tcp_443_ok INTEGER NOT NULL,
			http_ok INTEGER NOT NULL,
			ssh_banner_ms INTEGER NOT NULL,
			ports_json TEXT NOT NULL,
			http_checks_json TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_probe_samples_target_collected ON probe_samples(target_node_id, collected_at DESC)`,
		`CREATE TABLE IF NOT EXISTS node_state (
			node_id TEXT PRIMARY KEY,
			status TEXT NOT NULL,
			reason TEXT NOT NULL,
			rule_key TEXT NOT NULL,
			last_heartbeat_at TEXT NOT NULL,
			last_probe_summary_json TEXT NOT NULL,
			replicated_fresh INTEGER NOT NULL,
			cpu_pct REAL NOT NULL,
			mem_pct REAL NOT NULL,
			disk_pct REAL NOT NULL,
			load1 REAL NOT NULL,
			uptime_s INTEGER NOT NULL,
			services_json TEXT NOT NULL,
			bad_streak INTEGER NOT NULL,
			good_streak INTEGER NOT NULL,
			last_evaluated_at TEXT NOT NULL,
			primary_evidence_json TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS incidents (
			id TEXT PRIMARY KEY,
			node_id TEXT NOT NULL,
			rule_key TEXT NOT NULL,
			severity TEXT NOT NULL,
			status TEXT NOT NULL,
			summary TEXT NOT NULL,
			detail TEXT NOT NULL,
			opened_at TEXT NOT NULL,
			resolved_at TEXT,
			last_notified_at TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_incidents_status_node ON incidents(status, node_id)`,
		`CREATE TABLE IF NOT EXISTS alert_deliveries (
			delivery_key TEXT PRIMARY KEY,
			incident_id TEXT NOT NULL,
			channel TEXT NOT NULL,
			status TEXT NOT NULL,
			response TEXT NOT NULL,
			created_at TEXT NOT NULL,
			sent_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS events (
			id TEXT PRIMARY KEY,
			kind TEXT NOT NULL,
			severity TEXT NOT NULL,
			node_id TEXT NOT NULL,
			title TEXT NOT NULL,
			body TEXT NOT NULL,
			meta_json TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_events_created ON events(created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS ingress_state (
			singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
			active_node_id TEXT NOT NULL,
			desired_ip TEXT NOT NULL,
			dns_synced INTEGER NOT NULL,
			dns_synced_at TEXT NOT NULL,
			last_dns_error TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
	}

	for _, stmt := range statements {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("init sqlite schema: %w", err)
		}
	}
	return nil
}

func (s *Store) RecordHeartbeat(ctx context.Context, hb model.NodeHeartbeat) error {
	servicesJSON, err := marshalJSON(hb.Services)
	if err != nil {
		return err
	}
	dockerJSON, err := marshalJSON(hb.DockerChecks)
	if err != nil {
		return err
	}
	httpJSON, err := marshalJSON(hb.LocalHTTPChecks)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO metric_samples (
			node_id, collected_at, cpu_pct, mem_pct, disk_pct, load1, uptime_s, services_json, docker_json, http_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		hb.NodeID, hb.CollectedAt.Format(timeLayout), hb.CPUPct, hb.MemPct, hb.DiskPct, hb.Load1, hb.UptimeS, servicesJSON, dockerJSON, httpJSON,
	)
	return err
}

func (s *Store) RecordProbe(ctx context.Context, probe model.ProbeObservation) error {
	portsJSON, err := marshalJSON(probe.Ports)
	if err != nil {
		return err
	}
	httpJSON, err := marshalJSON(probe.HTTPChecks)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO probe_samples (
			source_node_id, target_node_id, collected_at, tcp_22_ok, tcp_443_ok, http_ok, ssh_banner_ms, ports_json, http_checks_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		probe.SourceNodeID, probe.TargetNodeID, probe.CollectedAt.Format(timeLayout), boolToInt(probe.TCP22OK), boolToInt(probe.TCP443OK), boolToInt(probe.HTTPOK), probe.SSHBannerMS, portsJSON, httpJSON,
	)
	return err
}

func (s *Store) UpsertNodeState(ctx context.Context, state model.NodeState) error {
	summaryJSON, err := marshalJSON(state.LastProbeSummary)
	if err != nil {
		return err
	}
	servicesJSON, err := marshalJSON(state.Services)
	if err != nil {
		return err
	}
	evidenceJSON, err := marshalJSON(state.PrimaryEvidence)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO node_state (
			node_id, status, reason, rule_key, last_heartbeat_at, last_probe_summary_json, replicated_fresh,
			cpu_pct, mem_pct, disk_pct, load1, uptime_s, services_json, bad_streak, good_streak,
			last_evaluated_at, primary_evidence_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(node_id) DO UPDATE SET
			status = excluded.status,
			reason = excluded.reason,
			rule_key = excluded.rule_key,
			last_heartbeat_at = excluded.last_heartbeat_at,
			last_probe_summary_json = excluded.last_probe_summary_json,
			replicated_fresh = excluded.replicated_fresh,
			cpu_pct = excluded.cpu_pct,
			mem_pct = excluded.mem_pct,
			disk_pct = excluded.disk_pct,
			load1 = excluded.load1,
			uptime_s = excluded.uptime_s,
			services_json = excluded.services_json,
			bad_streak = excluded.bad_streak,
			good_streak = excluded.good_streak,
			last_evaluated_at = excluded.last_evaluated_at,
			primary_evidence_json = excluded.primary_evidence_json`,
		state.NodeID, state.Status, state.Reason, state.RuleKey, state.LastHeartbeatAt.Format(timeLayout), summaryJSON, boolToInt(state.ReplicatedFresh),
		state.CPUPct, state.MemPct, state.DiskPct, state.Load1, state.UptimeS, servicesJSON, state.BadStreak, state.GoodStreak,
		state.LastEvaluatedAt.Format(timeLayout), evidenceJSON,
	)
	return err
}

func (s *Store) GetNodeState(ctx context.Context, nodeID string) (*model.NodeState, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT node_id, status, reason, rule_key, last_heartbeat_at, last_probe_summary_json, replicated_fresh,
			cpu_pct, mem_pct, disk_pct, load1, uptime_s, services_json, bad_streak, good_streak,
			last_evaluated_at, primary_evidence_json
		FROM node_state WHERE node_id = ?`, nodeID)
	return scanNodeState(row)
}

func (s *Store) ListNodeStates(ctx context.Context) ([]model.NodeState, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT node_id, status, reason, rule_key, last_heartbeat_at, last_probe_summary_json, replicated_fresh,
			cpu_pct, mem_pct, disk_pct, load1, uptime_s, services_json, bad_streak, good_streak,
			last_evaluated_at, primary_evidence_json
		FROM node_state ORDER BY node_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.NodeState
	for rows.Next() {
		state, err := scanNodeState(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *state)
	}
	return out, rows.Err()
}

func (s *Store) LatestHeartbeat(ctx context.Context, nodeID string) (*model.NodeHeartbeat, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT node_id, collected_at, cpu_pct, mem_pct, disk_pct, load1, uptime_s, services_json, docker_json, http_json
		FROM metric_samples
		WHERE node_id = ?
		ORDER BY collected_at DESC
		LIMIT 1`, nodeID)
	return scanHeartbeat(row)
}

func (s *Store) History(ctx context.Context, nodeID string, metric string, from, to time.Time) ([]model.MetricPoint, error) {
	column := "cpu_pct"
	switch metric {
	case "mem_pct":
		column = "mem_pct"
	case "disk_pct":
		column = "disk_pct"
	case "load1":
		column = "load1"
	}

	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT collected_at, %s FROM metric_samples
		WHERE node_id = ? AND collected_at BETWEEN ? AND ?
		ORDER BY collected_at ASC`, column),
		nodeID, from.Format(timeLayout), to.Format(timeLayout),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []model.MetricPoint
	for rows.Next() {
		var collectedAt string
		var value float64
		if err := rows.Scan(&collectedAt, &value); err != nil {
			return nil, err
		}
		ts, err := time.Parse(timeLayout, collectedAt)
		if err != nil {
			return nil, err
		}
		points = append(points, model.MetricPoint{Timestamp: ts, Value: value})
	}
	return points, rows.Err()
}

func (s *Store) RecentProbesForTarget(ctx context.Context, nodeID string, since time.Time, limit int) ([]model.ProbeObservation, error) {
	if limit <= 0 {
		limit = 12
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT source_node_id, target_node_id, collected_at, tcp_22_ok, tcp_443_ok, http_ok, ssh_banner_ms, ports_json, http_checks_json
		FROM probe_samples
		WHERE target_node_id = ? AND collected_at >= ?
		ORDER BY collected_at DESC
		LIMIT ?`, nodeID, since.Format(timeLayout), limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var probes []model.ProbeObservation
	for rows.Next() {
		var sourceNodeID, targetNodeID, collectedAt string
		var tcp22, tcp443, httpOK int
		var sshBannerMS int64
		var portsJSON, checksJSON string
		if err := rows.Scan(&sourceNodeID, &targetNodeID, &collectedAt, &tcp22, &tcp443, &httpOK, &sshBannerMS, &portsJSON, &checksJSON); err != nil {
			return nil, err
		}
		ts, err := time.Parse(timeLayout, collectedAt)
		if err != nil {
			return nil, err
		}
		var ports []model.PortResult
		var checks []model.HTTPCheckResult
		if err := json.Unmarshal([]byte(portsJSON), &ports); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(checksJSON), &checks); err != nil {
			return nil, err
		}
		probes = append(probes, model.ProbeObservation{
			SourceNodeID: sourceNodeID,
			TargetNodeID: targetNodeID,
			CollectedAt:  ts,
			TCP22OK:      tcp22 == 1,
			TCP443OK:     tcp443 == 1,
			HTTPOK:       httpOK == 1,
			SSHBannerMS:  sshBannerMS,
			Ports:        ports,
			HTTPChecks:   checks,
		})
	}
	return probes, rows.Err()
}

func (s *Store) UpsertIncident(ctx context.Context, inc model.Incident) error {
	var resolvedAt any
	if inc.ResolvedAt != nil {
		resolvedAt = inc.ResolvedAt.Format(timeLayout)
	}
	var lastNotifiedAt any
	if inc.LastNotifiedAt != nil {
		lastNotifiedAt = inc.LastNotifiedAt.Format(timeLayout)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO incidents (
			id, node_id, rule_key, severity, status, summary, detail, opened_at, resolved_at, last_notified_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			severity = excluded.severity,
			status = excluded.status,
			summary = excluded.summary,
			detail = excluded.detail,
			resolved_at = excluded.resolved_at,
			last_notified_at = excluded.last_notified_at`,
		inc.ID, inc.NodeID, inc.RuleKey, inc.Severity, inc.Status, inc.Summary, inc.Detail, inc.OpenedAt.Format(timeLayout), resolvedAt, lastNotifiedAt,
	)
	return err
}

func (s *Store) ActiveIncidentByRule(ctx context.Context, nodeID, ruleKey string) (*model.Incident, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, node_id, rule_key, severity, status, summary, detail, opened_at, resolved_at, last_notified_at
		FROM incidents
		WHERE node_id = ? AND rule_key = ? AND status = ?
		LIMIT 1`, nodeID, ruleKey, model.IncidentStatusActive,
	)
	return scanIncident(row)
}

func (s *Store) ListIncidents(ctx context.Context, status string, limit int) ([]model.Incident, error) {
	query := `
		SELECT id, node_id, rule_key, severity, status, summary, detail, opened_at, resolved_at, last_notified_at
		FROM incidents`
	var args []any
	if status != "" {
		query += ` WHERE status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY opened_at DESC`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.Incident
	for rows.Next() {
		inc, err := scanIncident(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *inc)
	}
	return out, rows.Err()
}

func (s *Store) ListIncidentsForNode(ctx context.Context, nodeID string, limit int) ([]model.Incident, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, node_id, rule_key, severity, status, summary, detail, opened_at, resolved_at, last_notified_at
		FROM incidents
		WHERE node_id = ?
		ORDER BY opened_at DESC
		LIMIT ?`, nodeID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidents []model.Incident
	for rows.Next() {
		inc, err := scanIncident(rows)
		if err != nil {
			return nil, err
		}
		incidents = append(incidents, *inc)
	}
	return incidents, rows.Err()
}

func (s *Store) ClaimAlertDelivery(ctx context.Context, delivery model.AlertDelivery) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO alert_deliveries (
			delivery_key, incident_id, channel, status, response, created_at, sent_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		delivery.DeliveryKey, delivery.IncidentID, delivery.Channel, delivery.Status, delivery.Response, delivery.CreatedAt.Format(timeLayout), nil,
	)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected == 1, nil
}

func (s *Store) CompleteAlertDelivery(ctx context.Context, deliveryKey, status, response string, sentAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE alert_deliveries
		SET status = ?, response = ?, sent_at = ?
		WHERE delivery_key = ?`, status, response, sentAt.Format(timeLayout), deliveryKey,
	)
	return err
}

func (s *Store) UpdateIncidentLastNotified(ctx context.Context, incidentID string, sentAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE incidents SET last_notified_at = ? WHERE id = ?`,
		sentAt.Format(timeLayout), incidentID,
	)
	return err
}

func (s *Store) AddEvent(ctx context.Context, event model.Event) error {
	metaJSON, err := marshalJSON(event.Meta)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO events (id, kind, severity, node_id, title, body, meta_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID, event.Kind, event.Severity, event.NodeID, event.Title, event.Body, metaJSON, event.CreatedAt.Format(timeLayout),
	)
	return err
}

func (s *Store) ListEvents(ctx context.Context, limit int) ([]model.Event, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, kind, severity, node_id, title, body, meta_json, created_at
		FROM events
		ORDER BY created_at DESC
		LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []model.Event
	for rows.Next() {
		var event model.Event
		var metaJSON, createdAt string
		if err := rows.Scan(&event.ID, &event.Kind, &event.Severity, &event.NodeID, &event.Title, &event.Body, &metaJSON, &createdAt); err != nil {
			return nil, err
		}
		ts, err := time.Parse(timeLayout, createdAt)
		if err != nil {
			return nil, err
		}
		event.CreatedAt = ts
		if err := json.Unmarshal([]byte(metaJSON), &event.Meta); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *Store) UpsertIngressState(ctx context.Context, state model.IngressState) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO ingress_state (singleton, active_node_id, desired_ip, dns_synced, dns_synced_at, last_dns_error, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(singleton) DO UPDATE SET
			active_node_id = excluded.active_node_id,
			desired_ip = excluded.desired_ip,
			dns_synced = excluded.dns_synced,
			dns_synced_at = excluded.dns_synced_at,
			last_dns_error = excluded.last_dns_error,
			updated_at = excluded.updated_at`,
		state.ActiveNodeID, state.DesiredIP, boolToInt(state.DNSSynced), state.DNSSyncedAt.Format(timeLayout), state.LastDNSError, state.UpdatedAt.Format(timeLayout),
	)
	return err
}

func (s *Store) GetIngressState(ctx context.Context) (*model.IngressState, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT active_node_id, desired_ip, dns_synced, dns_synced_at, last_dns_error, updated_at
		FROM ingress_state WHERE singleton = 1`)
	var state model.IngressState
	var dnsSynced int
	var syncedAt, updatedAt string
	err := row.Scan(&state.ActiveNodeID, &state.DesiredIP, &dnsSynced, &syncedAt, &state.LastDNSError, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	state.DNSSynced = dnsSynced == 1
	if state.DNSSyncedAt, err = time.Parse(timeLayout, syncedAt); err != nil {
		return nil, err
	}
	if state.UpdatedAt, err = time.Parse(timeLayout, updatedAt); err != nil {
		return nil, err
	}
	return &state, nil
}

func (s *Store) Snapshot(ctx context.Context) (*SnapshotData, error) {
	metrics, err := s.snapshotHeartbeats(ctx)
	if err != nil {
		return nil, err
	}
	probes, err := s.RecentProbesForAll(ctx)
	if err != nil {
		return nil, err
	}
	nodeStates, err := s.ListNodeStates(ctx)
	if err != nil {
		return nil, err
	}
	incidents, err := s.ListIncidents(ctx, "", 0)
	if err != nil {
		return nil, err
	}
	deliveries, err := s.listAlertDeliveries(ctx)
	if err != nil {
		return nil, err
	}
	events, err := s.ListEvents(ctx, 500)
	if err != nil {
		return nil, err
	}
	ingress, err := s.GetIngressState(ctx)
	if err != nil {
		return nil, err
	}
	return &SnapshotData{
		MetricSamples:   metrics,
		ProbeSamples:    probes,
		NodeStates:      nodeStates,
		Incidents:       incidents,
		AlertDeliveries: deliveries,
		Events:          events,
		Ingress:         ingress,
	}, nil
}

func (s *Store) Restore(ctx context.Context, snap SnapshotData) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, stmt := range []string{
		`DELETE FROM metric_samples`,
		`DELETE FROM probe_samples`,
		`DELETE FROM node_state`,
		`DELETE FROM incidents`,
		`DELETE FROM alert_deliveries`,
		`DELETE FROM events`,
		`DELETE FROM ingress_state`,
	} {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}

	for _, hb := range snap.MetricSamples {
		servicesJSON, _ := marshalJSON(hb.Services)
		dockerJSON, _ := marshalJSON(hb.DockerChecks)
		httpJSON, _ := marshalJSON(hb.LocalHTTPChecks)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO metric_samples (
				node_id, collected_at, cpu_pct, mem_pct, disk_pct, load1, uptime_s, services_json, docker_json, http_json
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			hb.NodeID, hb.CollectedAt.Format(timeLayout), hb.CPUPct, hb.MemPct, hb.DiskPct, hb.Load1, hb.UptimeS, servicesJSON, dockerJSON, httpJSON,
		); err != nil {
			return err
		}
	}

	for _, probe := range snap.ProbeSamples {
		portsJSON, _ := marshalJSON(probe.Ports)
		httpJSON, _ := marshalJSON(probe.HTTPChecks)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO probe_samples (
				source_node_id, target_node_id, collected_at, tcp_22_ok, tcp_443_ok, http_ok, ssh_banner_ms, ports_json, http_checks_json
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			probe.SourceNodeID, probe.TargetNodeID, probe.CollectedAt.Format(timeLayout), boolToInt(probe.TCP22OK), boolToInt(probe.TCP443OK), boolToInt(probe.HTTPOK), probe.SSHBannerMS, portsJSON, httpJSON,
		); err != nil {
			return err
		}
	}

	for _, state := range snap.NodeStates {
		summaryJSON, _ := marshalJSON(state.LastProbeSummary)
		servicesJSON, _ := marshalJSON(state.Services)
		evidenceJSON, _ := marshalJSON(state.PrimaryEvidence)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO node_state (
				node_id, status, reason, rule_key, last_heartbeat_at, last_probe_summary_json, replicated_fresh,
				cpu_pct, mem_pct, disk_pct, load1, uptime_s, services_json, bad_streak, good_streak, last_evaluated_at, primary_evidence_json
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			state.NodeID, state.Status, state.Reason, state.RuleKey, state.LastHeartbeatAt.Format(timeLayout), summaryJSON, boolToInt(state.ReplicatedFresh),
			state.CPUPct, state.MemPct, state.DiskPct, state.Load1, state.UptimeS, servicesJSON, state.BadStreak, state.GoodStreak, state.LastEvaluatedAt.Format(timeLayout), evidenceJSON,
		); err != nil {
			return err
		}
	}

	for _, inc := range snap.Incidents {
		var resolvedAt any
		if inc.ResolvedAt != nil {
			resolvedAt = inc.ResolvedAt.Format(timeLayout)
		}
		var lastNotified any
		if inc.LastNotifiedAt != nil {
			lastNotified = inc.LastNotifiedAt.Format(timeLayout)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO incidents (
				id, node_id, rule_key, severity, status, summary, detail, opened_at, resolved_at, last_notified_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			inc.ID, inc.NodeID, inc.RuleKey, inc.Severity, inc.Status, inc.Summary, inc.Detail, inc.OpenedAt.Format(timeLayout), resolvedAt, lastNotified,
		); err != nil {
			return err
		}
	}

	for _, delivery := range snap.AlertDeliveries {
		var sentAt any
		if delivery.SentAt != nil {
			sentAt = delivery.SentAt.Format(timeLayout)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO alert_deliveries (
				delivery_key, incident_id, channel, status, response, created_at, sent_at
			) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			delivery.DeliveryKey, delivery.IncidentID, delivery.Channel, delivery.Status, delivery.Response, delivery.CreatedAt.Format(timeLayout), sentAt,
		); err != nil {
			return err
		}
	}

	for _, event := range snap.Events {
		metaJSON, _ := marshalJSON(event.Meta)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO events (id, kind, severity, node_id, title, body, meta_json, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			event.ID, event.Kind, event.Severity, event.NodeID, event.Title, event.Body, metaJSON, event.CreatedAt.Format(timeLayout),
		); err != nil {
			return err
		}
	}

	if snap.Ingress != nil {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO ingress_state (singleton, active_node_id, desired_ip, dns_synced, dns_synced_at, last_dns_error, updated_at)
			VALUES (1, ?, ?, ?, ?, ?, ?)`,
			snap.Ingress.ActiveNodeID, snap.Ingress.DesiredIP, boolToInt(snap.Ingress.DNSSynced), snap.Ingress.DNSSyncedAt.Format(timeLayout), snap.Ingress.LastDNSError, snap.Ingress.UpdatedAt.Format(timeLayout),
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) PruneOldData(ctx context.Context, retentionDays int) error {
	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays).Format(timeLayout)
	for _, stmt := range []struct {
		query string
		arg   any
	}{
		{`DELETE FROM metric_samples WHERE collected_at < ?`, cutoff},
		{`DELETE FROM probe_samples WHERE collected_at < ?`, cutoff},
		{`DELETE FROM events WHERE created_at < ?`, cutoff},
		{`DELETE FROM alert_deliveries WHERE created_at < ?`, cutoff},
	} {
		if _, err := s.db.ExecContext(ctx, stmt.query, stmt.arg); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) RecentProbesForAll(ctx context.Context) ([]model.ProbeObservation, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT source_node_id, target_node_id, collected_at, tcp_22_ok, tcp_443_ok, http_ok, ssh_banner_ms, ports_json, http_checks_json
		FROM probe_samples ORDER BY collected_at DESC LIMIT 300`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var probes []model.ProbeObservation
	for rows.Next() {
		var sourceNodeID, targetNodeID, collectedAt string
		var tcp22, tcp443, httpOK int
		var sshBannerMS int64
		var portsJSON, checksJSON string
		if err := rows.Scan(&sourceNodeID, &targetNodeID, &collectedAt, &tcp22, &tcp443, &httpOK, &sshBannerMS, &portsJSON, &checksJSON); err != nil {
			return nil, err
		}
		ts, err := time.Parse(timeLayout, collectedAt)
		if err != nil {
			return nil, err
		}
		var ports []model.PortResult
		var checks []model.HTTPCheckResult
		if err := json.Unmarshal([]byte(portsJSON), &ports); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(checksJSON), &checks); err != nil {
			return nil, err
		}
		probes = append(probes, model.ProbeObservation{
			SourceNodeID: sourceNodeID,
			TargetNodeID: targetNodeID,
			CollectedAt:  ts,
			TCP22OK:      tcp22 == 1,
			TCP443OK:     tcp443 == 1,
			HTTPOK:       httpOK == 1,
			SSHBannerMS:  sshBannerMS,
			Ports:        ports,
			HTTPChecks:   checks,
		})
	}
	return probes, rows.Err()
}

func (s *Store) listAlertDeliveries(ctx context.Context) ([]model.AlertDelivery, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT incident_id, channel, delivery_key, status, response, created_at, sent_at
		FROM alert_deliveries
		ORDER BY created_at DESC
		LIMIT 300`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliveries []model.AlertDelivery
	for rows.Next() {
		var delivery model.AlertDelivery
		var createdAt string
		var sentAt sql.NullString
		if err := rows.Scan(&delivery.IncidentID, &delivery.Channel, &delivery.DeliveryKey, &delivery.Status, &delivery.Response, &createdAt, &sentAt); err != nil {
			return nil, err
		}
		ts, err := time.Parse(timeLayout, createdAt)
		if err != nil {
			return nil, err
		}
		delivery.CreatedAt = ts
		if sentAt.Valid {
			parsed, err := time.Parse(timeLayout, sentAt.String)
			if err != nil {
				return nil, err
			}
			delivery.SentAt = &parsed
		}
		deliveries = append(deliveries, delivery)
	}
	return deliveries, rows.Err()
}

func (s *Store) snapshotHeartbeats(ctx context.Context) ([]model.NodeHeartbeat, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT node_id, collected_at, cpu_pct, mem_pct, disk_pct, load1, uptime_s, services_json, docker_json, http_json
		FROM metric_samples ORDER BY collected_at DESC LIMIT 300`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var heartbeats []model.NodeHeartbeat
	for rows.Next() {
		hb, err := scanHeartbeat(rows)
		if err != nil {
			return nil, err
		}
		heartbeats = append(heartbeats, *hb)
	}
	return heartbeats, rows.Err()
}

func scanNodeState(scanner interface{ Scan(dest ...any) error }) (*model.NodeState, error) {
	var state model.NodeState
	var lastHeartbeatAt, summaryJSON, servicesJSON, evaluatedAt, evidenceJSON string
	var replicatedFresh int
	if err := scanner.Scan(
		&state.NodeID, &state.Status, &state.Reason, &state.RuleKey, &lastHeartbeatAt, &summaryJSON, &replicatedFresh,
		&state.CPUPct, &state.MemPct, &state.DiskPct, &state.Load1, &state.UptimeS, &servicesJSON,
		&state.BadStreak, &state.GoodStreak, &evaluatedAt, &evidenceJSON,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	var err error
	state.LastHeartbeatAt, err = time.Parse(timeLayout, lastHeartbeatAt)
	if err != nil {
		return nil, err
	}
	state.LastEvaluatedAt, err = time.Parse(timeLayout, evaluatedAt)
	if err != nil {
		return nil, err
	}
	state.ReplicatedFresh = replicatedFresh == 1
	if err := json.Unmarshal([]byte(summaryJSON), &state.LastProbeSummary); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(servicesJSON), &state.Services); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(evidenceJSON), &state.PrimaryEvidence); err != nil {
		return nil, err
	}
	return &state, nil
}

func scanHeartbeat(scanner interface{ Scan(dest ...any) error }) (*model.NodeHeartbeat, error) {
	var hb model.NodeHeartbeat
	var collectedAt, servicesJSON, dockerJSON, httpJSON string
	if err := scanner.Scan(&hb.NodeID, &collectedAt, &hb.CPUPct, &hb.MemPct, &hb.DiskPct, &hb.Load1, &hb.UptimeS, &servicesJSON, &dockerJSON, &httpJSON); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	var err error
	hb.CollectedAt, err = time.Parse(timeLayout, collectedAt)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(servicesJSON), &hb.Services); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(dockerJSON), &hb.DockerChecks); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(httpJSON), &hb.LocalHTTPChecks); err != nil {
		return nil, err
	}
	return &hb, nil
}

func scanIncident(scanner interface{ Scan(dest ...any) error }) (*model.Incident, error) {
	var inc model.Incident
	var openedAt string
	var resolvedAt sql.NullString
	var lastNotifiedAt sql.NullString
	if err := scanner.Scan(&inc.ID, &inc.NodeID, &inc.RuleKey, &inc.Severity, &inc.Status, &inc.Summary, &inc.Detail, &openedAt, &resolvedAt, &lastNotifiedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	var err error
	inc.OpenedAt, err = time.Parse(timeLayout, openedAt)
	if err != nil {
		return nil, err
	}
	if resolvedAt.Valid {
		parsed, err := time.Parse(timeLayout, resolvedAt.String)
		if err != nil {
			return nil, err
		}
		inc.ResolvedAt = &parsed
	}
	if lastNotifiedAt.Valid {
		parsed, err := time.Parse(timeLayout, lastNotifiedAt.String)
		if err != nil {
			return nil, err
		}
		inc.LastNotifiedAt = &parsed
	}
	return &inc, nil
}

func marshalJSON(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
