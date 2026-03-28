package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"vps-monitor/internal/auth"
	"vps-monitor/internal/cluster"
	"vps-monitor/internal/model"
)

const (
	adminSessionCookieName  = "vps_monitor_session"
	adminSessionTTL         = 24 * time.Hour
	maxNodeDisplayNameRunes = 80
)

type adminPasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type adminBootstrapRequest struct {
	Password string `json:"password"`
}

type adminLoginRequest struct {
	Password string `json:"password"`
}

type adminNodeDisplayNameRequest struct {
	DisplayName string `json:"display_name"`
}

type adminNodeDisplayNameView struct {
	NodeID               string     `json:"node_id"`
	DisplayName          string     `json:"display_name,omitempty"`
	ConfigDisplayName    string     `json:"config_display_name,omitempty"`
	EffectiveDisplayName string     `json:"effective_display_name"`
	UpdatedAt            *time.Time `json:"updated_at,omitempty"`
}

func (s *Server) handleAdminBootstrapStatus(w http.ResponseWriter, r *http.Request) {
	initialized, err := s.adminInitialized(r.Context())
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"initialized": initialized})
}

func (s *Server) handleAdminBootstrap(w http.ResponseWriter, r *http.Request) {
	initialized, err := s.adminInitialized(r.Context())
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	if initialized {
		s.renderError(w, http.StatusConflict, fmt.Errorf("administrator already initialized"))
		return
	}
	defer r.Body.Close()
	var req adminBootstrapRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		s.renderError(w, http.StatusBadRequest, err)
		return
	}
	if err := validatePassword(req.Password); err != nil {
		s.renderError(w, http.StatusBadRequest, err)
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	now := time.Now().UTC()
	settings := model.AdminSettings{
		PasswordHash:  hash,
		InitializedAt: now,
		UpdatedAt:     now,
	}
	if err := s.submitter.Apply(r.Context(), cluster.CommandAdminSettings, settings); err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	session, err := s.createAdminSession(r.Context())
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	s.setAdminSessionCookie(w, r, session)
	writeJSON(w, http.StatusCreated, map[string]any{"initialized": true, "is_admin": true})
}

func (s *Server) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	settings, err := s.store.GetAdminSettings(r.Context())
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	if settings == nil || settings.PasswordHash == "" {
		s.renderError(w, http.StatusConflict, fmt.Errorf("administrator is not initialized"))
		return
	}
	defer r.Body.Close()
	var req adminLoginRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		s.renderError(w, http.StatusBadRequest, err)
		return
	}
	if !auth.ComparePasswordHash(settings.PasswordHash, req.Password) {
		s.renderError(w, http.StatusUnauthorized, fmt.Errorf("invalid password"))
		return
	}
	session, err := s.createAdminSession(r.Context())
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	s.setAdminSessionCookie(w, r, session)
	writeJSON(w, http.StatusOK, map[string]any{"is_admin": true})
}

func (s *Server) handleAdminLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(adminSessionCookieName); err == nil && strings.TrimSpace(cookie.Value) != "" {
		_ = s.submitter.Apply(r.Context(), cluster.CommandDeleteSession, map[string]string{"id": strings.TrimSpace(cookie.Value)})
	}
	s.clearAdminSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleAdminMe(w http.ResponseWriter, r *http.Request) {
	initialized, err := s.adminInitialized(r.Context())
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"initialized": initialized,
		"is_admin":    s.isAdminRequest(r),
	})
}

