package cluster

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/raft"

	"vps-monitor/internal/config"
	"vps-monitor/internal/model"
)

var (
	ErrMemberConflict             = errors.New("cluster member conflict")
	ErrMemberNotFound             = errors.New("cluster member not found")
	ErrInvalidMemberRole          = errors.New("invalid cluster member role")
	ErrLastVoterRemoval           = errors.New("cannot remove the last voter")
	ErrNoHealthyTransferCandidate = errors.New("no healthy voter available for leadership transfer")
)

func (m *Manager) HasExistingState() bool {
	return m.hasState
}

func (m *Manager) NeedsJoin() bool {
	return m.cfg.UsesDynamicMembership() && !m.hasState && !m.cfg.Cluster.Bootstrap
}

func (m *Manager) SelfMember() model.ClusterMember {
	now := time.Now().UTC()
	return model.ClusterMember{
		NodeID:           strings.TrimSpace(m.cfg.Cluster.NodeID),
		DisplayName:      strings.TrimSpace(m.cfg.DefaultDisplayName()),
		APIAddr:          strings.TrimSpace(m.cfg.APIAddr()),
		RaftAddr:         strings.TrimSpace(m.cfg.Cluster.RaftAddr),
		PublicIPv4:       strings.TrimSpace(m.cfg.Network.PublicIPv4),
		Priority:         m.cfg.Cluster.Priority,
		IngressCandidate: m.cfg.Cluster.IngressCandidate,
		DesiredRole:      m.cfg.NormalizedRole(),
		Status:           model.ClusterMemberStatusActive,
		JoinedAt:         now,
		UpdatedAt:        now,
	}
}

func (m *Manager) ListMembers(ctx context.Context) ([]model.ClusterMember, error) {
	members, err := m.store.ListClusterMembers(ctx)
	if err != nil {
		return nil, err
	}
	if len(members) > 0 {
		return members, nil
	}

	switch {
	case m.cfg.UsesStaticPeers():
		out := make([]model.ClusterMember, 0, len(m.cfg.Cluster.Peers))
		now := time.Now().UTC()
		for _, peer := range m.cfg.Cluster.Peers {
			out = append(out, clusterMemberFromPeer(peer, now))
		}
		return out, nil
	case m.cfg.Cluster.Bootstrap || m.hasState:
		return []model.ClusterMember{m.SelfMember()}, nil
	default:
		return nil, nil
	}
}

func (m *Manager) ActiveMembers(ctx context.Context) ([]model.ClusterMember, error) {
	members, err := m.ListMembers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]model.ClusterMember, 0, len(members))
	for _, member := range members {
		if member.IsActive() {
			out = append(out, member)
		}
	}
	return out, nil
}

func (m *Manager) OrderedMembers(ctx context.Context) ([]model.ClusterMember, error) {
	members, err := m.ActiveMembers(ctx)
	if err != nil {
		return nil, err
	}
	sortClusterMembers(members)
	return members, nil
}

func (m *Manager) MemberByID(ctx context.Context, nodeID string) (model.ClusterMember, bool, error) {
	members, err := m.ListMembers(ctx)
	if err != nil {
		return model.ClusterMember{}, false, err
	}
	for _, member := range members {
		if strings.TrimSpace(member.NodeID) == strings.TrimSpace(nodeID) {
			return member, true, nil
		}
	}
	return model.ClusterMember{}, false, nil
}

func (m *Manager) LeaderID() string {
	_, leaderID := m.raft.LeaderWithID()
	if leaderID != "" {
		return string(leaderID)
	}

	leaderAddr := string(m.raft.Leader())
	if leaderAddr == "" {
		return ""
	}
	members, err := m.ListMembers(context.Background())
	if err == nil {
		for _, member := range members {
			if strings.TrimSpace(member.RaftAddr) == strings.TrimSpace(leaderAddr) {
				return member.NodeID
			}
		}
	}
	for _, peer := range m.cfg.Cluster.Peers {
		if peer.RaftAddr == leaderAddr {
			return peer.NodeID
		}
	}
	if strings.TrimSpace(m.cfg.Cluster.RaftAddr) == strings.TrimSpace(leaderAddr) {
		return m.cfg.Cluster.NodeID
	}
	return ""
}

