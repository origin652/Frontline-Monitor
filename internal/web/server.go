package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	frontendapp "vps-monitor/frontend"
	"vps-monitor/internal/cluster"
	"vps-monitor/internal/config"
	"vps-monitor/internal/model"
	"vps-monitor/internal/notify"
	"vps-monitor/internal/store"
)

type Server struct {
	cfg           *config.Config
	store         *store.Store
	cluster       *cluster.Manager
	submitter     *cluster.Submitter
	alertResolver *notify.Resolver
	logger        *slog.Logger
}

func New(cfg *config.Config, st *store.Store, cl *cluster.Manager, submitter *cluster.Submitter, alertResolver *notify.Resolver, logger *slog.Logger) (*Server, error) {
	return &Server{
		cfg:           cfg,
		store:         st,
		cluster:       cl,
		submitter:     submitter,
		alertResolver: alertResolver,
		logger:        logger,
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /frontend/", http.StripPrefix("/frontend/", http.FileServer(http.FS(frontendapp.Assets))))
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /", s.handleFrontendApp)
	mux.HandleFunc("GET /admin", s.handleFrontendApp)
	mux.HandleFunc("GET /nodes/{nodeID}", s.handleFrontendApp)
	mux.HandleFunc("GET /events", s.handleFrontendApp)
	mux.HandleFunc("GET /api/v1/cluster", s.handleClusterAPI)
	mux.HandleFunc("GET /api/v1/meta", s.handleMetaAPI)
	mux.HandleFunc("GET /api/v1/nodes", s.handleNodesAPI)
	mux.HandleFunc("GET /api/v1/nodes/{nodeID}", s.handleNodeAPI)
	mux.HandleFunc("GET /api/v1/ingress", s.handleIngressAPI)
	mux.HandleFunc("GET /api/v1/incidents", s.handleIncidentsAPI)
	mux.HandleFunc("GET /api/v1/events", s.handleEventsAPI)
	mux.HandleFunc("GET /api/v1/history", s.handleHistoryAPI)
	mux.HandleFunc("POST /api/v1/test-alert", s.handleTestAlert)
	mux.HandleFunc("GET /api/v1/admin/bootstrap-status", s.handleAdminBootstrapStatus)
	mux.HandleFunc("POST /api/v1/admin/bootstrap", s.handleAdminBootstrap)
	mux.HandleFunc("POST /api/v1/admin/login", s.handleAdminLogin)
	mux.HandleFunc("POST /api/v1/admin/logout", s.handleAdminLogout)
	mux.HandleFunc("GET /api/v1/admin/me", s.handleAdminMe)
	mux.HandleFunc("POST /api/v1/admin/password", s.handleAdminPassword)
	mux.HandleFunc("GET /api/v1/admin/alerts", s.handleAdminAlerts)
	mux.HandleFunc("PUT /api/v1/admin/alerts/{channel}", s.handleAdminAlertByChannel)
	mux.HandleFunc("DELETE /api/v1/admin/alerts/{channel}", s.handleAdminAlertByChannel)
	mux.HandleFunc("GET /api/v1/admin/checks", s.handleAdminChecks)
	mux.HandleFunc("POST /api/v1/admin/checks", s.handleAdminChecks)
	mux.HandleFunc("PUT /api/v1/admin/checks/{id}", s.handleAdminCheckByID)
	mux.HandleFunc("DELETE /api/v1/admin/checks/{id}", s.handleAdminCheckByID)
	mux.HandleFunc("GET /api/v1/admin/members", s.handleAdminMembers)
	mux.HandleFunc("PUT /api/v1/admin/members/{nodeID}/role", s.handleAdminMemberRole)
	mux.HandleFunc("DELETE /api/v1/admin/members/{nodeID}", s.handleAdminMemberByID)
	mux.HandleFunc("GET /api/v1/admin/nodes", s.handleAdminNodes)
	mux.HandleFunc("PUT /api/v1/admin/nodes/{nodeID}", s.handleAdminNodeByID)
	mux.HandleFunc("DELETE /api/v1/admin/nodes/{nodeID}", s.handleAdminNodeByID)
	mux.HandleFunc("POST /internal/v1/observations/heartbeat", s.handleInternalHeartbeat)
	mux.HandleFunc("POST /internal/v1/observations/probe", s.handleInternalProbe)
	mux.HandleFunc("POST /internal/v1/cluster/apply", s.handleInternalApply)
	mux.HandleFunc("POST /internal/v1/cluster/join", s.handleInternalJoin)
	mux.HandleFunc("PUT /internal/v1/cluster/members/{nodeID}/role", s.handleInternalMemberRole)
	mux.HandleFunc("DELETE /internal/v1/cluster/members/{nodeID}", s.handleInternalMemberByID)
	return mux
}

func (s *Server) handleFrontendApp(w http.ResponseWriter, r *http.Request) {
	content, err := frontendapp.Assets.ReadFile("index.html")
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(content)
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	state, _ := s.store.GetNodeState(ctx, s.cfg.Cluster.NodeID)
	status := http.StatusOK
	payload := map[string]any{
		"node_id":   s.cfg.Cluster.NodeID,
		"leader_id": s.cluster.LeaderID(),
		"ok":        true,
	}
	if state != nil {
		payload["status"] = state.Status
		payload["reason"] = state.Reason
		if state.Status == model.StatusCritical {
			status = http.StatusServiceUnavailable
			payload["ok"] = false
		}
	}
	writeJSON(w, status, payload)
}

func (s *Server) handleClusterAPI(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.snapshot(r.Context(), s.isAdminRequest(r))
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) handleMetaAPI(w http.ResponseWriter, r *http.Request) {
	initialized, err := s.adminInitialized(r.Context())
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	resolver, err := s.newNodeNameResolver(r.Context())
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	isAdmin := s.isAdminRequest(r)
	channels := []string{}
	if isAdmin {
		channels, err = s.enabledAlertChannels(r.Context())
		if err != nil {
			s.renderError(w, http.StatusInternalServerError, err)
			return
		}
	}
	leaderID := s.cluster.LeaderID()
	writeJSON(w, http.StatusOK, map[string]any{
		"node_id":             s.cfg.Cluster.NodeID,
		"node_name":           resolver.DisplayName(s.cfg.Cluster.NodeID),
		"leader_id":           leaderID,
		"leader_name":         resolver.DisplayName(leaderID),
		"test_alert_channels": channels,
		"admin_initialized":   initialized,
		"is_admin":            isAdmin,
	})
}

func (s *Server) handleNodesAPI(w http.ResponseWriter, r *http.Request) {
	resolver, err := s.newNodeNameResolver(r.Context())
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	states, err := s.nodeStates(r.Context(), s.isAdminRequest(r), resolver)
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, states)
}

