package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"vps-monitor/internal/cluster"
	"vps-monitor/internal/config"
	"vps-monitor/internal/model"
	"vps-monitor/internal/notify"
	"vps-monitor/internal/store"
)

func TestAdminBootstrapChecksAndRedaction(t *testing.T) {
	t.Parallel()

	raftAddr := freeTCPAddr(t)
	cfg := &config.Config{
		Cluster: config.ClusterConfig{
			NodeID:   "node-a",
			RaftAddr: raftAddr,
			Peers: []config.ClusterPeer{
				{
					NodeID:      "node-a",
					DisplayName: "Shanghai-A",
					APIAddr:     "127.0.0.1:8443",
					RaftAddr:    raftAddr,
					PublicIPv4:  "203.0.113.10",
					Priority:    100,
				},
			},
			Priority: 100,
		},
		Network: config.NetworkConfig{
			ListenAddr:      "127.0.0.1:8443",
			PublicIPv4:      "203.0.113.10",
			PublicHTTPSPort: 443,
		},
		Storage: config.StorageConfig{
			DataDir:       t.TempDir(),
			SQLitePath:    t.TempDir() + "/monitor.db",
			RaftDir:       t.TempDir() + "/raft",
			RetentionDays: 30,
		},
	}

	st, err := store.Open(cfg.Storage.SQLitePath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	manager, err := cluster.NewManager(cfg, st, logger)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = manager.Shutdown(ctx)
	}()

	waitForLeader(t, manager)

	submitter := cluster.NewSubmitter(manager, cfg)
	alertResolver := notify.NewResolver(cfg, st, logger)
	server, err := New(cfg, st, manager, submitter, alertResolver, logger)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	status, meta, cookie := requestJSON(t, ts.Client(), http.MethodGet, ts.URL+"/api/v1/meta", nil, "")
	if status != http.StatusOK {
		t.Fatalf("meta status = %d", status)
	}
	if meta["admin_initialized"] != false {
		t.Fatalf("expected uninitialized admin, got %+v", meta)
	}
	if meta["node_name"] != "Shanghai-A" {
		t.Fatalf("expected node_name in meta, got %+v", meta)
	}
	if cookie != "" {
		t.Fatalf("unexpected cookie on anonymous meta")
	}

	status, _, cookie = requestJSON(t, ts.Client(), http.MethodPost, ts.URL+"/api/v1/admin/bootstrap", map[string]any{
		"password": "supersecret-password",
	}, "")
	if status != http.StatusCreated {
		t.Fatalf("bootstrap status = %d", status)
	}
	if cookie == "" {
		t.Fatal("expected bootstrap session cookie")
	}

	status, meta, _ = requestJSON(t, ts.Client(), http.MethodGet, ts.URL+"/api/v1/meta", nil, cookie)
	if status != http.StatusOK {
		t.Fatalf("admin meta status = %d", status)
	}
	if meta["is_admin"] != true {
		t.Fatalf("expected admin session after bootstrap, got %+v", meta)
	}

	status, members, _ := requestJSONArray(t, ts.Client(), http.MethodGet, ts.URL+"/api/v1/admin/members", nil, cookie)
	if status != http.StatusOK {
		t.Fatalf("list members status = %d", status)
	}
	if len(members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(members))
	}
	if got := members[0]["node_id"]; got != "node-a" {
		t.Fatalf("expected member node_id node-a, got %v", got)
	}
	if got := members[0]["current_role"]; got != "voter" {
		t.Fatalf("expected current_role voter, got %v", got)
	}

	status, _, _ = requestJSON(t, ts.Client(), http.MethodPut, ts.URL+"/api/v1/admin/members/node-a/role", map[string]any{
		"role": "nonvoter",
	}, cookie)
	if status != http.StatusConflict {
		t.Fatalf("demote last voter status = %d, want %d", status, http.StatusConflict)
	}

	status, _, _ = requestJSON(t, ts.Client(), http.MethodDelete, ts.URL+"/api/v1/admin/members/node-a", nil, cookie)
	if status != http.StatusConflict {
		t.Fatalf("remove last voter status = %d, want %d", status, http.StatusConflict)
	}

	status, _, _ = requestJSON(t, ts.Client(), http.MethodPut, ts.URL+"/api/v1/admin/nodes/node-a", map[string]any{
		"display_name": "Primary Shanghai",
	}, cookie)
	if status != http.StatusOK {
		t.Fatalf("update node display name status = %d", status)
	}

	status, nodeNames, _ := requestJSONArray(t, ts.Client(), http.MethodGet, ts.URL+"/api/v1/admin/nodes", nil, cookie)
	if status != http.StatusOK {
		t.Fatalf("list node names status = %d", status)
	}
	if len(nodeNames) != 1 {
		t.Fatalf("expected 1 node name entry, got %d", len(nodeNames))
	}
	if got := nodeNames[0]["display_name"]; got != "Primary Shanghai" {
		t.Fatalf("expected override display_name, got %v", got)
	}
	if got := nodeNames[0]["effective_display_name"]; got != "Primary Shanghai" {
		t.Fatalf("expected effective_display_name, got %v", got)
	}

	status, meta, _ = requestJSON(t, ts.Client(), http.MethodGet, ts.URL+"/api/v1/meta", nil, cookie)
	if status != http.StatusOK {
		t.Fatalf("meta after rename status = %d", status)
	}
	if got := meta["node_name"]; got != "Primary Shanghai" {
		t.Fatalf("expected renamed node_name in meta, got %v", got)
	}

	status, _, _ = requestJSON(t, ts.Client(), http.MethodPost, ts.URL+"/api/v1/admin/bootstrap", map[string]any{
		"password": "another-password",
	}, "")
	if status != http.StatusConflict {
		t.Fatalf("second bootstrap status = %d", status)
	}

	status, _, _ = requestJSON(t, ts.Client(), http.MethodPost, ts.URL+"/api/v1/admin/checks", map[string]any{
		"name":         "sshd",
		"type":         "systemd",
		"enabled":      true,
		"scope_mode":   "include_nodes",
		"node_ids":     []string{"node-a"},
		"service_name": "ssh",
	}, cookie)
	if status != http.StatusCreated {
		t.Fatalf("create check status = %d", status)
	}

	status, checks, _ := requestJSONArray(t, ts.Client(), http.MethodGet, ts.URL+"/api/v1/admin/checks", nil, cookie)
	if status != http.StatusOK {
		t.Fatalf("list checks status = %d", status)
	}
	if len(checks) != 1 {
		t.Fatalf("expected 1 check, got %d", len(checks))
	}
	if got := checks[0]["scope_mode"]; got != "include_nodes" {
		t.Fatalf("expected scope_mode include_nodes, got %v", got)
	}
	nodeIDs, ok := checks[0]["node_ids"].([]any)
	if !ok || len(nodeIDs) != 1 || nodeIDs[0] != "node-a" {
		t.Fatalf("expected node_ids [node-a], got %#v", checks[0]["node_ids"])
	}

	_, err = manager.Apply(context.Background(), cluster.CommandIngress, model.IngressState{
		ActiveNodeID: "node-a",
		DesiredIP:    "203.0.113.10",
		DNSSynced:    true,
		DNSSyncedAt:  time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("apply ingress: %v", err)
	}

	status, ingress, _ := requestJSON(t, ts.Client(), http.MethodGet, ts.URL+"/api/v1/ingress", nil, "")
	if status != http.StatusOK {
		t.Fatalf("public ingress status = %d", status)
	}
	if got := ingress["desired_ip"]; got != "" {
		t.Fatalf("expected redacted desired_ip, got %v", got)
	}

	status, ingress, _ = requestJSON(t, ts.Client(), http.MethodGet, ts.URL+"/api/v1/ingress", nil, cookie)
	if status != http.StatusOK {
		t.Fatalf("admin ingress status = %d", status)
	}
	if got := ingress["active_node_name"]; got != "Primary Shanghai" {
		t.Fatalf("expected active_node_name, got %v", got)
	}
	if got := ingress["desired_ip"]; got != "203.0.113.10" {
		t.Fatalf("expected visible desired_ip, got %v", got)
	}

	_, err = manager.Apply(context.Background(), cluster.CommandNodeState, model.NodeState{
		NodeID:          "node-a",
		Status:          model.StatusHealthy,
		Reason:          "fresh heartbeat and observer reachability confirmed",
		RuleKey:         "healthy",
		LastHeartbeatAt: time.Now().UTC(),
		LastProbeSummary: model.ProbeSummary{
			SuccessfulPeers: 1,
			TotalPeers:      1,
			ExpectedPeers:   3,
			Reachable:       true,
			LastSources:     []string{"node-b"},
		},
		ReplicatedFresh: true,
		LastEvaluatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("apply node state: %v", err)
	}

	status, clusterPayload, _ := requestJSON(t, ts.Client(), http.MethodGet, ts.URL+"/api/v1/cluster", nil, "")
	if status != http.StatusOK {
		t.Fatalf("cluster status = %d", status)
	}
	if got := clusterPayload["node_name"]; got != "Primary Shanghai" {
		t.Fatalf("expected cluster node_name, got %v", got)
	}
	nodesRaw, ok := clusterPayload["nodes"].([]any)
	if !ok || len(nodesRaw) == 0 {
		t.Fatalf("expected cluster nodes, got %+v", clusterPayload["nodes"])
	}
	firstNode, ok := nodesRaw[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first node object, got %#v", nodesRaw[0])
	}
	if got := firstNode["node_name"]; got != "Primary Shanghai" {
		t.Fatalf("expected node_name on cluster node, got %v", got)
	}
	summary, ok := firstNode["last_probe_summary"].(map[string]any)
	if !ok {
		t.Fatalf("expected last_probe_summary, got %#v", firstNode["last_probe_summary"])
	}
	if got := summary["expected_peers"]; got != float64(3) {
		t.Fatalf("expected expected_peers 3, got %v", got)
	}

	status, nodeDetail, _ := requestJSON(t, ts.Client(), http.MethodGet, ts.URL+"/api/v1/nodes/node-a", nil, "")
	if status != http.StatusOK {
		t.Fatalf("node detail status = %d", status)
	}
	statePayload, ok := nodeDetail["state"].(map[string]any)
	if !ok {
		t.Fatalf("expected state payload, got %#v", nodeDetail["state"])
	}
	summary, ok = statePayload["last_probe_summary"].(map[string]any)
	if !ok {
		t.Fatalf("expected node detail last_probe_summary, got %#v", statePayload["last_probe_summary"])
	}
	if got := summary["expected_peers"]; got != float64(3) {
		t.Fatalf("expected node detail expected_peers 3, got %v", got)
	}

	status, _, _ = requestJSON(t, ts.Client(), http.MethodPost, ts.URL+"/api/v1/admin/logout", map[string]any{}, cookie)
	if status != http.StatusOK {
		t.Fatalf("logout status = %d", status)
	}

	status, _, _ = requestJSON(t, ts.Client(), http.MethodGet, ts.URL+"/api/v1/admin/checks", nil, cookie)
	if status != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized checks after logout, got %d", status)
	}

	status, _, cookie = requestJSON(t, ts.Client(), http.MethodPost, ts.URL+"/api/v1/admin/login", map[string]any{
		"password": "supersecret-password",
	}, "")
	if status != http.StatusOK {
		t.Fatalf("login status = %d", status)
	}
	if cookie == "" {
		t.Fatal("expected new login cookie")
	}
}