func (m *Manager) LeaderAPIAddr() string {
	leaderID := m.LeaderID()
	if leaderID == "" {
		return ""
	}
	member, ok, err := m.MemberByID(context.Background(), leaderID)
	if err == nil && ok && strings.TrimSpace(member.APIAddr) != "" {
		return strings.TrimSpace(member.APIAddr)
	}
	return m.cfg.LeaderAPIAddr(leaderID)
}

func (m *Manager) CurrentRoleMap() (map[string]string, error) {
	future := m.raft.GetConfiguration()
	if err := future.Error(); err != nil {
		return nil, err
	}
	out := make(map[string]string, len(future.Configuration().Servers))
	for _, server := range future.Configuration().Servers {
		out[string(server.ID)] = raftSuffrageToRole(server.Suffrage)
	}
	return out, nil
}

func (m *Manager) EnsureMemberDirectorySeeded(ctx context.Context, now time.Time) error {
	if !m.IsLeader() {
		return nil
	}
	count, err := m.store.CountClusterMembers(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	switch {
	case m.cfg.UsesStaticPeers():
		for _, peer := range m.cfg.Cluster.Peers {
			member := clusterMemberFromPeer(peer, now)
			if _, err := m.Apply(ctx, CommandClusterMember, member); err != nil {
				return err
			}
		}
	case m.cfg.UsesDynamicMembership():
		member := m.SelfMember()
		member.JoinedAt = now
		member.UpdatedAt = now
		if _, err := m.Apply(ctx, CommandClusterMember, member); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) JoinMember(ctx context.Context, candidate model.ClusterMember) (model.ClusterMember, error) {
	if !m.IsLeader() {
		return model.ClusterMember{}, raft.ErrNotLeader
	}

	now := time.Now().UTC()
	if err := m.EnsureMemberDirectorySeeded(ctx, now); err != nil {
		return model.ClusterMember{}, err
	}

	member, previous, err := m.normalizeJoinCandidate(ctx, candidate, now)
	if err != nil {
		return model.ClusterMember{}, err
	}

	if _, err := m.Apply(ctx, CommandClusterMember, member); err != nil {
		return model.ClusterMember{}, err
	}
	if err := m.ensureRaftMembership(ctx, member); err != nil {
		if rollbackErr := m.rollbackMemberRecord(ctx, previous, member, now); rollbackErr != nil {
			m.logger.Error("rollback cluster member after join failure failed", "node_id", member.NodeID, "error", rollbackErr)
		}
		return model.ClusterMember{}, err
	}

	if err := m.recordMembershipEvent(ctx, "member_joined", member.NodeID, fmt.Sprintf("%s joined cluster", member.NodeID), map[string]any{
		"api_addr":     member.APIAddr,
		"raft_addr":    member.RaftAddr,
		"desired_role": member.DesiredRole,
	}); err != nil {
		m.logger.Warn("record member joined event failed", "node_id", member.NodeID, "error", err)
	}

	return member, nil
}

func (m *Manager) UpdateMemberRole(ctx context.Context, nodeID, role string) (model.ClusterMember, error) {
	if !m.IsLeader() {
		return model.ClusterMember{}, raft.ErrNotLeader
	}

	role = model.NormalizeClusterMemberRole(role)
	if !model.IsValidClusterMemberRole(role) {
		return model.ClusterMember{}, fmt.Errorf("%w: %s", ErrInvalidMemberRole, role)
	}

	now := time.Now().UTC()
	if err := m.EnsureMemberDirectorySeeded(ctx, now); err != nil {
		return model.ClusterMember{}, err
	}

	member, ok, err := m.MemberByID(ctx, nodeID)
	if err != nil {
		return model.ClusterMember{}, err
	}
	if !ok || !member.IsActive() {
		return model.ClusterMember{}, fmt.Errorf("%w: %s", ErrMemberNotFound, nodeID)
	}

	member.DesiredRole = role
	member.UpdatedAt = now

	roleMap, err := m.CurrentRoleMap()
	if err != nil {
		return model.ClusterMember{}, err
	}
	if roleMap[nodeID] == model.ClusterMemberRoleVoter && role == model.ClusterMemberRoleNonvoter && countRole(roleMap, model.ClusterMemberRoleVoter) <= 1 {
		return model.ClusterMember{}, ErrLastVoterRemoval
	}

	if err := m.ensureRaftMembership(ctx, member); err != nil {
		return model.ClusterMember{}, err
	}
	if _, err := m.Apply(ctx, CommandClusterMember, member); err != nil {
		return model.ClusterMember{}, err
	}

	if err := m.recordMembershipEvent(ctx, "member_role_changed", member.NodeID, fmt.Sprintf("%s role updated", member.NodeID), map[string]any{
		"desired_role": role,
	}); err != nil {
		m.logger.Warn("record member role event failed", "node_id", member.NodeID, "error", err)
	}

	return member, nil
}

func (m *Manager) RemoveMember(ctx context.Context, nodeID string) (model.ClusterMember, error) {
	if !m.IsLeader() {
		return model.ClusterMember{}, raft.ErrNotLeader
	}

	now := time.Now().UTC()
	if err := m.EnsureMemberDirectorySeeded(ctx, now); err != nil {
		return model.ClusterMember{}, err
	}

	member, ok, err := m.MemberByID(ctx, nodeID)
	if err != nil {
		return model.ClusterMember{}, err
	}
	if !ok || !member.IsActive() {
		return model.ClusterMember{}, fmt.Errorf("%w: %s", ErrMemberNotFound, nodeID)
	}

	roleMap, err := m.CurrentRoleMap()
	if err != nil {
		return model.ClusterMember{}, err
	}
	if roleMap[nodeID] == model.ClusterMemberRoleVoter && countRole(roleMap, model.ClusterMemberRoleVoter) <= 1 {
		return model.ClusterMember{}, ErrLastVoterRemoval
	}

	if currentRole, exists := roleMap[nodeID]; exists {
		if err := m.removeServer(ctx, nodeID, currentRole); err != nil {
			return model.ClusterMember{}, err
		}
	}

	member.Status = model.ClusterMemberStatusRemoved
	member.RemovedAt = &now
	member.UpdatedAt = now
	if _, err := m.Apply(ctx, CommandClusterMember, member); err != nil {
		return model.ClusterMember{}, err
	}

	if err := m.recordMembershipEvent(ctx, "member_removed", member.NodeID, fmt.Sprintf("%s removed from cluster", member.NodeID), map[string]any{
		"desired_role": member.DesiredRole,
	}); err != nil {
		m.logger.Warn("record member removed event failed", "node_id", member.NodeID, "error", err)
	}

	return member, nil
}

func (m *Manager) SelectLeadershipTransferTarget(ctx context.Context, excludeNodeID string) (model.ClusterMember, error) {
	roleMap, err := m.CurrentRoleMap()
	if err != nil {
		return model.ClusterMember{}, err
	}
	members, err := m.ActiveMembers(ctx)
	if err != nil {
		return model.ClusterMember{}, err
	}
	states, err := m.store.ListNodeStates(ctx)
	if err != nil {
		return model.ClusterMember{}, err
	}
	stateMap := make(map[string]model.NodeState, len(states))
	for _, state := range states {
		stateMap[state.NodeID] = state
	}

	candidates := make([]model.ClusterMember, 0, len(members))
	for _, member := range members {
		if member.NodeID == excludeNodeID {
			continue
		}
		if roleMap[member.NodeID] != model.ClusterMemberRoleVoter {
			continue
		}
		state, ok := stateMap[member.NodeID]
		if !ok {
			continue
		}
		if state.Status != model.StatusHealthy || !state.ReplicatedFresh {
			continue
		}
		candidates = append(candidates, member)
	}
	if len(candidates) == 0 {
		return model.ClusterMember{}, ErrNoHealthyTransferCandidate
	}
	sortClusterMembers(candidates)
	return candidates[0], nil
}

func (m *Manager) TransferLeadershipTo(ctx context.Context, member model.ClusterMember) error {
	future := m.raft.LeadershipTransferToServer(raft.ServerID(member.NodeID), raft.ServerAddress(member.RaftAddr))
	done := make(chan error, 1)
	go func() {
		done <- future.Error()
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

func (m *Manager) normalizeJoinCandidate(ctx context.Context, candidate model.ClusterMember, now time.Time) (model.ClusterMember, *model.ClusterMember, error) {
	member := candidate
	member.NodeID = strings.TrimSpace(member.NodeID)
	member.DisplayName = strings.TrimSpace(member.DisplayName)
	member.APIAddr = strings.TrimSpace(member.APIAddr)
	member.RaftAddr = strings.TrimSpace(member.RaftAddr)
	member.PublicIPv4 = strings.TrimSpace(member.PublicIPv4)
	member.DesiredRole = model.NormalizeClusterMemberRole(member.DesiredRole)
	member.Status = model.ClusterMemberStatusActive
	member.RemovedAt = nil
	member.UpdatedAt = now

	if member.NodeID == "" || member.APIAddr == "" || member.RaftAddr == "" || member.PublicIPv4 == "" {
		return model.ClusterMember{}, nil, fmt.Errorf("node_id, api_addr, raft_addr, and public_ipv4 are required")
	}
	if !model.IsValidClusterMemberRole(member.DesiredRole) {
		return model.ClusterMember{}, nil, fmt.Errorf("%w: %s", ErrInvalidMemberRole, member.DesiredRole)
	}

	members, err := m.ListMembers(ctx)
	if err != nil {
		return model.ClusterMember{}, nil, err
	}

	var previous *model.ClusterMember
	for _, existing := range members {
		if existing.NodeID == member.NodeID {
			copy := existing
			previous = &copy
			if existing.IsActive() {
				if existing.APIAddr != member.APIAddr || existing.RaftAddr != member.RaftAddr || existing.PublicIPv4 != member.PublicIPv4 {
					return model.ClusterMember{}, nil, fmt.Errorf("%w: active node_id %s already exists with different addresses", ErrMemberConflict, member.NodeID)
				}
				member.JoinedAt = existing.JoinedAt
			} else {
				member.JoinedAt = now
			}
			continue
		}
		if existing.IsActive() && existing.APIAddr == member.APIAddr {
			return model.ClusterMember{}, nil, fmt.Errorf("%w: api_addr %s is already used by %s", ErrMemberConflict, member.APIAddr, existing.NodeID)
		}
		if existing.IsActive() && existing.RaftAddr == member.RaftAddr {
			return model.ClusterMember{}, nil, fmt.Errorf("%w: raft_addr %s is already used by %s", ErrMemberConflict, member.RaftAddr, existing.NodeID)
		}
	}
	if member.JoinedAt.IsZero() {
		member.JoinedAt = now
	}
	return member, previous, nil
}

func (m *Manager) ensureRaftMembership(ctx context.Context, member model.ClusterMember) error {
	future := m.raft.GetConfiguration()
	if err := future.Error(); err != nil {
		return err
	}

	for _, server := range future.Configuration().Servers {
		if string(server.ID) == member.NodeID {
			if string(server.Address) != member.RaftAddr {
				return fmt.Errorf("%w: raft server %s already exists at %s", ErrMemberConflict, member.NodeID, server.Address)
			}
			currentRole := raftSuffrageToRole(server.Suffrage)
			if currentRole == member.DesiredRole {
				return nil
			}
			if member.DesiredRole == model.ClusterMemberRoleVoter {
				return m.raft.AddVoter(raft.ServerID(member.NodeID), raft.ServerAddress(member.RaftAddr), 0, 10*time.Second).Error()
			}
			if currentRole == model.ClusterMemberRoleVoter {
				return m.raft.DemoteVoter(raft.ServerID(member.NodeID), 0, 10*time.Second).Error()
			}
			return m.raft.AddNonvoter(raft.ServerID(member.NodeID), raft.ServerAddress(member.RaftAddr), 0, 10*time.Second).Error()
		}
	}

	if member.DesiredRole == model.ClusterMemberRoleVoter {
		return m.raft.AddVoter(raft.ServerID(member.NodeID), raft.ServerAddress(member.RaftAddr), 0, 10*time.Second).Error()
	}
	return m.raft.AddNonvoter(raft.ServerID(member.NodeID), raft.ServerAddress(member.RaftAddr), 0, 10*time.Second).Error()
}

func (m *Manager) rollbackMemberRecord(ctx context.Context, previous *model.ClusterMember, member model.ClusterMember, now time.Time) error {
	if previous != nil {
		_, err := m.Apply(ctx, CommandClusterMember, *previous)
		return err
	}
	member.Status = model.ClusterMemberStatusRemoved
	member.RemovedAt = &now
	member.UpdatedAt = now
	_, err := m.Apply(ctx, CommandClusterMember, member)
	return err
}

func (m *Manager) removeServer(ctx context.Context, nodeID, role string) error {
	if role == model.ClusterMemberRoleVoter && countRoleMust(m, role) <= 1 {
		return ErrLastVoterRemoval
	}
	future := m.raft.RemoveServer(raft.ServerID(nodeID), 0, 10*time.Second)
	done := make(chan error, 1)
	go func() {
		done <- future.Error()
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

func (m *Manager) recordMembershipEvent(ctx context.Context, kind, nodeID, title string, meta map[string]any) error {
	_, err := m.Apply(ctx, CommandEvent, model.Event{
		ID:        uuid.NewString(),
		Kind:      kind,
		Severity:  model.StatusHealthy,
		NodeID:    nodeID,
		Title:     title,
		Body:      title,
		CreatedAt: time.Now().UTC(),
		Meta:      meta,
	})
	return err
}

func clusterMemberFromPeer(peer config.ClusterPeer, now time.Time) model.ClusterMember {
	return model.ClusterMember{
		NodeID:           strings.TrimSpace(peer.NodeID),
		DisplayName:      strings.TrimSpace(peer.DisplayName),
		APIAddr:          strings.TrimSpace(peer.APIAddr),
		RaftAddr:         strings.TrimSpace(peer.RaftAddr),
		PublicIPv4:       strings.TrimSpace(peer.PublicIPv4),
		Priority:         peer.Priority,
		IngressCandidate: peer.IngressCandidate,
		DesiredRole:      model.ClusterMemberRoleVoter,
		Status:           model.ClusterMemberStatusActive,
		JoinedAt:         now,
		UpdatedAt:        now,
	}
}

func sortClusterMembers(members []model.ClusterMember) {
	slices.SortFunc(members, func(a, b model.ClusterMember) int {
		if a.Priority != b.Priority {
			return b.Priority - a.Priority
		}
		return strings.Compare(a.NodeID, b.NodeID)
	})
}

func raftSuffrageToRole(suffrage raft.ServerSuffrage) string {
	if suffrage == raft.Nonvoter {
		return model.ClusterMemberRoleNonvoter
	}
	return model.ClusterMemberRoleVoter
}

func countRole(roleMap map[string]string, role string) int {
	count := 0
	for _, current := range roleMap {
		if current == role {
			count++
		}
	}
	return count
}

func countRoleMust(m *Manager, role string) int {
	roleMap, err := m.CurrentRoleMap()
	if err != nil {
		return 0
	}
	return countRole(roleMap, role)
}
