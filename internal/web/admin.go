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
	"vps-monitor/internal/config"
	"vps-monitor/internal/model"
	"vps-monitor/internal/notify"
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

type adminAlertChannelRequest struct {
	Enabled         bool     `json:"enabled"`
	BotToken        string   `json:"bot_token,omitempty"`
	ChatID          string   `json:"chat_id,omitempty"`
	ParseMode       string   `json:"parse_mode,omitempty"`
	SMTPHost        string   `json:"smtp_host,omitempty"`
	SMTPPort        int      `json:"smtp_port,omitempty"`
	SMTPUsername    string   `json:"smtp_username,omitempty"`
	SMTPPassword    string   `json:"smtp_password,omitempty"`
	SMTPFrom        string   `json:"smtp_from,omitempty"`
	SMTPTo          []string `json:"smtp_to,omitempty"`
	SMTPUseStartTLS bool     `json:"smtp_use_starttls,omitempty"`
	SubjectPrefix   string   `json:"subject_prefix,omitempty"`
	WebhookURL      string   `json:"webhook_url,omitempty"`
	Secret          string   `json:"secret,omitempty"`
	TitlePrefix     string   `json:"title_prefix,omitempty"`
	RequestTimeout  string   `json:"request_timeout,omitempty"`
}

type adminAlertSettingsView struct {
	Telegram adminAlertChannelView `json:"telegram"`
	SMTP     adminAlertChannelView `json:"smtp"`
	Webhook  adminAlertChannelView `json:"webhook"`
}

type adminAlertChannelView struct {
	Channel          string     `json:"channel"`
	Source           string     `json:"source"`
	Managed          bool       `json:"managed"`
	Enabled          bool       `json:"enabled"`
	SecretConfigured bool       `json:"secret_configured"`
	UpdatedAt        *time.Time `json:"updated_at,omitempty"`
	ChatID           string     `json:"chat_id,omitempty"`
	ParseMode        string     `json:"parse_mode,omitempty"`
	RequestTimeout   string     `json:"request_timeout,omitempty"`
	SMTPHost         string     `json:"smtp_host,omitempty"`
	SMTPPort         int        `json:"smtp_port,omitempty"`
	SMTPUsername     string     `json:"smtp_username,omitempty"`
	SMTPFrom         string     `json:"smtp_from,omitempty"`
	SMTPTo           []string   `json:"smtp_to,omitempty"`
	SMTPUseStartTLS  bool       `json:"smtp_use_starttls,omitempty"`
	SubjectPrefix    string     `json:"subject_prefix,omitempty"`
	WebhookURL       string     `json:"webhook_url,omitempty"`
	TitlePrefix      string     `json:"title_prefix,omitempty"`
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
		Alerts:        settings.Alerts,
	}
	if err := s.submitter.Apply(r.Context(), cluster.CommandAdminSettings, updated); err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleAdminAlerts(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdminSession(w, r); !ok {
		return
	}
	view, err := s.listAdminAlerts(r.Context())
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) handleAdminAlertByChannel(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdminSession(w, r); !ok {
		return
	}
	channel := strings.TrimSpace(strings.ToLower(r.PathValue("channel")))
	if !isAlertChannel(channel) {
		s.renderError(w, http.StatusBadRequest, fmt.Errorf("unsupported alert channel"))
		return
	}
	settings, err := s.store.GetAdminSettings(r.Context())
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err)
		return
	}
	if settings == nil || settings.PasswordHash == "" {
		s.renderError(w, http.StatusConflict, fmt.Errorf("administrator is not initialized"))
		return
	}

	switch r.Method {
	case http.MethodPut:
		defer r.Body.Close()
		var req adminAlertChannelRequest
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
			s.renderError(w, http.StatusBadRequest, err)
			return
		}
		updated := *settings
		if err := applyAlertChannelRequest(&updated.Alerts, channel, req); err != nil {
			s.renderError(w, http.StatusBadRequest, err)
			return
		}
		effective := notify.ApplyAndCloneConfig(s.cfg, updated.Alerts)
		if err := validateEffectiveAlertChannel(channel, effective); err != nil {
			s.renderError(w, http.StatusBadRequest, err)
			return
		}
		updated.UpdatedAt = time.Now().UTC()
		if err := s.submitter.Apply(r.Context(), cluster.CommandAdminSettings, updated); err != nil {
			s.renderError(w, http.StatusInternalServerError, err)
			return
		}
		view, err := s.adminAlertChannelView(effective, updated.Alerts, channel)
		if err != nil {
			s.renderError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, view)
	case http.MethodDelete:
		resetAlertChannel(&settings.Alerts, channel)
		settings.UpdatedAt = time.Now().UTC()
		if err := s.submitter.Apply(r.Context(), cluster.CommandAdminSettings, *settings); err != nil {
			s.renderError(w, http.StatusInternalServerError, err)
			return
		}
		view, err := s.listAdminAlerts(r.Context())
		if err != nil {
			s.renderError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, view)
	default:
		http.NotFound(w, r)
	}
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

