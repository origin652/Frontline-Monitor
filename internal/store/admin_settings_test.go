package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"vps-monitor/internal/model"
)

func TestAdminSettingsRoundTripAlertSettings(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "monitor.db")
	st, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	now := time.Now().UTC().Round(time.Second)
	settings := model.AdminSettings{
		PasswordHash:  "hash",
		InitializedAt: now,
		UpdatedAt:     now,
		Alerts: model.AlertSettings{
			Telegram: model.AlertChannelSettings{
				Managed:        true,
				Enabled:        true,
				BotToken:       "telegram-secret",
				ChatID:         "12345",
				ParseMode:      "Markdown",
				RequestTimeout: "10s",
				UpdatedAt:      now,
			},
			SMTP: model.AlertChannelSettings{
				Managed:         true,
				Enabled:         true,
				SMTPHost:        "smtp.example.com",
				SMTPPort:        587,
				SMTPUsername:    "ops@example.com",
				SMTPPassword:    "smtp-secret",
				SMTPFrom:        "ops@example.com",
				SMTPTo:          []string{"a@example.com", "b@example.com"},
				SMTPUseStartTLS: true,
				SubjectPrefix:   "[Frontline]",
				RequestTimeout:  "15s",
				UpdatedAt:       now,
			},
		},
	}

	if err := st.UpsertAdminSettings(context.Background(), settings); err != nil {
		t.Fatalf("UpsertAdminSettings() error = %v", err)
	}

	got, err := st.GetAdminSettings(context.Background())
	if err != nil {
		t.Fatalf("GetAdminSettings() error = %v", err)
	}
	if got == nil {
		t.Fatal("GetAdminSettings() = nil, want settings")
	}
	if got.Alerts.Telegram.BotToken != "telegram-secret" {
		t.Fatalf("telegram bot token = %q", got.Alerts.Telegram.BotToken)
	}
	if got.Alerts.SMTP.SMTPPassword != "smtp-secret" {
		t.Fatalf("smtp password = %q", got.Alerts.SMTP.SMTPPassword)
	}
	if len(got.Alerts.SMTP.SMTPTo) != 2 || got.Alerts.SMTP.SMTPTo[0] != "a@example.com" || got.Alerts.SMTP.SMTPTo[1] != "b@example.com" {
		t.Fatalf("smtp recipients = %#v", got.Alerts.SMTP.SMTPTo)
	}
}
