package cluster

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"vps-monitor/internal/config"
	"vps-monitor/internal/model"
)

type Submitter struct {
	manager *Manager
	cfg     *config.Config
	client  *http.Client
}

type applyCommandRequest struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func NewSubmitter(manager *Manager, cfg *config.Config) *Submitter {
	return &Submitter{
		manager: manager,
		cfg:     cfg,
		client: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

func (s *Submitter) SubmitHeartbeat(ctx context.Context, hb model.NodeHeartbeat) error {
	if s.manager.IsLeader() {
		_, err := s.manager.Apply(ctx, CommandHeartbeat, hb)
		return err
	}
	return s.post(ctx, "/internal/v1/observations/heartbeat", hb)
}

func (s *Submitter) SubmitProbe(ctx context.Context, probe model.ProbeObservation) error {
	if s.manager.IsLeader() {
		_, err := s.manager.Apply(ctx, CommandProbe, probe)
		return err
	}
	return s.post(ctx, "/internal/v1/observations/probe", probe)
}

func (s *Submitter) post(ctx context.Context, path string, payload any) error {
	leaderAddr := s.manager.LeaderAPIAddr()
	if leaderAddr == "" {
		return fmt.Errorf("leader api address unavailable")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+leaderAddr+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token := s.cfg.InternalToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("leader submit failed with %s: %s", resp.Status, string(body))
	}
	return nil
}

func (s *Submitter) Apply(ctx context.Context, cmdType string, payload any) error {
	if s.manager.IsLeader() {
		_, err := s.manager.Apply(ctx, cmdType, payload)
		return err
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return s.post(ctx, "/internal/v1/cluster/apply", applyCommandRequest{
		Type:    cmdType,
		Payload: payloadBytes,
	})
}