func (s *Server) handleAdminPassword(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdminSession(w, r); !ok {
		return
	}
	settings, err := s.store.GetAdminSettings(r.Context())
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	if settings == nil {
		s.renderError(w, http.StatusConflict, fmt.Errorf("administrator is not initialized"))
		return
	}
	defer r.Body.Close()
	var req adminPasswordRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		s.renderError(w, http.StatusBadRequest, err)
		return
	}
	if !auth.ComparePasswordHash(settings.PasswordHash, req.CurrentPassword) {
		s.renderError(w, http.StatusUnauthorized, fmt.Errorf("current password is invalid"))
		return
	}
	if err := validatePassword(req.NewPassword); err != nil {
		s.renderError(w, http.StatusBadRequest, err)
		return
	}
	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	updated := model.AdminSettings{
		PasswordHash:  hash,
		InitializedAt: settings.InitializedAt,
		UpdatedAt:     time.Now().UTC(),
	}
	if err := s.submitter.Apply(r.Context(), cluster.CommandAdminSettings, updated); err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleAdminChecks(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdminSession(w, r); !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		checks, err := s.store.ListMonitorChecks(r.Context())
		if err != nil {
			s.renderError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, checks)
	case http.MethodPost:
		defer r.Body.Close()
		var check model.MonitorCheck
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&check); err != nil {
			s.renderError(w, http.StatusBadRequest, err)
			return
		}
		now := time.Now().UTC()
		check = check.Normalize()
		if check.ID == "" {
			check.ID = uuid.NewString()
		}
		if check.SortOrder == 0 {
			maxSort, err := s.store.MaxMonitorCheckSortOrder(r.Context())
			if err != nil {
				s.renderError(w, http.StatusInternalServerError, err)
				return
			}
			check.SortOrder = maxSort + 10
		}
		check.CreatedAt = now
		check.UpdatedAt = now
		if err := check.Validate(); err != nil {
			s.renderError(w, http.StatusBadRequest, err)
			return
		}
		if err := s.validateMonitorCheckScope(check); err != nil {
			s.renderError(w, http.StatusBadRequest, err)
			return
		}
		if err := s.submitter.Apply(r.Context(), cluster.CommandMonitorCheck, check); err != nil {
			s.renderError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, check)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleAdminCheckByID(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdminSession(w, r); !ok {
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		s.renderError(w, http.StatusBadRequest, fmt.Errorf("id is required"))
		return
	}
	switch r.Method {
	case http.MethodPut:
		defer r.Body.Close()
		var check model.MonitorCheck
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&check); err != nil {
			s.renderError(w, http.StatusBadRequest, err)
			return
		}
		check = check.Normalize()
		check.ID = id
		if check.CreatedAt.IsZero() {
			check.CreatedAt = time.Now().UTC()
		}
		check.UpdatedAt = time.Now().UTC()
		if err := check.Validate(); err != nil {
			s.renderError(w, http.StatusBadRequest, err)
			return
		}
		if err := s.validateMonitorCheckScope(check); err != nil {
			s.renderError(w, http.StatusBadRequest, err)
			return
		}
		if err := s.submitter.Apply(r.Context(), cluster.CommandMonitorCheck, check); err != nil {
			s.renderError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	case http.MethodDelete:
		if err := s.submitter.Apply(r.Context(), cluster.CommandDeleteCheck, map[string]string{"id": id}); err != nil {
			s.renderError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleAdminNodes(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdminSession(w, r); !ok {
		return
	}
	nodes, err := s.listAdminNodeDisplayNames(r.Context())
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, nodes)
}

func (s *Server) handleAdminNodeByID(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdminSession(w, r); !ok {
		return
	}
	nodeID := strings.TrimSpace(r.PathValue("nodeID"))
	if nodeID == "" {
		s.renderError(w, http.StatusBadRequest, fmt.Errorf("nodeID is required"))
		return
	}
	member, ok, err := s.cluster.MemberByID(r.Context(), nodeID)
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok || !member.IsActive() {
		s.renderError(w, http.StatusNotFound, fmt.Errorf("unknown node %q", nodeID))
		return
	}
	switch r.Method {
	case http.MethodPut:
		defer r.Body.Close()
		var req adminNodeDisplayNameRequest
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
			s.renderError(w, http.StatusBadRequest, err)
			return
		}
		displayName, err := normalizeNodeDisplayName(req.DisplayName)
		if err != nil {
			s.renderError(w, http.StatusBadRequest, err)
			return
		}
		if displayName == "" {
			if err := s.submitter.Apply(r.Context(), cluster.CommandDeleteNodeName, map[string]string{"node_id": nodeID}); err != nil {
				s.renderError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"ok":                     true,
				"effective_display_name": resolverDisplayNameOrNodeID(s, r.Context(), nodeID),
			})
			return
		}
		item := model.NodeDisplayName{
			NodeID:      nodeID,
			DisplayName: displayName,
			UpdatedAt:   time.Now().UTC(),
		}
		if err := s.submitter.Apply(r.Context(), cluster.CommandNodeDisplayName, item); err != nil {
			s.renderError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":                     true,
			"effective_display_name": displayName,
		})
	case http.MethodDelete:
		if err := s.submitter.Apply(r.Context(), cluster.CommandDeleteNodeName, map[string]string{"node_id": nodeID}); err != nil {
			s.renderError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":                     true,
			"effective_display_name": resolverDisplayNameOrNodeID(s, r.Context(), nodeID),
		})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) listAdminNodeDisplayNames(ctx context.Context) ([]adminNodeDisplayNameView, error) {
	resolver, err := s.newNodeNameResolver(ctx)
	if err != nil {
		return nil, err
	}
	members, err := s.cluster.OrderedMembers(ctx)
	if err != nil {
		return nil, err
	}
	nodes := make([]adminNodeDisplayNameView, 0, len(members))
	for _, member := range members {
		item := adminNodeDisplayNameView{
			NodeID:               member.NodeID,
			ConfigDisplayName:    resolver.ConfigDisplayName(member.NodeID),
			EffectiveDisplayName: resolver.DisplayName(member.NodeID),
		}
		if override, ok := resolver.Override(member.NodeID); ok {
			item.DisplayName = override.DisplayName
			updatedAt := override.UpdatedAt
			item.UpdatedAt = &updatedAt
		}
		nodes = append(nodes, item)
	}
	return nodes, nil
}

