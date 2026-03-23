package notify

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"

	"vps-monitor/internal/config"
	"vps-monitor/internal/model"
)

type Notifier interface {
	Name() string
	Send(ctx context.Context, action string, incident model.Incident) (string, error)
}

func Build(cfg *config.Config, logger *slog.Logger) []Notifier {
	var notifiers []Notifier
	if cfg.Alerts.Telegram.Enabled {
		notifiers = append(notifiers, NewTelegramNotifier(cfg, logger))
	}
	if cfg.Alerts.SMTP.Enabled {
		notifiers = append(notifiers, NewSMTPNotifier(cfg, logger))
	}
	if cfg.Alerts.WeCom.Enabled {
		notifiers = append(notifiers, NewWebhookNotifier(cfg, logger))
	}
	return notifiers
}

type TelegramNotifier struct {
	cfg    *config.Config
	logger *slog.Logger
	client *http.Client
}

func NewTelegramNotifier(cfg *config.Config, logger *slog.Logger) *TelegramNotifier {
	return &TelegramNotifier{
		cfg:    cfg,
		logger: logger,
		client: &http.Client{Timeout: cfg.TelegramTimeout()},
	}
}

func (n *TelegramNotifier) Name() string { return "telegram" }

func (n *TelegramNotifier) Send(ctx context.Context, action string, incident model.Incident) (string, error) {
	token := os.Getenv(n.cfg.Alerts.Telegram.BotTokenEnv)
	if token == "" {
		return "", fmt.Errorf("telegram bot token env %q is empty", n.cfg.Alerts.Telegram.BotTokenEnv)
	}
	body, err := json.Marshal(map[string]any{
		"chat_id":    n.cfg.Alerts.Telegram.ChatID,
		"text":       renderAlertText(action, incident),
		"parse_mode": defaultString(n.cfg.Alerts.Telegram.ParseMode, "Markdown"),
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.telegram.org/bot"+token+"/sendMessage", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := n.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode >= 300 {
		return string(respBody), fmt.Errorf("telegram returned %s", resp.Status)
	}
	return string(respBody), nil
}

type SMTPNotifier struct {
	cfg    *config.Config
	logger *slog.Logger
}

func NewSMTPNotifier(cfg *config.Config, logger *slog.Logger) *SMTPNotifier {
	return &SMTPNotifier{cfg: cfg, logger: logger}
}

func (n *SMTPNotifier) Name() string { return "smtp" }

func (n *SMTPNotifier) Send(ctx context.Context, action string, incident model.Incident) (string, error) {
	password := os.Getenv(n.cfg.Alerts.SMTP.PasswordEnv)
	if password == "" {
		return "", fmt.Errorf("smtp password env %q is empty", n.cfg.Alerts.SMTP.PasswordEnv)
	}
	address := fmt.Sprintf("%s:%d", n.cfg.Alerts.SMTP.Host, n.cfg.Alerts.SMTP.Port)
	subject := fmt.Sprintf("%s[%s] %s", defaultString(n.cfg.Alerts.SMTP.SubjectPrefix, "Monitor "), strings.ToUpper(action), incident.Summary)
	body := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s\r\n", n.cfg.Alerts.SMTP.From, strings.Join(n.cfg.Alerts.SMTP.To, ","), subject, renderAlertText(action, incident))
	auth := smtp.PlainAuth("", n.cfg.Alerts.SMTP.Username, password, n.cfg.Alerts.SMTP.Host)

	var err error
	done := make(chan struct{})
	go func() {
		defer close(done)
		err = sendMail(address, auth, n.cfg.Alerts.SMTP.From, n.cfg.Alerts.SMTP.To, []byte(body), n.cfg.Alerts.SMTP.UseStartTLS)
	}()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-done:
		if err != nil {
			return "", err
		}
		return "sent", nil
	}
}

type WebhookNotifier struct {
	cfg    *config.Config
	logger *slog.Logger
	client *http.Client
}

func NewWebhookNotifier(cfg *config.Config, logger *slog.Logger) *WebhookNotifier {
	return &WebhookNotifier{
		cfg:    cfg,
		logger: logger,
		client: &http.Client{Timeout: cfg.WebhookTimeout()},
	}
}

func (n *WebhookNotifier) Name() string { return "webhook" }

func (n *WebhookNotifier) Send(ctx context.Context, action string, incident model.Incident) (string, error) {
	payload := map[string]any{
		"msg_type": "text",
		"content": map[string]string{
			"text": fmt.Sprintf("%s%s", defaultString(n.cfg.Alerts.WeCom.TitlePrefix, ""), renderAlertText(action, incident)),
		},
	}
	url := n.cfg.Alerts.WeCom.WebhookURL
	if secret := os.Getenv(n.cfg.Alerts.WeCom.SecretEnv); secret != "" {
		ts := time.Now().UnixMilli()
		signature := signWebhook(secret, ts)
		if strings.Contains(url, "?") {
			url = fmt.Sprintf("%s&timestamp=%d&sign=%s", url, ts, signature)
		} else {
			url = fmt.Sprintf("%s?timestamp=%d&sign=%s", url, ts, signature)
		}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := n.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode >= 300 {
		return string(respBody), fmt.Errorf("webhook returned %s", resp.Status)
	}
	return string(respBody), nil
}

func renderAlertText(action string, incident model.Incident) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("*%s* `%s`\n", strings.ToUpper(action), incident.Severity))
	builder.WriteString(fmt.Sprintf("Node: `%s`\n", incident.NodeID))
	builder.WriteString(fmt.Sprintf("Rule: `%s`\n", incident.RuleKey))
	builder.WriteString(fmt.Sprintf("Summary: %s\n", incident.Summary))
	if incident.Detail != "" {
		builder.WriteString(fmt.Sprintf("Detail: %s\n", incident.Detail))
	}
	builder.WriteString(fmt.Sprintf("Opened: %s UTC", incident.OpenedAt.UTC().Format(time.RFC3339)))
	if incident.ResolvedAt != nil {
		builder.WriteString(fmt.Sprintf("\nResolved: %s UTC", incident.ResolvedAt.UTC().Format(time.RFC3339)))
	}
	return builder.String()
}

func signWebhook(secret string, timestamp int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(fmt.Sprintf("%d\n%s", timestamp, secret)))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func sendMail(addr string, auth smtp.Auth, from string, to []string, msg []byte, useStartTLS bool) error {
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, strings.Split(addr, ":")[0])
	if err != nil {
		return err
	}
	defer client.Quit()

	if useStartTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: strings.Split(addr, ":")[0]}); err != nil {
				return err
			}
		}
	}
	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(auth); err != nil {
				return err
			}
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(msg); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}