func (s *Server) listAdminAlerts(ctx context.Context) (adminAlertSettingsView, error) {
	settings, err := s.store.GetAdminSettings(ctx)
	if err != nil {
		return adminAlertSettingsView{}, err
	}
	if settings == nil {
		settings = &model.AdminSettings{}
	}
	effective := s.cfg
	if s.alertResolver != nil {
		effective, err = s.alertResolver.EffectiveConfig(ctx)
		if err != nil {
			return adminAlertSettingsView{}, err
		}
	}
	view := adminAlertSettingsView{}
	if view.Telegram, err = s.adminAlertChannelView(effective, settings.Alerts, notify.ChannelTelegram); err != nil {
		return adminAlertSettingsView{}, err
	}
	if view.SMTP, err = s.adminAlertChannelView(effective, settings.Alerts, notify.ChannelSMTP); err != nil {
		return adminAlertSettingsView{}, err
	}
	if view.Webhook, err = s.adminAlertChannelView(effective, settings.Alerts, notify.ChannelWebhook); err != nil {
		return adminAlertSettingsView{}, err
	}
	return view, nil
}

func (s *Server) adminAlertChannelView(effective *config.Config, alerts model.AlertSettings, channel string) (adminAlertChannelView, error) {
	switch channel {
	case notify.ChannelTelegram:
		runtime := alerts.Telegram
		view := adminAlertChannelView{
			Channel:          channel,
			Source:           alertSource(runtime.Managed),
			Managed:          runtime.Managed,
			Enabled:          effective != nil && effective.Alerts.Telegram.Enabled,
			SecretConfigured: notify.TelegramTokenConfigured(effective),
			ChatID:           strings.TrimSpace(effective.Alerts.Telegram.ChatID),
			ParseMode:        strings.TrimSpace(effective.Alerts.Telegram.ParseMode),
			RequestTimeout:   strings.TrimSpace(effective.Alerts.Telegram.RequestTout),
		}
		if runtime.Managed && !runtime.UpdatedAt.IsZero() {
			updatedAt := runtime.UpdatedAt
			view.UpdatedAt = &updatedAt
		}
		return view, nil
	case notify.ChannelSMTP:
		runtime := alerts.SMTP
		view := adminAlertChannelView{
			Channel:          channel,
			Source:           alertSource(runtime.Managed),
			Managed:          runtime.Managed,
			Enabled:          effective != nil && effective.Alerts.SMTP.Enabled,
			SecretConfigured: notify.SMTPPasswordConfigured(effective),
			RequestTimeout:   strings.TrimSpace(effective.Alerts.SMTP.RequestTout),
			SMTPHost:         strings.TrimSpace(effective.Alerts.SMTP.Host),
			SMTPPort:         effective.Alerts.SMTP.Port,
			SMTPUsername:     strings.TrimSpace(effective.Alerts.SMTP.Username),
			SMTPFrom:         strings.TrimSpace(effective.Alerts.SMTP.From),
			SMTPTo:           append([]string(nil), effective.Alerts.SMTP.To...),
			SMTPUseStartTLS:  effective.Alerts.SMTP.UseStartTLS,
			SubjectPrefix:    strings.TrimSpace(effective.Alerts.SMTP.SubjectPrefix),
		}
		if runtime.Managed && !runtime.UpdatedAt.IsZero() {
			updatedAt := runtime.UpdatedAt
			view.UpdatedAt = &updatedAt
		}
		return view, nil
	case notify.ChannelWebhook:
		runtime := alerts.Webhook
		view := adminAlertChannelView{
			Channel:          channel,
			Source:           alertSource(runtime.Managed),
			Managed:          runtime.Managed,
			Enabled:          effective != nil && effective.Alerts.WeCom.Enabled,
			SecretConfigured: notify.WebhookSecretConfigured(effective),
			RequestTimeout:   strings.TrimSpace(effective.Alerts.WeCom.RequestTout),
			WebhookURL:       strings.TrimSpace(effective.Alerts.WeCom.WebhookURL),
			TitlePrefix:      strings.TrimSpace(effective.Alerts.WeCom.TitlePrefix),
		}
		if runtime.Managed && !runtime.UpdatedAt.IsZero() {
			updatedAt := runtime.UpdatedAt
			view.UpdatedAt = &updatedAt
		}
		return view, nil
	default:
		return adminAlertChannelView{}, fmt.Errorf("unsupported alert channel")
	}
}

