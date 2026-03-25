package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"vps-monitor/internal/model"
)

type monitorCheckConfig struct {
	ScopeMode     string   `json:"scope_mode,omitempty"`
	NodeIDs       []string `json:"node_ids,omitempty"`
	ServiceName   string   `json:"service_name,omitempty"`
	ContainerName string   `json:"container_name,omitempty"`
	Scheme        string   `json:"scheme,omitempty"`
	HostMode      string   `json:"host_mode,omitempty"`
	Port          int      `json:"port,omitempty"`
	Path          string   `json:"path,omitempty"`
	ExpectStatus  int      `json:"expect_status,omitempty"`
	Timeout       string   `json:"timeout,omitempty"`
	Label         string   `json:"label,omitempty"`
}

func (s *Store) GetAdminSettings(ctx context.Context) (*model.AdminSettings, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT password_hash, initialized_at, updated_at
		FROM admin_settings
		WHERE singleton = 1`)
	var settings model.AdminSettings
	var initializedAt, updatedAt string
	err := row.Scan(&settings.PasswordHash, &initializedAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var parseErr error
	settings.InitializedAt, parseErr = time.Parse(timeLayout, initializedAt)
	if parseErr != nil {
		return nil, parseErr
	}
	settings.UpdatedAt, parseErr = time.Parse(timeLayout, updatedAt)
	if parseErr != nil {
		return nil, parseErr
	}
	return &settings, nil
}

func (s *Store) UpsertAdminSettings(ctx context.Context, settings model.AdminSettings) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO admin_settings (singleton, password_hash, initialized_at, updated_at)
		VALUES (1, ?, ?, ?)
		ON CONFLICT(singleton) DO UPDATE SET
			password_hash = excluded.password_hash,
			initialized_at = excluded.initialized_at,
			updated_at = excluded.updated_at`,
		settings.PasswordHash, settings.InitializedAt.Format(timeLayout), settings.UpdatedAt.Format(timeLayout),
	)
	return err
}

func (s *Store) GetAdminSession(ctx context.Context, sessionID string) (*model.AdminSession, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT session_id, created_at, expires_at
		FROM admin_sessions
		WHERE session_id = ?`, sessionID)
	return scanAdminSession(row)
}

func (s *Store) ListAdminSessions(ctx context.Context) ([]model.AdminSession, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT session_id, created_at, expires_at
		FROM admin_sessions
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []model.AdminSession
	for rows.Next() {
		session, err := scanAdminSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, *session)
	}
	return sessions, rows.Err()
}

func (s *Store) UpsertAdminSession(ctx context.Context, session model.AdminSession) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO admin_sessions (session_id, created_at, expires_at)
		VALUES (?, ?, ?)
		ON CONFLICT(session_id) DO UPDATE SET
			created_at = excluded.created_at,
			expires_at = excluded.expires_at`,
		session.ID, session.CreatedAt.Format(timeLayout), session.ExpiresAt.Format(timeLayout),
	)
	return err
}

func (s *Store) DeleteAdminSession(ctx context.Context, sessionID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM admin_sessions WHERE session_id = ?`, sessionID)
	return err
}

func (s *Store) DeleteExpiredAdminSessions(ctx context.Context, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM admin_sessions WHERE expires_at < ?`, now.Format(timeLayout))
	return err
}

func (s *Store) CountMonitorChecks(ctx context.Context) (int, error) {
	row := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM monitor_checks`)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Store) MaxMonitorCheckSortOrder(ctx context.Context) (int, error) {
	row := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(sort_order), 0) FROM monitor_checks`)
	var max int
	if err := row.Scan(&max); err != nil {
		return 0, err
	}
	return max, nil
}

func (s *Store) ListMonitorChecks(ctx context.Context) ([]model.MonitorCheck, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, type, name, enabled, sort_order, config_json, created_at, updated_at
		FROM monitor_checks
		ORDER BY sort_order ASC, created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []model.MonitorCheck
	for rows.Next() {
		check, err := scanMonitorCheck(rows)
		if err != nil {
			return nil, err
		}
		checks = append(checks, *check)
	}
	return checks, rows.Err()
}

func (s *Store) UpsertMonitorCheck(ctx context.Context, check model.MonitorCheck) error {
	configJSON, err := marshalMonitorCheckConfig(check)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO monitor_checks (id, type, name, enabled, sort_order, config_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			type = excluded.type,
			name = excluded.name,
			enabled = excluded.enabled,
			sort_order = excluded.sort_order,
			config_json = excluded.config_json,
			updated_at = excluded.updated_at`,
		check.ID, check.Type, check.Name, boolToInt(check.Enabled), check.SortOrder, configJSON, check.CreatedAt.Format(timeLayout), check.UpdatedAt.Format(timeLayout),
	)
	return err
}

func (s *Store) DeleteMonitorCheck(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM monitor_checks WHERE id = ?`, id)
	return err
}

func scanAdminSession(scanner interface{ Scan(dest ...any) error }) (*model.AdminSession, error) {
	var session model.AdminSession
	var createdAt, expiresAt string
	if err := scanner.Scan(&session.ID, &createdAt, &expiresAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	var parseErr error
	session.CreatedAt, parseErr = time.Parse(timeLayout, createdAt)
	if parseErr != nil {
		return nil, parseErr
	}
	session.ExpiresAt, parseErr = time.Parse(timeLayout, expiresAt)
	if parseErr != nil {
		return nil, parseErr
	}
	return &session, nil
}

func scanMonitorCheck(scanner interface{ Scan(dest ...any) error }) (*model.MonitorCheck, error) {
	var check model.MonitorCheck
	var enabled int
	var configJSON, createdAt, updatedAt string
	if err := scanner.Scan(&check.ID, &check.Type, &check.Name, &enabled, &check.SortOrder, &configJSON, &createdAt, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	var cfg monitorCheckConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, err
	}
	check.Enabled = enabled == 1
	check.ScopeMode = cfg.ScopeMode
	check.NodeIDs = append([]string(nil), cfg.NodeIDs...)
	check.ServiceName = cfg.ServiceName
	check.ContainerName = cfg.ContainerName
	check.Scheme = cfg.Scheme
	check.HostMode = cfg.HostMode
	check.Port = cfg.Port
	check.Path = cfg.Path
	check.ExpectStatus = cfg.ExpectStatus
	check.Timeout = cfg.Timeout
	check.Label = cfg.Label
	var parseErr error
	check.CreatedAt, parseErr = time.Parse(timeLayout, createdAt)
	if parseErr != nil {
		return nil, parseErr
	}
	check.UpdatedAt, parseErr = time.Parse(timeLayout, updatedAt)
	if parseErr != nil {
		return nil, parseErr
	}
	normalized := check.Normalize()
	check.Scheme = normalized.Scheme
	check.HostMode = normalized.HostMode
	check.Path = normalized.Path
	return &check, nil
}

func marshalMonitorCheckConfig(check model.MonitorCheck) (string, error) {
	raw, err := json.Marshal(monitorCheckConfig{
		ScopeMode:     check.ScopeMode,
		NodeIDs:       check.NodeIDs,
		ServiceName:   check.ServiceName,
		ContainerName: check.ContainerName,
		Scheme:        check.Scheme,
		HostMode:      check.HostMode,
		Port:          check.Port,
		Path:          check.Path,
		ExpectStatus:  check.ExpectStatus,
		Timeout:       check.Timeout,
		Label:         check.Label,
	})
	if err != nil {
		return "", fmt.Errorf("marshal monitor check config: %w", err)
	}
	return string(raw), nil
}