func (s *Server) handleNodeAPI(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("nodeID")
	resolver, err := s.newNodeNameResolver(r.Context())
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	detail, err := s.nodeDetail(r.Context(), nodeID, s.isAdminRequest(r), resolver)
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) handleIngressAPI(w http.ResponseWriter, r *http.Request) {
	ingress, err := s.store.GetIngressState(r.Context())
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	if ingress == nil {
		ingress = &model.IngressState{}
	}
	resolver, err := s.newNodeNameResolver(r.Context())
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	s.decorateIngress(resolver, ingress)
	if !s.isAdminRequest(r) {
		s.redactIngress(ingress)
	}
	writeJSON(w, http.StatusOK, ingress)
}

func (s *Server) handleIncidentsAPI(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	limit := parseLimit(r.URL.Query().Get("limit"), 40)
	incidents, err := s.store.ListIncidents(r.Context(), status, limit)
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	resolver, err := s.newNodeNameResolver(r.Context())
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	s.decorateIncidents(resolver, incidents)
	writeJSON(w, http.StatusOK, incidents)
}

func (s *Server) handleEventsAPI(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 80)
	events, err := s.store.ListEvents(r.Context(), limit)
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	resolver, err := s.newNodeNameResolver(r.Context())
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	s.decorateEvents(resolver, events)
	if !s.isAdminRequest(r) {
		events = s.redactEvents(events)
	}
	writeJSON(w, http.StatusOK, events)
}

func (s *Server) handleHistoryAPI(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("node_id")
	metric := r.URL.Query().Get("metric")
	if nodeID == "" {
		s.renderError(w, http.StatusBadRequest, fmt.Errorf("node_id is required"))
		return
	}
	from := time.Now().UTC().Add(-24 * time.Hour)
	to := time.Now().UTC()
	if raw := r.URL.Query().Get("from"); raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			from = parsed
		}
	}
	if raw := r.URL.Query().Get("to"); raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			to = parsed
		}
	}
	points, err := s.store.History(r.Context(), nodeID, metric, from, to)
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, points)
}

type testAlertRequest struct {
	Channel string `json:"channel"`
	Token   string `json:"token"`
	Note    string `json:"note"`
}