func TestAdminAlertSettingsRuntimeOverrideAndReset(t *testing.T) {
	t.Parallel()

	raftAddr := freeTCPAddr(t)
	cfg := &config.Config{
		Cluster: config.ClusterConfig{
			NodeID:   "node-a",
			RaftAddr: raftAddr,
			Peers: []config.ClusterPeer{
				{
					NodeID:      "node-a",
					DisplayName: "Shanghai-A",
					APIAddr:     "127.0.0.1:8443",
					RaftAddr:    raftAddr,
					PublicIPv4:  "203.0.113.10",
					Priority:    100,
				},
			},
			Priority: 100,
		},
		Network: config.NetworkConfig{
			ListenAddr:      "127.0.0.1:8443",
			PublicIPv4:      "203.0.113.10",
			PublicHTTPSPort: 443,
		},
		Storage: config.StorageConfig{
			DataDir:       t.TempDir(),
			SQLitePath:    t.TempDir() + "/monitor.db",
			RaftDir:       t.TempDir() + "/raft",
			RetentionDays: 30,
		},
	}

	st, err := store.Open(cfg.Storage.SQLitePath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	manager, err := cluster.NewManager(cfg, st, logger)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = manager.Shutdown(ctx)
	}()

	waitForLeader(t, manager)

	submitter := cluster.NewSubmitter(manager, cfg)
	server, err := New(cfg, st, manager, submitter, notify.NewResolver(cfg, st, logger), logger)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	status, _, cookie := requestJSON(t, ts.Client(), http.MethodPost, ts.URL+"/api/v1/admin/bootstrap", map[string]any{
		"password": "supersecret-password",
	}, "")
	if status != http.StatusCreated {
		t.Fatalf("bootstrap status = %d", status)
	}

	status, _, _ = requestJSON(t, ts.Client(), http.MethodPut, ts.URL+"/api/v1/admin/alerts/telegram", map[string]any{
		"enabled":         true,
		"bot_token":       "runtime-telegram-token",
		"chat_id":         "123456789",
		"parse_mode":      "Markdown",
		"request_timeout": "12s",
	}, cookie)
	if status != http.StatusOK {
		t.Fatalf("put telegram alert status = %d", status)
	}

	status, alerts, _ := requestJSON(t, ts.Client(), http.MethodGet, ts.URL+"/api/v1/admin/alerts", nil, cookie)
	if status != http.StatusOK {
		t.Fatalf("get admin alerts status = %d", status)
	}
	telegram, ok := alerts["telegram"].(map[string]any)
	if !ok {
		t.Fatalf("expected telegram alert view, got %#v", alerts["telegram"])
	}
	if telegram["source"] != "runtime" {
		t.Fatalf("telegram source = %v, want runtime", telegram["source"])
	}
	if telegram["enabled"] != true {
		t.Fatalf("telegram enabled = %v, want true", telegram["enabled"])
	}
	if telegram["secret_configured"] != true {
		t.Fatalf("telegram secret_configured = %v, want true", telegram["secret_configured"])
	}

	status, meta, _ := requestJSON(t, ts.Client(), http.MethodGet, ts.URL+"/api/v1/meta", nil, cookie)
	if status != http.StatusOK {
		t.Fatalf("meta status = %d", status)
	}
	channels, ok := meta["test_alert_channels"].([]any)
	if !ok || len(channels) != 1 || channels[0] != "telegram" {
		t.Fatalf("test_alert_channels = %#v, want [telegram]", meta["test_alert_channels"])
	}

	status, _, _ = requestJSON(t, ts.Client(), http.MethodPost, ts.URL+"/api/v1/admin/password", map[string]any{
		"current_password": "supersecret-password",
		"new_password":     "another-secret-password",
	}, cookie)
	if status != http.StatusOK {
		t.Fatalf("password update status = %d", status)
	}

	status, alerts, _ = requestJSON(t, ts.Client(), http.MethodGet, ts.URL+"/api/v1/admin/alerts", nil, cookie)
	if status != http.StatusOK {
		t.Fatalf("get admin alerts after password status = %d", status)
	}
	telegram, ok = alerts["telegram"].(map[string]any)
	if !ok || telegram["source"] != "runtime" || telegram["enabled"] != true {
		t.Fatalf("telegram alert after password update = %#v", alerts["telegram"])
	}

	status, alerts, _ = requestJSON(t, ts.Client(), http.MethodDelete, ts.URL+"/api/v1/admin/alerts/telegram", nil, cookie)
	if status != http.StatusOK {
		t.Fatalf("delete telegram alert status = %d", status)
	}
	telegram, ok = alerts["telegram"].(map[string]any)
	if !ok {
		t.Fatalf("expected telegram alert view after delete, got %#v", alerts["telegram"])
	}
	if telegram["source"] != "config" {
		t.Fatalf("telegram source after delete = %v, want config", telegram["source"])
	}
	if telegram["enabled"] != false {
		t.Fatalf("telegram enabled after delete = %v, want false", telegram["enabled"])
	}

	status, meta, _ = requestJSON(t, ts.Client(), http.MethodGet, ts.URL+"/api/v1/meta", nil, cookie)
	if status != http.StatusOK {
		t.Fatalf("meta after delete status = %d", status)
	}
	channels, ok = meta["test_alert_channels"].([]any)
	if !ok || len(channels) != 0 {
		t.Fatalf("test_alert_channels after delete = %#v, want []", meta["test_alert_channels"])
	}
}

func requestJSON(t *testing.T, client *http.Client, method, url string, payload any, cookie string) (int, map[string]any, string) {
	t.Helper()

	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		body = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	out := map[string]any{}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	setCookie := ""
	if len(resp.Cookies()) > 0 {
		setCookie = resp.Cookies()[0].Name + "=" + resp.Cookies()[0].Value
	}
	return resp.StatusCode, out, setCookie
}

func requestJSONArray(t *testing.T, client *http.Client, method, url string, payload any, cookie string) (int, []map[string]any, string) {
	t.Helper()

	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	var out []map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	setCookie := ""
	if len(resp.Cookies()) > 0 {
		setCookie = resp.Cookies()[0].Name + "=" + resp.Cookies()[0].Value
	}
	return resp.StatusCode, out, setCookie
}

func waitForLeader(t *testing.T, manager *cluster.Manager) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if manager.IsLeader() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("raft leader not elected in time")
}

func freeTCPAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen free addr: %v", err)
	}
	defer ln.Close()
	return ln.Addr().String()
}
