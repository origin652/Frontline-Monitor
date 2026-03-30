package model

import "time"

type AdminSettings struct {
	PasswordHash  string        `json:"password_hash"`
	InitializedAt time.Time     `json:"initialized_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
	Alerts        AlertSettings `json:"alerts,omitempty"`
}

type AlertSettings struct {
	Telegram AlertChannelSettings `json:"telegram,omitempty"`
	SMTP     AlertChannelSettings `json:"smtp,omitempty"`
	Webhook  AlertChannelSettings `json:"webhook,omitempty"`
}

type AlertChannelSettings struct {
	Managed         bool      `json:"managed,omitempty"`
	Enabled         bool      `json:"enabled"`
	BotToken        string    `json:"bot_token,omitempty"`
	ChatID          string    `json:"chat_id,omitempty"`
	ParseMode       string    `json:"parse_mode,omitempty"`
	SMTPHost        string    `json:"smtp_host,omitempty"`
	SMTPPort        int       `json:"smtp_port,omitempty"`
	SMTPUsername    string    `json:"smtp_username,omitempty"`
	SMTPPassword    string    `json:"smtp_password,omitempty"`
	SMTPFrom        string    `json:"smtp_from,omitempty"`
	SMTPTo          []string  `json:"smtp_to,omitempty"`
	SMTPUseStartTLS bool      `json:"smtp_use_starttls,omitempty"`
	SubjectPrefix   string    `json:"subject_prefix,omitempty"`
	WebhookURL      string    `json:"webhook_url,omitempty"`
	Secret          string    `json:"secret,omitempty"`
	TitlePrefix     string    `json:"title_prefix,omitempty"`
	RequestTimeout  string    `json:"request_timeout,omitempty"`
	UpdatedAt       time.Time `json:"updated_at,omitempty"`
}

type AdminSession struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type NodeDisplayName struct {
	NodeID      string    `json:"node_id"`
	DisplayName string    `json:"display_name"`
	UpdatedAt   time.Time `json:"updated_at"`
}