type testAlertResult struct {
	Channel  string `json:"channel"`
	OK       bool   `json:"ok"`
	Response string `json:"response,omitempty"`
	Error    string `json:"error,omitempty"`
}

func (s *Server) handleTestAlert(w http.ResponseWriter, r *http.Request) {
	if !s.isAdminRequest(r) {
		s.renderError(w, http.StatusUnauthorized, fmt.Errorf("admin login required"))
		return
	}
	defer r.Body.Close()

	var req testAlertRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		s.renderError(w, http.StatusBadRequest, err)
		return
	}

	notifiers, err := s.pickNotifiers(r.Context(), req.Channel)
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	if len(notifiers) == 0 {
		message := "no alert channels are enabled"
		if strings.TrimSpace(strings.ToLower(req.Channel)) != "" && strings.TrimSpace(strings.ToLower(req.Channel)) != "all" {
			message = "requested channel is not enabled"
		}
		s.renderError(w, http.StatusBadRequest, fmt.Errorf(message))
		return
	}

	incident := model.Incident{
		ID:       "test-alert:" + time.Now().UTC().Format("20060102150405"),
		NodeID:   s.cfg.Cluster.NodeID,
		RuleKey:  "manual-test",
		Severity: model.StatusDegraded,
		Status:   model.IncidentStatusActive,
		Summary:  "Manual test alert",
		Detail:   truncate(strings.TrimSpace(req.Note), 160),
		OpenedAt: time.Now().UTC(),
	}
	if incident.Detail == "" {
		incident.Detail = "Triggered from the dashboard test-alert panel."
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	results := make([]testAlertResult, 0, len(notifiers))
	for _, notifier := range notifiers {
		response, err := notifier.Send(ctx, "test", incident)
		result := testAlertResult{
			Channel: notifier.Name(),
			OK:      err == nil,
		}
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Response = truncate(response, 240)
		}
		results = append(results, result)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"sent_at": time.Now().UTC(),
		"results": results,
	})
}

func (s *Server) handleInternalHeartbeat(w http.ResponseWriter, r *http.Request) {
	if err := s.requireInternalRequest(r); err != nil {
		s.renderError(w, http.StatusForbidden, err)
		return
	}
	if !s.cluster.IsLeader() {
		s.renderError(w, http.StatusConflict, fmt.Errorf("not leader"))
		return
	}
	defer r.Body.Close()
	var hb model.NodeHeartbeat
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&hb); err != nil {
		s.renderError(w, http.StatusBadRequest, err)
		return
	}
	if _, err := s.cluster.Apply(r.Context(), cluster.CommandHeartbeat, hb); err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true})
}

func (s *Server) handleInternalProbe(w http.ResponseWriter, r *http.Request) {
	if err := s.requireInternalRequest(r); err != nil {
		s.renderError(w, http.StatusForbidden, err)
		return
	}
	if !s.cluster.IsLeader() {
		s.renderError(w, http.StatusConflict, fmt.Errorf("not leader"))
		return
	}
	defer r.Body.Close()
	var probe model.ProbeObservation
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&probe); err != nil {
		s.renderError(w, http.StatusBadRequest, err)
		return
	}
	if _, err := s.cluster.Apply(r.Context(), cluster.CommandProbe, probe); err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true})
}

func (s *Server) handleInternalApply(w http.ResponseWriter, r *http.Request) {
	if err := s.requireInternalRequest(r); err != nil {
		s.renderError(w, http.StatusForbidden, err)
		return
	}
	if !s.cluster.IsLeader() {
		s.renderError(w, http.StatusConflict, fmt.Errorf("not leader"))
		return
	}
	defer r.Body.Close()
	var req struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		s.renderError(w, http.StatusBadRequest, err)
		return
	}
	if !s.isAllowedInternalCommand(req.Type) {
		s.renderError(w, http.StatusBadRequest, fmt.Errorf("unsupported internal command"))
		return
	}
	if _, err := s.cluster.ApplyRaw(r.Context(), req.Type, req.Payload); err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true})
}

func (s *Server) renderError(w http.ResponseWriter, status int, err error) {
	s.logger.Error("http handler error", "status", status, "error", err)
	writeJSON(w, status, map[string]any{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) pickNotifiers(ctx context.Context, channel string) ([]notify.Notifier, error) {
	if s.alertResolver == nil {
		return nil, nil
	}
	return s.alertResolver.Pick(ctx, channel)
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "..."
}

func parseLimit(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
