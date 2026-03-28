package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/raft"

	"vps-monitor/internal/cluster"
	"vps-monitor/internal/model"
)

type adminMemberRoleRequest struct {
	Role string `json:"role"`
}

func (s *Server) leaderAPIAddrOrError() (string, error) {
	addr := strings.TrimSpace(s.cluster.LeaderAPIAddr())
	if addr == "" {
		return "", fmt.Errorf("cluster leader unavailable, retry later")
	}
	return addr, nil
}

func (s *Server) proxyInternalJSON(w http.ResponseWriter, r *http.Request, method, addr, path string, payload any) bool {
	status, body, err := s.doInternalJSON(r.Context(), method, addr, path, payload)
	if err != nil {
		s.renderError(w, http.StatusServiceUnavailable, err)
		return true
	}
	writeJSONBytes(w, status, body)
	return true
}

func (s *Server) proxyLeaderJSON(w http.ResponseWriter, r *http.Request, method, path string, payload any) bool {
	addr, err := s.leaderAPIAddrOrError()
	if err != nil {
		s.renderError(w, http.StatusServiceUnavailable, err)
		return true
	}
	return s.proxyInternalJSON(w, r, method, addr, path, payload)
}

func (s *Server) handleLeaderSelfRemoval(w http.ResponseWriter, r *http.Request, nodeID string) bool {
	roleMap, err := s.cluster.CurrentRoleMap()
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return true
	}
	if roleMap[nodeID] == model.ClusterMemberRoleVoter {
		voterCount := 0
		for _, role := range roleMap {
			if role == model.ClusterMemberRoleVoter {
				voterCount++
			}
		}
		if voterCount <= 1 {
			s.renderMembershipError(w, cluster.ErrLastVoterRemoval)
			return true
		}
	}

	target, err := s.cluster.SelectLeadershipTransferTarget(r.Context(), nodeID)
	if err != nil {
		s.renderMembershipError(w, err)
		return true
	}
	if err := s.cluster.TransferLeadershipTo(r.Context(), target); err != nil {
		s.renderError(w, http.StatusConflict, err)
		return true
	}
	return s.proxyInternalJSON(w, r, http.MethodDelete, target.APIAddr, "/internal/v1/cluster/members/"+url.PathEscape(nodeID), nil)
}

func (s *Server) handleAdminMembers(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdminSession(w, r); !ok {
		return
	}
	members, err := s.listAdminClusterMembers(r.Context())
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, members)
}

func (s *Server) handleAdminMemberRole(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdminSession(w, r); !ok {
		return
	}
	nodeID := strings.TrimSpace(r.PathValue("nodeID"))
	if nodeID == "" {
		s.renderError(w, http.StatusBadRequest, fmt.Errorf("nodeID is required"))
		return
	}
	defer r.Body.Close()
	var req adminMemberRoleRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		s.renderError(w, http.StatusBadRequest, err)
		return
	}

	if !s.cluster.IsLeader() {
		s.proxyLeaderJSON(w, r, http.MethodPut, "/internal/v1/cluster/members/"+url.PathEscape(nodeID)+"/role", req)
		return
	}

	member, err := s.cluster.UpdateMemberRole(r.Context(), nodeID, req.Role)
	if err != nil {
		s.renderMembershipError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, member)
}

func (s *Server) handleAdminMemberByID(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdminSession(w, r); !ok {
		return
	}
	nodeID := strings.TrimSpace(r.PathValue("nodeID"))
	if nodeID == "" {
		s.renderError(w, http.StatusBadRequest, fmt.Errorf("nodeID is required"))
		return
	}

	if !s.cluster.IsLeader() {
		s.proxyLeaderJSON(w, r, http.MethodDelete, "/internal/v1/cluster/members/"+url.PathEscape(nodeID), nil)
		return
	}

	if nodeID == s.cfg.Cluster.NodeID {
		s.handleLeaderSelfRemoval(w, r, nodeID)
		return
	}

	member, err := s.cluster.RemoveMember(r.Context(), nodeID)
	if err != nil {
		s.renderMembershipError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, member)
}

func (s *Server) handleInternalJoin(w http.ResponseWriter, r *http.Request) {
	if err := s.requireInternalRequest(r); err != nil {
		s.renderError(w, http.StatusForbidden, err)
		return
	}
	defer r.Body.Close()
	var member model.ClusterMember
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&member); err != nil {
		s.renderError(w, http.StatusBadRequest, err)
		return
	}

	if !s.cluster.IsLeader() {
		s.proxyLeaderJSON(w, r, http.MethodPost, "/internal/v1/cluster/join", member)
		return
	}

	joined, err := s.cluster.JoinMember(r.Context(), member)
	if err != nil {
		s.renderMembershipError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, joined)
}

