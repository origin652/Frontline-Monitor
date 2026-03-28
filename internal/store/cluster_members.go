package store

import (
	"context"
	"database/sql"
	"time"

	"vps-monitor/internal/model"
)

func (s *Store) CountClusterMembers(ctx context.Context) (int, error) {
	row := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM cluster_members`)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Store) GetClusterMember(ctx context.Context, nodeID string) (*model.ClusterMember, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT node_id, display_name, api_addr, raft_addr, public_ipv4, priority, ingress_candidate, desired_role,
			status, joined_at, updated_at, removed_at
		FROM cluster_members
		WHERE node_id = ?`, nodeID)
	return scanClusterMember(row)
}

func (s *Store) ListClusterMembers(ctx context.Context) ([]model.ClusterMember, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT node_id, display_name, api_addr, raft_addr, public_ipv4, priority, ingress_candidate, desired_role,
			status, joined_at, updated_at, removed_at
		FROM cluster_members
		ORDER BY node_id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.ClusterMember
	for rows.Next() {
		item, err := scanClusterMember(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (s *Store) UpsertClusterMember(ctx context.Context, member model.ClusterMember) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO cluster_members (
			node_id, display_name, api_addr, raft_addr, public_ipv4, priority, ingress_candidate, desired_role,
			status, joined_at, updated_at, removed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(node_id) DO UPDATE SET
			display_name = excluded.display_name,
			api_addr = excluded.api_addr,
			raft_addr = excluded.raft_addr,
			public_ipv4 = excluded.public_ipv4,
			priority = excluded.priority,
			ingress_candidate = excluded.ingress_candidate,
			desired_role = excluded.desired_role,
			status = excluded.status,
			joined_at = excluded.joined_at,
			updated_at = excluded.updated_at,
			removed_at = excluded.removed_at`,
		member.NodeID,
		member.DisplayName,
		member.APIAddr,
		member.RaftAddr,
		member.PublicIPv4,
		member.Priority,
		boolPointerToDB(member.IngressCandidate),
		member.DesiredRole,
		member.Status,
		member.JoinedAt.Format(timeLayout),
		member.UpdatedAt.Format(timeLayout),
		nullTimeString(member.RemovedAt),
	)
	return err
}

func scanClusterMember(scanner interface{ Scan(dest ...any) error }) (*model.ClusterMember, error) {
	var member model.ClusterMember
	var ingressCandidate sql.NullInt64
	var joinedAt, updatedAt string
	var removedAt sql.NullString
	if err := scanner.Scan(
		&member.NodeID,
		&member.DisplayName,
		&member.APIAddr,
		&member.RaftAddr,
		&member.PublicIPv4,
		&member.Priority,
		&ingressCandidate,
		&member.DesiredRole,
		&member.Status,
		&joinedAt,
		&updatedAt,
		&removedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if ingressCandidate.Valid {
		value := ingressCandidate.Int64 == 1
		member.IngressCandidate = &value
	}
	parsedJoinedAt, err := time.Parse(timeLayout, joinedAt)
	if err != nil {
		return nil, err
	}
	member.JoinedAt = parsedJoinedAt
	parsedUpdatedAt, err := time.Parse(timeLayout, updatedAt)
	if err != nil {
		return nil, err
	}
	member.UpdatedAt = parsedUpdatedAt
	if removedAt.Valid {
		parsedRemovedAt, err := time.Parse(timeLayout, removedAt.String)
		if err != nil {
			return nil, err
		}
		member.RemovedAt = &parsedRemovedAt
	}
	member.DesiredRole = model.NormalizeClusterMemberRole(member.DesiredRole)
	member.Status = model.NormalizeClusterMemberStatus(member.Status)
	return &member, nil
}

func boolPointerToDB(value *bool) any {
	if value == nil {
		return nil
	}
	if *value {
		return 1
	}
	return 0
}

func nullTimeString(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.Format(timeLayout)
}
