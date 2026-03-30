package notify

import (
	"context"
	"os"
	"strings"

	"log/slog"

	"vps-monitor/internal/config"
	"vps-monitor/internal/model"
)

const (
	ChannelTelegram = "telegram"
	ChannelSMTP     = "smtp"
	ChannelWebhook  = "webhook"
)

type SettingsReader interface {
	GetAdminSettings(ctx context.Context) (*model.AdminSettings, error)
}

type Resolver struct {
	cfg    *config.Config
	source SettingsReader
	logger *slog.Logger
}

func NewResolver(cfg *config.Config, source SettingsReader, logger *slog.Logger) *Resolver {
	return &Resolver{
		cfg:    cfg,
		source: source,
		logger: logger,
	}
}

func (r *Resolver) EffectiveConfig(ctx context.Context) (*config.Config, error) {
	if r == nil || r.cfg == nil {
		return nil, nil
	}
	effective := cloneConfig(r.cfg)
	if r.source == nil {
		return effective, nil
	}
	settings, err := r.source.GetAdminSettings(ctx)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		return effective, nil
	}
	ApplyAlertSettings(effective, settings.Alerts)
	return effective, nil
}

func (r *Resolver) Build(ctx context.Context) ([]Notifier, error) {
	effective, err := r.EffectiveConfig(ctx)
	if err != nil {
		return nil, err
	}
	if effective == nil {
		return nil, nil
	}
	return Build(effective, r.logger), nil
}

func (r *Resolver) EnabledChannels(ctx context.Context) ([]string, error) {
	notifiers, err := r.Build(ctx)
	if err != nil {
		return nil, err
	}
	channels := make([]string, 0, len(notifiers))
	for _, notifier := range notifiers {
		channels = append(channels, notifier.Name())
	}
	return channels, nil
}

func (r *Resolver) Pick(ctx context.Context, channel string) ([]Notifier, error) {
	notifiers, err := r.Build(ctx)
	if err != nil {
		return nil, err
	}
	channel = strings.TrimSpace(strings.ToLower(channel))
	if channel == "" || channel == "all" {
		return notifiers, nil
	}
	selected := make([]Notifier, 0, 1)
	for _, notifier := range notifiers {
		if notifier.Name() == channel {
			selected = append(selected, notifier)
		}
	}
	return selected, nil
}

func ApplyAndCloneConfig(cfg *config.Config, alerts model.AlertSettings) *config.Config {
	effective := cloneConfig(cfg)
	ApplyAlertSettings(effective, alerts)
	return effective
}

func ApplyAlertSettings(cfg *config.Config, alerts model.AlertSettings) {
	if cfg == nil {
		return
	}
	applyTelegramSettings(&cfg.Alerts.Telegram, alerts.Telegram)
	applySMTPSettings(&cfg.Alerts.SMTP, alerts.SMTP)
	applyWebhookSettings(&cfg.Alerts.WeCom, alerts.Webhook)
}

func TelegramTokenConfigured(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	if strings.TrimSpace(cfg.Alerts.Telegram.BotToken) != "" {
		return true
	}
	if env := strings.TrimSpace(cfg.Alerts.Telegram.BotTokenEnv); env != "" {
		return strings.TrimSpace(os.Getenv(env)) != ""
	}
	return false
}

func SMTPPasswordConfigured(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	if strings.TrimSpace(cfg.Alerts.SMTP.Password) != "" {
		return true
	}
	if env := strings.TrimSpace(cfg.Alerts.SMTP.PasswordEnv); env != "" {
		return strings.TrimSpace(os.Getenv(env)) != ""
	}
	return false
}

func WebhookSecretConfigured(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	if strings.TrimSpace(cfg.Alerts.WeCom.Secret) != "" {
		return true
	}
	if env := strings.TrimSpace(cfg.Alerts.WeCom.SecretEnv); env != "" {
		return strings.TrimSpace(os.Getenv(env)) != ""
	}
	return false
}

func cloneConfig(cfg *config.Config) *config.Config {
	if cfg == nil {
		return nil
	}
	clone := *cfg
	clone.Cluster.Peers = append([]config.ClusterPeer(nil), cfg.Cluster.Peers...)
	clone.Cluster.JoinSeeds = append([]string(nil), cfg.Cluster.JoinSeeds...)
	clone.Checks.Services = append([]string(nil), cfg.Checks.Services...)
	clone.Checks.TCPPorts = append([]int(nil), cfg.Checks.TCPPorts...)
	clone.Checks.HTTPChecks = append([]config.HTTPCheck(nil), cfg.Checks.HTTPChecks...)
	clone.Checks.DockerChecks = append([]string(nil), cfg.Checks.DockerChecks...)
	clone.Alerts.SMTP.To = append([]string(nil), cfg.Alerts.SMTP.To...)
	return &clone
}

func applyTelegramSettings(dst *config.TelegramConfig, src model.AlertChannelSettings) {
	if dst == nil || !src.Managed {
		return
	}
	dst.Enabled = src.Enabled
	dst.BotToken = strings.TrimSpace(src.BotToken)
	dst.ChatID = strings.TrimSpace(src.ChatID)
	dst.ParseMode = strings.TrimSpace(src.ParseMode)
	dst.RequestTout = strings.TrimSpace(src.RequestTimeout)
}

func applySMTPSettings(dst *config.SMTPConfig, src model.AlertChannelSettings) {
	if dst == nil || !src.Managed {
		return
	}
	dst.Enabled = src.Enabled
	dst.Host = strings.TrimSpace(src.SMTPHost)
	dst.Port = src.SMTPPort
	dst.Username = strings.TrimSpace(src.SMTPUsername)
	dst.Password = strings.TrimSpace(src.SMTPPassword)
	dst.From = strings.TrimSpace(src.SMTPFrom)
	dst.To = NormalizeAlertAddresses(src.SMTPTo)
	dst.RequestTout = strings.TrimSpace(src.RequestTimeout)
	dst.UseStartTLS = src.SMTPUseStartTLS
	dst.SubjectPrefix = strings.TrimSpace(src.SubjectPrefix)
}

func applyWebhookSettings(dst *config.WebhookAlertConfig, src model.AlertChannelSettings) {
	if dst == nil || !src.Managed {
		return
	}
	dst.Enabled = src.Enabled
	dst.WebhookURL = strings.TrimSpace(src.WebhookURL)
	dst.Secret = strings.TrimSpace(src.Secret)
	dst.RequestTout = strings.TrimSpace(src.RequestTimeout)
	dst.TitlePrefix = strings.TrimSpace(src.TitlePrefix)
}

func NormalizeAlertAddresses(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
