package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"vps-monitor/internal/config"
)

type Client struct {
	cfg    *config.Config
	client *http.Client
}

func New(cfg *config.Config) *Client {
	return &Client{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.CloudflareTimeout(),
		},
	}
}

func (c *Client) Enabled() bool {
	return c.cfg.Cloudflare.Enabled
}

func (c *Client) UpdateARecord(ctx context.Context, ip string) error {
	token := os.Getenv(c.cfg.Cloudflare.APITokenEnv)
	if token == "" {
		return fmt.Errorf("cloudflare token env %q is empty", c.cfg.Cloudflare.APITokenEnv)
	}
	body, err := json.Marshal(map[string]any{
		"type":    "A",
		"name":    c.cfg.Cloudflare.Hostname,
		"content": ip,
		"ttl":     1,
		"proxied": true,
	})
	if err != nil {
		return err
	}
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", c.cfg.Cloudflare.ZoneID, c.cfg.Cloudflare.DNSRecordID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "vps-monitor/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("cloudflare returned %s: %s", resp.Status, string(raw))
	}
	var envelope struct {
		Success bool `json:"success"`
		Errors  []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return err
	}
	if !envelope.Success {
		if len(envelope.Errors) > 0 {
			return fmt.Errorf("cloudflare api error: %s", envelope.Errors[0].Message)
		}
		return fmt.Errorf("cloudflare api update failed")
	}
	return nil
}

func BackoffSchedule() []time.Duration {
	return []time.Duration{2 * time.Second, 5 * time.Second, 10 * time.Second, 30 * time.Second}
}
