package store

import (
	"context"
	"database/sql"
	"time"

	"vps-monitor/internal/model"
)

func (s *Store) ListNodeDisplayNames(ctx context.Context) ([]model.NodeDisplayName, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT node_id, display_name, updated_at
		FROM node_display_names
		ORDER BY node_id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.NodeDisplayName
	for rows.Next() {
		item, err := scanNodeDisplayName(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (s *Store) UpsertNodeDisplayName(ctx context.Context, item model.NodeDisplayName) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO node_display_names (node_id, display_name, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(node_id) DO UPDATE SET
			display_name = excluded.display_name,
			updated_at = excluded.updated_at`,
		item.NodeID, item.DisplayName, item.UpdatedAt.Format(timeLayout),
	)
	return err
}

func (s *Store) DeleteNodeDisplayName(ctx context.Context, nodeID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM node_display_names WHERE node_id = ?`, nodeID)
	return err
}

func scanNodeDisplayName(scanner interface{ Scan(dest ...any) error }) (*model.NodeDisplayName, error) {
	var item model.NodeDisplayName
	var updatedAt string
	if err := scanner.Scan(&item.NodeID, &item.DisplayName, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	parsed, err := time.Parse(timeLayout, updatedAt)
	if err != nil {
		return nil, err
	}
	item.UpdatedAt = parsed
	return &item, nil
}
