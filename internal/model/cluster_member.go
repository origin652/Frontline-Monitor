package model

import (
	"strings"
	"time"
)

const (
	ClusterMemberRoleVoter    = "voter"
	ClusterMemberRoleNonvoter = "nonvoter"
)

const (
	ClusterMemberStatusActive  = "active"
	ClusterMemberStatusRemoved = "removed"
)

type ClusterMember struct {
	NodeID           string     `json:"node_id"`
	DisplayName      string     `json:"display_name,omitempty"`
	APIAddr          string     `json:"api_addr"`
	RaftAddr         string     `json:"raft_addr"`
	PublicIPv4       string     `json:"public_ipv4"`
	Priority         int        `json:"priority"`
	IngressCandidate *bool      `json:"ingress_candidate,omitempty"`
	DesiredRole      string     `json:"desired_role"`
	Status           string     `json:"status"`
	JoinedAt         time.Time  `json:"joined_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	RemovedAt        *time.Time `json:"removed_at,omitempty"`
}

type ClusterMemberView struct {
	NodeID               string     `json:"node_id"`
	DisplayName          string     `json:"display_name,omitempty"`
	EffectiveDisplayName string     `json:"effective_display_name"`
	APIAddr              string     `json:"api_addr"`
	RaftAddr             string     `json:"raft_addr"`
	PublicIPv4           string     `json:"public_ipv4"`
	Priority             int        `json:"priority"`
	IngressCandidate     bool       `json:"ingress_candidate"`
	DesiredRole          string     `json:"desired_role"`
	CurrentRole          string     `json:"current_role,omitempty"`
	Status               string     `json:"status"`
	IsLeader             bool       `json:"is_leader"`
	LastHeartbeatAt      *time.Time `json:"last_heartbeat_at,omitempty"`
	HealthStatus         string     `json:"health_status,omitempty"`
	HealthReason         string     `json:"health_reason,omitempty"`
	JoinedAt             time.Time  `json:"joined_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	RemovedAt            *time.Time `json:"removed_at,omitempty"`
}

func NormalizeClusterMemberRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "", ClusterMemberRoleVoter:
		return ClusterMemberRoleVoter
	case ClusterMemberRoleNonvoter:
		return ClusterMemberRoleNonvoter
	default:
		return strings.ToLower(strings.TrimSpace(role))
	}
}

func IsValidClusterMemberRole(role string) bool {
	switch NormalizeClusterMemberRole(role) {
	case ClusterMemberRoleVoter, ClusterMemberRoleNonvoter:
		return true
	default:
		return false
	}
}

func NormalizeClusterMemberStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", ClusterMemberStatusActive:
		return ClusterMemberStatusActive
	case ClusterMemberStatusRemoved:
		return ClusterMemberStatusRemoved
	default:
		return strings.ToLower(strings.TrimSpace(status))
	}
}

func (m ClusterMember) EffectiveName() string {
	if strings.TrimSpace(m.DisplayName) != "" {
		return strings.TrimSpace(m.DisplayName)
	}
	return strings.TrimSpace(m.NodeID)
}

func (m ClusterMember) IsIngressCandidate() bool {
	if m.IngressCandidate == nil {
		return true
	}
	return *m.IngressCandidate
}

func (m ClusterMember) IsActive() bool {
	return NormalizeClusterMemberStatus(m.Status) == ClusterMemberStatusActive
}
