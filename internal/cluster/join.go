package cluster

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"vps-monitor/internal/model"
)

type joinAttemptError struct {
	statusCode int
	retryable  bool
	message    string
}

func (e *joinAttemptError) Error() string {
	if e == nil {
		return ""
	}
	if e.statusCode > 0 {
		return fmt.Sprintf("join failed with HTTP %d: %s", e.statusCode, e.message)
	}
	return e.message
}

func (m *Manager) AutoJoin(ctx context.Context) error {
	if !m.NeedsJoin() {
		return nil
	}

	seeds := normalizeJoinSeeds(m.cfg.NormalizedJoinSeeds(), m.cfg.APIAddr())
	if len(seeds) == 0 {
		return fmt.Errorf("dynamic join requires at least one remote join seed")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	payload := m.SelfMember()
	round := 0

	for {
		for _, seed := range seeds {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			member, err := m.tryJoinSeed(ctx, client, seed, payload)
			if err == nil {
				m.logger.Info("joined cluster via seed", "seed", seed, "node_id", member.NodeID, "desired_role", member.DesiredRole)
				return nil
			}

			var joinErr *joinAttemptError
			if !errorAsJoinAttempt(err, &joinErr) || !joinErr.retryable {
				return err
			}
			m.logger.Warn("cluster join attempt failed", "seed", seed, "error", joinErr.message, "status", joinErr.statusCode)
		}

		delay := joinRetryDelay(round)
		round++
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (m *Manager) tryJoinSeed(ctx context.Context, client *http.Client, seed string, payload model.ClusterMember) (model.ClusterMember, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return model.ClusterMember{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+seed+"/internal/v1/cluster/join", bytes.NewReader(raw))
	if err != nil {
		return model.ClusterMember{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token := m.cfg.InternalToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return model.ClusterMember{}, &joinAttemptError{
			retryable: true,
			message:   err.Error(),
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return model.ClusterMember{}, &joinAttemptError{
			retryable: true,
			message:   err.Error(),
		}
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var member model.ClusterMember
		if len(body) == 0 {
			return payload, nil
		}
		if err := json.Unmarshal(body, &member); err != nil {
			return model.ClusterMember{}, err
		}
		return member, nil
	}

	message := parseErrorMessage(body)
	retryable := isRetryableJoinStatus(resp.StatusCode, message)
	return model.ClusterMember{}, &joinAttemptError{
		statusCode: resp.StatusCode,
		retryable:  retryable,
		message:    message,
	}
}

func normalizeJoinSeeds(seeds []string, selfAPIAddr string) []string {
	selfAPIAddr = strings.TrimSpace(selfAPIAddr)
	out := make([]string, 0, len(seeds))
	seen := map[string]struct{}{}
	for _, seed := range seeds {
		seed = strings.TrimSpace(seed)
		if seed == "" || seed == selfAPIAddr {
			continue
		}
		if _, ok := seen[seed]; ok {
			continue
		}
		seen[seed] = struct{}{}
		out = append(out, seed)
	}
	return out
}

func parseErrorMessage(body []byte) string {
	var payload struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err == nil && strings.TrimSpace(payload.Error) != "" {
		return strings.TrimSpace(payload.Error)
	}
	return strings.TrimSpace(string(body))
}

func isRetryableJoinStatus(statusCode int, message string) bool {
	if statusCode >= 500 {
		return true
	}
	if statusCode == http.StatusTooManyRequests || statusCode == http.StatusRequestTimeout {
		return true
	}
	if statusCode == http.StatusConflict {
		lower := strings.ToLower(strings.TrimSpace(message))
		return strings.Contains(lower, "leader") && strings.Contains(lower, "retry")
	}
	return statusCode == http.StatusServiceUnavailable
}

func joinRetryDelay(round int) time.Duration {
	backoff := []time.Duration{
		1 * time.Second,
		2 * time.Second,
		3 * time.Second,
		5 * time.Second,
		8 * time.Second,
	}
	if round < len(backoff) {
		return backoff[round]
	}
	return 10 * time.Second
}

func errorAsJoinAttempt(err error, target **joinAttemptError) bool {
	if err == nil || target == nil {
		return false
	}
	typed, ok := err.(*joinAttemptError)
	if !ok {
		return false
	}
	*target = typed
	return true
}
