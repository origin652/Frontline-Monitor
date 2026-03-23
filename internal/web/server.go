package web

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"math"
	"net/http"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"time"

	"vps-monitor/internal/cluster"
	"vps-monitor/internal/config"
	"vps-monitor/internal/model"
	"vps-monitor/internal/notify"
	"vps-monitor/internal/store"
)

//go:embed templates/*.gohtml static/*
var assets embed.FS

type Server struct {
	cfg       *config.Config
	store     *store.Store
	cluster   *cluster.Manager
	notifiers []notify.Notifier
	logger    *slog.Logger
	templates *template.Template
}

func New(cfg *config.Config, st *store.Store, cl *cluster.Manager, notifiers []notify.Notifier, logger *slog.Logger) (*Server, error) {
	tpl, err := template.New("base").Funcs(template.FuncMap{
		"formatTime":  formatTime,
		"fromNow":     fromNow,
		"statusClass": statusClass,
		"percent":     percent,
		"sparkline":   sparkline,
		"truncate":    truncate,
	}).ParseFS(assets, "templates/*.gohtml")
	if err != nil {
		return nil, err
	}
	return &Server{
		cfg:       cfg,
		store:     st,
		cluster:   cl,
		notifiers: notifiers,
		logger:    logger,
		templates: tpl,
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	staticFS, _ := fs.Sub(assets, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /", s.handleOverviewPage)
	mux.HandleFunc("GET /nodes/{nodeID}", s.handleNodePage)
	mux.HandleFunc("GET /events", s.handleEventsPage)
	mux.HandleFunc("GET /api/v1/cluster", s.handleClusterAPI)
	mux.HandleFunc("GET /api/v1/nodes", s.handleNodesAPI)
	mux.HandleFunc("GET /api/v1/nodes/{nodeID}", s.handleNodeAPI)
	mux.HandleFunc("GET /api/v1/ingress", s.handleIngressAPI)
	mux.HandleFunc("GET /api/v1/incidents", s.handleIncidentsAPI)
	mux.HandleFunc("GET /api/v1/events", s.handleEventsAPI)
	mux.HandleFunc("GET /api/v1/history", s.handleHistoryAPI)
	mux.HandleFunc("POST /api/v1/test-alert", s.handleTestAlert)
	mux.HandleFunc("POST /internal/v1/observations/heartbeat", s.handleInternalHeartbeat)
	mux.HandleFunc("POST /internal/v1/observations/probe", s.handleInternalProbe)
	return mux
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

func (s *Server) handleOverviewPage(w http.ResponseWriter, r *http.Request) {
	view, err := s.buildOverviewView(r.Context())
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	if err := s.templates.ExecuteTemplate(w, "overview", view); err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
	}
}

func (s *Server) handleNodePage(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("nodeID")
	view, err := s.buildNodeView(r.Context(), nodeID)
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	if err := s.templates.ExecuteTemplate(w, "node", view); err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
	}
}

func (s *Server) handleEventsPage(w http.ResponseWriter, r *http.Request) {
	view, err := s.buildEventsView(r.Context())
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	if err := s.templates.ExecuteTemplate(w, "events", view); err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
	}
}

func (s *Server) handleClusterAPI(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.snapshot(r.Context())
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) handleNodesAPI(w http.ResponseWriter, r *http.Request) {
	states, err := s.nodeStates(r.Context())
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, states)
}

func (s *Server) handleNodeAPI(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("nodeID")
	detail, err := s.nodeDetail(r.Context(), nodeID)
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
	writeJSON(w, http.StatusOK, incidents)
}

func (s *Server) handleEventsAPI(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 80)
	events, err := s.store.ListEvents(r.Context(), limit)
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
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
	if len(s.notifiers) == 0 {
		s.renderError(w, http.StatusBadRequest, fmt.Errorf("no alert channels are enabled"))
		return
	}
	defer r.Body.Close()

	var req testAlertRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		s.renderError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.authorizeTestAlert(r, req.Token); err != nil {
		s.renderError(w, http.StatusUnauthorized, err)
		return
	}

	notifiers := s.pickNotifiers(req.Channel)
	if len(notifiers) == 0 {
		s.renderError(w, http.StatusBadRequest, fmt.Errorf("requested channel is not enabled"))
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

func (s *Server) renderError(w http.ResponseWriter, status int, err error) {
	s.logger.Error("http handler error", "status", status, "error", err)
	writeJSON(w, status, map[string]any{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) authorizeTestAlert(r *http.Request, token string) error {
	envToken := strings.TrimSpace(os.Getenv("MONITOR_TEST_ALERT_TOKEN"))
	if envToken != "" {
		if subtleCompare(token, envToken) {
			return nil
		}
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") && subtleCompare(strings.TrimSpace(authHeader[7:]), envToken) {
			return nil
		}
		return fmt.Errorf("invalid test alert token")
	}

	host := r.RemoteAddr
	if parsed, err := netip.ParseAddrPort(host); err == nil {
		if parsed.Addr().IsLoopback() {
			return nil
		}
	}
	if strings.HasPrefix(host, "127.0.0.1:") || strings.HasPrefix(host, "[::1]:") {
		return nil
	}
	return fmt.Errorf("test alerts require loopback access or MONITOR_TEST_ALERT_TOKEN")
}

func (s *Server) pickNotifiers(channel string) []notify.Notifier {
	channel = strings.TrimSpace(strings.ToLower(channel))
	if channel == "" || channel == "all" {
		return s.notifiers
	}
	var selected []notify.Notifier
	for _, notifier := range s.notifiers {
		if notifier.Name() == channel {
			selected = append(selected, notifier)
		}
	}
	return selected
}

func subtleCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := 0; i < len(a); i++ {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	return t.Local().Format("2006-01-02 15:04:05")
}

func fromNow(t time.Time) string {
	if t.IsZero() {
		return "no signal"
	}
	diff := time.Since(t)
	if diff < 0 {
		diff = -diff
	}
	switch {
	case diff < time.Minute:
		return fmt.Sprintf("%ds ago", int(diff.Seconds()))
	case diff < time.Hour:
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	default:
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	}
}

func statusClass(status string) string {
	switch status {
	case model.StatusHealthy:
		return "tone-ok"
	case model.StatusDegraded:
		return "tone-warn"
	case model.StatusCritical:
		return "tone-bad"
	default:
		return "tone-unknown"
	}
}

func percent(value float64) string {
	return fmt.Sprintf("%.0f%%", value)
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "..."
}

func sparkline(points []model.MetricPoint) template.HTML {
	if len(points) == 0 {
		return template.HTML(`<svg viewBox="0 0 100 24" preserveAspectRatio="none"><path d="M0 20 L100 20" /></svg>`)
	}
	maxValue := 0.0
	for _, point := range points {
		maxValue = math.Max(maxValue, point.Value)
	}
	if maxValue == 0 {
		maxValue = 1
	}
	var path strings.Builder
	for i, point := range points {
		x := float64(i) / float64(max(1, len(points)-1)) * 100
		y := 22 - (point.Value/maxValue)*20
		if i == 0 {
			path.WriteString(fmt.Sprintf("M%.2f %.2f", x, y))
		} else {
			path.WriteString(fmt.Sprintf(" L%.2f %.2f", x, y))
		}
	}
	return template.HTML(fmt.Sprintf(`<svg viewBox="0 0 100 24" preserveAspectRatio="none"><path d="%s" /></svg>`, path.String()))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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