func applyAlertChannelRequest(alerts *model.AlertSettings, channel string, req adminAlertChannelRequest) error {
	if alerts == nil {
		return fmt.Errorf("alert settings are required")
	}
	now := time.Now().UTC()
	switch channel {
	case notify.ChannelTelegram:
		next := alerts.Telegram
		next.Managed = true
		next.Enabled = req.Enabled
		if token := strings.TrimSpace(req.BotToken); token != "" {
			next.BotToken = token
		}
		next.ChatID = strings.TrimSpace(req.ChatID)
		next.ParseMode = strings.TrimSpace(req.ParseMode)
		next.RequestTimeout = strings.TrimSpace(req.RequestTimeout)
		next.UpdatedAt = now
		alerts.Telegram = next
		return nil
	case notify.ChannelSMTP:
		next := alerts.SMTP
		next.Managed = true
		next.Enabled = req.Enabled
		next.SMTPHost = strings.TrimSpace(req.SMTPHost)
		next.SMTPPort = req.SMTPPort
		next.SMTPUsername = strings.TrimSpace(req.SMTPUsername)
		if password := strings.TrimSpace(req.SMTPPassword); password != "" {
			next.SMTPPassword = password
		}
		next.SMTPFrom = strings.TrimSpace(req.SMTPFrom)
		next.SMTPTo = notify.NormalizeAlertAddresses(req.SMTPTo)
		next.SMTPUseStartTLS = req.SMTPUseStartTLS
		next.SubjectPrefix = strings.TrimSpace(req.SubjectPrefix)
		next.RequestTimeout = strings.TrimSpace(req.RequestTimeout)
		next.UpdatedAt = now
		alerts.SMTP = next
		return nil
	case notify.ChannelWebhook:
		next := alerts.Webhook
		next.Managed = true
		next.Enabled = req.Enabled
		next.WebhookURL = strings.TrimSpace(req.WebhookURL)
		if secret := strings.TrimSpace(req.Secret); secret != "" {
			next.Secret = secret
		}
		next.TitlePrefix = strings.TrimSpace(req.TitlePrefix)
		next.RequestTimeout = strings.TrimSpace(req.RequestTimeout)
		next.UpdatedAt = now
		alerts.Webhook = next
		return nil
	default:
		return fmt.Errorf("unsupported alert channel")
	}
}

func resetAlertChannel(alerts *model.AlertSettings, channel string) {
	if alerts == nil {
		return
	}
	switch channel {
	case notify.ChannelTelegram:
		alerts.Telegram = model.AlertChannelSettings{}
	case notify.ChannelSMTP:
		alerts.SMTP = model.AlertChannelSettings{}
	case notify.ChannelWebhook:
		alerts.Webhook = model.AlertChannelSettings{}
	}
}

func validateEffectiveAlertChannel(channel string, cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("effective config is not available")
	}
	switch channel {
	case notify.ChannelTelegram:
		if raw := strings.TrimSpace(cfg.Alerts.Telegram.RequestTout); raw != "" {
			if _, err := time.ParseDuration(raw); err != nil {
				return fmt.Errorf("telegram request_timeout must be a valid duration")
			}
		}
		if !cfg.Alerts.Telegram.Enabled {
			return nil
		}
		if strings.TrimSpace(cfg.Alerts.Telegram.ChatID) == "" {
			return fmt.Errorf("telegram chat_id is required when enabled")
		}
		if !notify.TelegramTokenConfigured(cfg) {
			return fmt.Errorf("telegram bot token is required when enabled")
		}
		return nil
	case notify.ChannelSMTP:
		if raw := strings.TrimSpace(cfg.Alerts.SMTP.RequestTout); raw != "" {
			if _, err := time.ParseDuration(raw); err != nil {
				return fmt.Errorf("smtp request_timeout must be a valid duration")
			}
		}
		if !cfg.Alerts.SMTP.Enabled {
			return nil
		}
		if strings.TrimSpace(cfg.Alerts.SMTP.Host) == "" {
			return fmt.Errorf("smtp host is required when enabled")
		}
		if cfg.Alerts.SMTP.Port <= 0 {
			return fmt.Errorf("smtp port must be a positive integer when enabled")
		}
		if strings.TrimSpace(cfg.Alerts.SMTP.From) == "" {
			return fmt.Errorf("smtp from is required when enabled")
		}
		if len(notify.NormalizeAlertAddresses(cfg.Alerts.SMTP.To)) == 0 {
			return fmt.Errorf("smtp to must contain at least one recipient when enabled")
		}
		return nil
	case notify.ChannelWebhook:
		if raw := strings.TrimSpace(cfg.Alerts.WeCom.RequestTout); raw != "" {
			if _, err := time.ParseDuration(raw); err != nil {
				return fmt.Errorf("webhook request_timeout must be a valid duration")
			}
		}
		if !cfg.Alerts.WeCom.Enabled {
			return nil
		}
		if strings.TrimSpace(cfg.Alerts.WeCom.WebhookURL) == "" {
			return fmt.Errorf("webhook_url is required when enabled")
		}
		return nil
	default:
		return fmt.Errorf("unsupported alert channel")
	}
}

func isAlertChannel(channel string) bool {
	switch channel {
	case notify.ChannelTelegram, notify.ChannelSMTP, notify.ChannelWebhook:
		return true
	default:
		return false
	}
}

func alertSource(managed bool) string {
	if managed {
		return "runtime"
	}
	return "config"
}