func (s *Server) handleInternalMemberRole(w http.ResponseWriter, r *http.Request) {
	if err := s.requireInternalRequest(r); err != nil {
		s.renderError(w, http.StatusForbidden, err)
		return
	}
	nodeID := strings.TrimSpace(r.PathValue("nodeID"))
	if nodeID == "" {
		s.renderError(w, http.StatusBadRequest, fmt.Errorf("nodeID is required"))
		return
	}
	defer r.Body.Close()
	var req adminMemberRoleRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		s.renderError(w, http.StatusBadRequest, err)
		return
	}

	if !s.cluster.IsLeader() {
		s.proxyLeaderJSON(w, r, http.MethodPut, "/internal/v1/cluster/members/"+url.PathEscape(nodeID)+"/role", req)
		return
	}

	member, err := s.cluster.UpdateMemberRole(r.Context(), nodeID, req.Role)
	if err != nil {
		s.renderMembershipError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, member)
}

func (s *Server) handleInternalMemberByID(w http.ResponseWriter, r *http.Request) {
	if err := s.requireInternalRequest(r); err != nil {
		s.renderError(w, http.StatusForbidden, err)
		return
	}
	nodeID := strings.TrimSpace(r.PathValue("nodeID"))
	if nodeID == "" {
		s.renderError(w, http.StatusBadRequest, fmt.Errorf("nodeID is required"))
		return
	}

	if !s.cluster.IsLeader() {
		s.proxyLeaderJSON(w, r, http.MethodDelete, "/internal/v1/cluster/members/"+url.PathEscape(nodeID), nil)
		return
	}

	if nodeID == s.cfg.Cluster.NodeID {
		s.handleLeaderSelfRemoval(w, r, nodeID)
		return
	}

	member, err := s.cluster.RemoveMember(r.Context(), nodeID)
	if err != nil {
		s.renderMembershipError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, member)
}

func (s *Server) listAdminClusterMembers(ctx context.Context) ([]model.ClusterMemberView, error) {
	resolver, err := s.newNodeNameResolver(ctx)
	if err != nil {
		return nil, err
	}
	members, err := s.cluster.ListMembers(ctx)
	if err != nil {
		return nil, err
	}
	roleMap, err := s.cluster.CurrentRoleMap()
	if err != nil && !errors.Is(err, raft.ErrNotLeader) {
		return nil, err
	}
	states, err := s.store.ListNodeStates(ctx)
	if err != nil {
		return nil, err
	}
	stateMap := make(map[string]model.NodeState, len(states))
	for _, state := range states {
		stateMap[state.NodeID] = state
	}
	sortClusterMemberViews(members)
	leaderID := s.cluster.LeaderID()
	out := make([]model.ClusterMemberView, 0, len(members))
	for _, member := range members {
		view := model.ClusterMemberView{
			NodeID:               member.NodeID,
			DisplayName:          member.DisplayName,
			EffectiveDisplayName: resolver.DisplayName(member.NodeID),
			APIAddr:              member.APIAddr,
			RaftAddr:             member.RaftAddr,
			PublicIPv4:           member.PublicIPv4,
			Priority:             member.Priority,
			IngressCandidate:     member.IsIngressCandidate(),
			DesiredRole:          member.DesiredRole,
			CurrentRole:          roleMap[member.NodeID],
			Status:               member.Status,
			IsLeader:             member.NodeID == leaderID,
			JoinedAt:             member.JoinedAt,
			UpdatedAt:            member.UpdatedAt,
			RemovedAt:            member.RemovedAt,
		}
		if state, ok := stateMap[member.NodeID]; ok {
			view.HealthStatus = state.Status
			view.HealthReason = state.Reason
			if !state.LastHeartbeatAt.IsZero() {
				lastHeartbeatAt := state.LastHeartbeatAt
				view.LastHeartbeatAt = &lastHeartbeatAt
			}
		}
		if view.CurrentRole == "" && member.IsActive() {
			view.CurrentRole = member.DesiredRole
		}
		out = append(out, view)
	}
	return out, nil
}

func (s *Server) doInternalJSON(ctx context.Context, method, addr, path string, payload any) (int, []byte, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return 0, nil, fmt.Errorf("leader api address unavailable")
	}

	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
		body = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, "http://"+addr+path, body)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token := s.cfg.InternalToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	rawBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, rawBody, nil
}

func writeJSONBytes(w http.ResponseWriter, status int, body []byte) {
	if len(body) == 0 {
		writeJSON(w, status, map[string]any{})
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func (s *Server) renderMembershipError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, cluster.ErrInvalidMemberRole):
		s.renderError(w, http.StatusBadRequest, err)
	case errors.Is(err, cluster.ErrMemberNotFound):
		s.renderError(w, http.StatusNotFound, err)
	case errors.Is(err, cluster.ErrMemberConflict), errors.Is(err, cluster.ErrLastVoterRemoval), errors.Is(err, cluster.ErrNoHealthyTransferCandidate), errors.Is(err, raft.ErrNotLeader):
		s.renderError(w, http.StatusConflict, err)
	default:
		s.renderError(w, http.StatusInternalServerError, err)
	}
}

func sortClusterMemberViews(members []model.ClusterMember) {
	// Active nodes first, then removed history.
	slices.SortFunc(members, func(a, b model.ClusterMember) int {
		if a.Status != b.Status {
			if a.Status == model.ClusterMemberStatusActive {
				return -1
			}
			if b.Status == model.ClusterMemberStatusActive {
				return 1
			}
		}
		if a.Priority != b.Priority {
			return b.Priority - a.Priority
		}
		return strings.Compare(a.NodeID, b.NodeID)
	})
}