func (s *Server) adminInitialized(ctx context.Context) (bool, error) {
	settings, err := s.store.GetAdminSettings(ctx)
	if err != nil {
		return false, err
	}
	return settings != nil && strings.TrimSpace(settings.PasswordHash) != "", nil
}

func (s *Server) isAdminRequest(r *http.Request) bool {
	session, err := s.currentAdminSession(r)
	return err == nil && session != nil
}

func (s *Server) requireAdminSession(w http.ResponseWriter, r *http.Request) (*model.AdminSession, bool) {
	session, err := s.currentAdminSession(r)
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return nil, false
	}
	if session == nil {
		s.renderError(w, http.StatusUnauthorized, fmt.Errorf("admin login required"))
		return nil, false
	}
	return session, true
}

func (s *Server) currentAdminSession(r *http.Request) (*model.AdminSession, error) {
	cookie, err := r.Cookie(adminSessionCookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return nil, nil
	}
	session, err := s.store.GetAdminSession(r.Context(), strings.TrimSpace(cookie.Value))
	if err != nil || session == nil {
		return session, err
	}
	if time.Now().UTC().After(session.ExpiresAt) {
		return nil, nil
	}
	return session, nil
}

func (s *Server) createAdminSession(ctx context.Context) (model.AdminSession, error) {
	sessionID, err := auth.GenerateSessionID()
	if err != nil {
		return model.AdminSession{}, err
	}
	session := model.AdminSession{
		ID:        sessionID,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(adminSessionTTL),
	}
	if err := s.submitter.Apply(ctx, cluster.CommandAdminSession, session); err != nil {
		return model.AdminSession{}, err
	}
	return session, nil
}

func (s *Server) setAdminSessionCookie(w http.ResponseWriter, r *http.Request, session model.AdminSession) {
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookieName,
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
		Expires:  session.ExpiresAt,
		MaxAge:   int(adminSessionTTL.Seconds()),
	})
}

func (s *Server) clearAdminSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0).UTC(),
	})
}

func validatePassword(password string) error {
	if len(strings.TrimSpace(password)) < auth.MinPasswordLength {
		return fmt.Errorf("password must be at least %d characters", auth.MinPasswordLength)
	}
	return nil
}

func normalizeNodeDisplayName(displayName string) (string, error) {
	displayName = strings.TrimSpace(displayName)
	if len([]rune(displayName)) > maxNodeDisplayNameRunes {
		return "", fmt.Errorf("display_name must be at most %d characters", maxNodeDisplayNameRunes)
	}
	return displayName, nil
}

func (s *Server) validateMonitorCheckScope(check model.MonitorCheck) error {
	for _, nodeID := range check.NodeIDs {
		member, ok, err := s.cluster.MemberByID(context.Background(), nodeID)
		if err != nil {
			return err
		}
		if !ok || !member.IsActive() {
			return fmt.Errorf("unknown node %q in node_ids", nodeID)
		}
	}
	return nil
}

func resolverDisplayNameOrNodeID(s *Server, ctx context.Context, nodeID string) string {
	resolver, err := s.newNodeNameResolver(ctx)
	if err != nil {
		return nodeID
	}
	return resolver.DisplayName(nodeID)
}
