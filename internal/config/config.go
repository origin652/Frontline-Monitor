package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"vps-monitor/internal/model"
)

type Config struct {
	Cluster    ClusterConfig    `yaml:"cluster"`
	Network    NetworkConfig    `yaml:"network"`
	Cloudflare CloudflareConfig `yaml:"cloudflare"`
	Checks     ChecksConfig     `yaml:"checks"`
	Runtime    RuntimeConfig    `yaml:"runtime"`
	Thresholds Thresholds       `yaml:"thresholds"`
	Alerts     AlertsConfig     `yaml:"alerts"`
	Storage    StorageConfig    `yaml:"storage"`
}

type ClusterConfig struct {
	NodeID           string        `yaml:"node_id"`
	APIAddr          string        `yaml:"api_addr,omitempty"`
	DisplayName      string        `yaml:"display_name,omitempty"`
	RaftAddr         string        `yaml:"raft_addr"`
	RaftBindAddr     string        `yaml:"raft_bind_addr,omitempty"`
	Peers            []ClusterPeer `yaml:"peers"`
	Priority         int           `yaml:"priority"`
	IngressCandidate *bool         `yaml:"ingress_candidate,omitempty"`
	Role             string        `yaml:"role,omitempty"`
	JoinSeeds        []string      `yaml:"join_seeds,omitempty"`
	Bootstrap        bool          `yaml:"bootstrap,omitempty"`
	InternalTokenEnv string        `yaml:"internal_token_env,omitempty"`
}

type ClusterPeer struct {
	NodeID           string `yaml:"node_id"`
	DisplayName      string `yaml:"display_name,omitempty"`
	APIAddr          string `yaml:"api_addr"`
	RaftAddr         string `yaml:"raft_addr"`
	PublicIPv4       string `yaml:"public_ipv4"`
	Priority         int    `yaml:"priority"`
	IngressCandidate *bool  `yaml:"ingress_candidate,omitempty"`
}

type NetworkConfig struct {
	ListenAddr      string `yaml:"listen_addr"`
	PublicIPv4      string `yaml:"public_ipv4"`
	PublicHTTPSPort int    `yaml:"public_https_port"`
	TLSCertFile     string `yaml:"tls_cert_file"`
	TLSKeyFile      string `yaml:"tls_key_file"`
}

type CloudflareConfig struct {
	Hostname      string `yaml:"hostname"`
	ZoneID        string `yaml:"zone_id"`
	DNSRecordID   string `yaml:"dns_record_id"`
	APITokenEnv   string `yaml:"api_token_env"`
	Enabled       bool   `yaml:"enabled"`
	RequestTimout string `yaml:"request_timeout"`
}

type ChecksConfig struct {
	Services     []string    `yaml:"services"`
	TCPPorts     []int       `yaml:"tcp_ports"`
	HTTPChecks   []HTTPCheck `yaml:"http_checks"`
	DockerChecks []string    `yaml:"docker_checks"`
}

type HTTPCheck struct {
	Name         string `yaml:"name"`
	Scheme       string `yaml:"scheme"`
	Path         string `yaml:"path"`
	Port         int    `yaml:"port"`
	ExpectStatus int    `yaml:"expect_status"`
	Timeout      string `yaml:"timeout"`
}

type RuntimeConfig struct {
	LoopInterval            string `yaml:"loop_interval,omitempty"`
	ProbeObserversPerTarget int    `yaml:"probe_observers_per_target,omitempty"`
}

type Thresholds struct {
	CPUWarn  float64 `yaml:"cpu_warn"`
	CPUCrit  float64 `yaml:"cpu_crit"`
	MemWarn  float64 `yaml:"mem_warn"`
	MemCrit  float64 `yaml:"mem_crit"`
	DiskWarn float64 `yaml:"disk_warn"`
	DiskCrit float64 `yaml:"disk_crit"`
}

type AlertsConfig struct {
	Telegram TelegramConfig     `yaml:"telegram"`
	SMTP     SMTPConfig         `yaml:"smtp"`
	WeCom    WebhookAlertConfig `yaml:"wecom_or_feishu"`
}

type TelegramConfig struct {
	Enabled     bool   `yaml:"enabled"`
	BotTokenEnv string `yaml:"bot_token_env"`
	BotToken    string `yaml:"-"`
	ChatID      string `yaml:"chat_id"`
	ParseMode   string `yaml:"parse_mode"`
	RequestTout string `yaml:"request_timeout"`
}

type SMTPConfig struct {
	Enabled       bool     `yaml:"enabled"`
	Host          string   `yaml:"host"`
	Port          int      `yaml:"port"`
	Username      string   `yaml:"username"`
	PasswordEnv   string   `yaml:"password_env"`
	Password      string   `yaml:"-"`
	From          string   `yaml:"from"`
	To            []string `yaml:"to"`
	RequestTout   string   `yaml:"request_timeout"`
	UseStartTLS   bool     `yaml:"use_starttls"`
	SubjectPrefix string   `yaml:"subject_prefix"`
}

type WebhookAlertConfig struct {
	Enabled     bool   `yaml:"enabled"`
	WebhookURL  string `yaml:"webhook_url"`
	SecretEnv   string `yaml:"secret_env"`
	Secret      string `yaml:"-"`
	RequestTout string `yaml:"request_timeout"`
	TitlePrefix string `yaml:"title_prefix"`
}

type StorageConfig struct {
	DataDir       string `yaml:"data_dir"`
	SQLitePath    string `yaml:"sqlite_path"`
	RaftDir       string `yaml:"raft_dir"`
	RetentionDays int    `yaml:"retention_days"`
}

func Load(path string) (*Config, error) {
	if path == "" {
		path = "monitor.yaml"
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := defaultConfig(filepath.Dir(path))
	if err := yaml.Unmarshal(raw, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func defaultConfig(baseDir string) *Config {
	dataDir := filepath.Join(baseDir, "data")
	return &Config{
		Network: NetworkConfig{
			ListenAddr:      ":8443",
			PublicHTTPSPort: 443,
		},
		Cloudflare: CloudflareConfig{
			RequestTimout: "10s",
		},
		Thresholds: Thresholds{
			CPUWarn:  80,
			CPUCrit:  92,
			MemWarn:  85,
			MemCrit:  95,
			DiskWarn: 80,
			DiskCrit: 92,
		},
		Storage: StorageConfig{
			DataDir:       dataDir,
			SQLitePath:    filepath.Join(dataDir, "monitor.db"),
			RaftDir:       filepath.Join(dataDir, "raft"),
			RetentionDays: 30,
		},
	}
}

func (c *Config) Validate() error {
	return c.validate(true)
}

func (c *Config) ValidateForRender() error {
	return c.validate(false)
}

func (c *Config) validate(requireRuntimeSecrets bool) error {
	var problems []string
	if c.Cluster.NodeID == "" {
		problems = append(problems, "cluster.node_id is required")
	}
	if c.Cluster.RaftAddr == "" {
		problems = append(problems, "cluster.raft_addr is required")
	}
	if c.Network.ListenAddr == "" {
		problems = append(problems, "network.listen_addr is required")
	}
	if c.Network.PublicIPv4 == "" {
		problems = append(problems, "network.public_ipv4 is required")
	}
	if c.Storage.SQLitePath == "" || c.Storage.RaftDir == "" {
		problems = append(problems, "storage paths are required")
	}
	if raw := strings.TrimSpace(c.Runtime.LoopInterval); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil || parsed <= 0 {
			problems = append(problems, "runtime.loop_interval must be a positive duration")
		}
	}
	if c.Runtime.ProbeObserversPerTarget < 0 {
		problems = append(problems, "runtime.probe_observers_per_target must be zero or a positive integer")
	}
	if c.UsesStaticPeers() {
		nodeIDs := map[string]struct{}{}
		foundSelf := false
		for _, peer := range c.Cluster.Peers {
			if peer.NodeID == "" || peer.APIAddr == "" || peer.RaftAddr == "" || peer.PublicIPv4 == "" {
				problems = append(problems, "each cluster peer needs node_id, api_addr, raft_addr, and public_ipv4")
				continue
			}
			if _, exists := nodeIDs[peer.NodeID]; exists {
				problems = append(problems, "cluster.peers contains duplicate node_id "+peer.NodeID)
			}
			nodeIDs[peer.NodeID] = struct{}{}
			if peer.NodeID == c.Cluster.NodeID {
				foundSelf = true
			}
		}
		if !foundSelf {
			problems = append(problems, "cluster.peers must contain the local node")
		}
	} else {
		if strings.TrimSpace(c.Cluster.APIAddr) == "" {
			problems = append(problems, "cluster.api_addr is required when cluster.peers is empty")
		}
		if !model.IsValidClusterMemberRole(c.Cluster.Role) {
			problems = append(problems, "cluster.role must be voter or nonvoter")
		}
		if !c.Cluster.Bootstrap && len(c.NormalizedJoinSeeds()) == 0 {
			problems = append(problems, "dynamic mode requires cluster.bootstrap=true or at least one cluster.join_seeds entry")
		}
		if requireRuntimeSecrets && c.InternalToken() == "" {
			problems = append(problems, "dynamic mode requires the internal token environment variable to be set")
		}
	}

	if c.Cloudflare.Enabled {
		if c.Cloudflare.Hostname == "" || c.Cloudflare.ZoneID == "" || c.Cloudflare.DNSRecordID == "" || c.Cloudflare.APITokenEnv == "" {
			problems = append(problems, "cloudflare enabled but hostname/zone_id/dns_record_id/api_token_env missing")
		}
	}

	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

func (c *Config) SelfPeer() ClusterPeer {
	for _, peer := range c.Cluster.Peers {
		if peer.NodeID == c.Cluster.NodeID {
			return peer
		}
	}
	return ClusterPeer{
		NodeID:           c.Cluster.NodeID,
		DisplayName:      c.DefaultDisplayName(),
		APIAddr:          c.APIAddr(),
		RaftAddr:         c.Cluster.RaftAddr,
		PublicIPv4:       c.Network.PublicIPv4,
		Priority:         c.Cluster.Priority,
		IngressCandidate: c.Cluster.IngressCandidate,
	}
}

func (c *Config) RaftBindAddr() string {
	if bind := strings.TrimSpace(c.Cluster.RaftBindAddr); bind != "" {
		return bind
	}
	return strings.TrimSpace(c.Cluster.RaftAddr)
}

func (c *Config) PeerByID(nodeID string) (ClusterPeer, bool) {
	for _, peer := range c.Cluster.Peers {
		if peer.NodeID == nodeID {
			return peer, true
		}
	}
	if strings.TrimSpace(nodeID) == strings.TrimSpace(c.Cluster.NodeID) {
		return c.SelfPeer(), true
	}
	return ClusterPeer{}, false
}

func (c *Config) LeaderAPIAddr(leaderID string) string {
	if leaderID == "" {
		return ""
	}
	peer, ok := c.PeerByID(leaderID)
	if !ok {
		return ""
	}
	return peer.APIAddr
}

func (c *Config) PeerDisplayName(nodeID string) string {
	peer, ok := c.PeerByID(nodeID)
	if !ok {
		return nodeID
	}
	return peer.DisplayNameOrNodeID()
}

func (c *Config) OrderedPeers() []ClusterPeer {
	peers := append([]ClusterPeer(nil), c.Cluster.Peers...)
	if len(peers) == 0 {
		peers = append(peers, c.SelfPeer())
	}
	slices.SortFunc(peers, func(a, b ClusterPeer) int {
		if a.Priority != b.Priority {
			return b.Priority - a.Priority
		}
		return strings.Compare(a.NodeID, b.NodeID)
	})
	return peers
}

func (c *Config) CloudflareTimeout() time.Duration {
	return parseDurationOr(c.Cloudflare.RequestTimout, 10*time.Second)
}

func (c *Config) HTTPCheckTimeout(check HTTPCheck) time.Duration {
	return parseDurationOr(check.Timeout, 4*time.Second)
}

func (c *Config) TelegramTimeout() time.Duration {
	return parseDurationOr(c.Alerts.Telegram.RequestTout, 10*time.Second)
}

func (c *Config) SMTPTimeout() time.Duration {
	return parseDurationOr(c.Alerts.SMTP.RequestTout, 15*time.Second)
}

func (c *Config) WebhookTimeout() time.Duration {
	return parseDurationOr(c.Alerts.WeCom.RequestTout, 10*time.Second)
}

func (c *Config) LoopInterval() time.Duration {
	return parsePositiveDurationOr(c.Runtime.LoopInterval, 15*time.Second)
}

func (c *Config) ProbeObserversPerTarget() int {
	if c.Runtime.ProbeObserversPerTarget < 0 {
		return 0
	}
	return c.Runtime.ProbeObserversPerTarget
}

func parseDurationOr(raw string, fallback time.Duration) time.Duration {
	if raw == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

func parsePositiveDurationOr(raw string, fallback time.Duration) time.Duration {
	parsed := parseDurationOr(raw, fallback)
	if parsed <= 0 {
		return fallback
	}
	return parsed
}

func (c *Config) InternalToken() string {
	env := c.Cluster.InternalTokenEnv
	if env == "" {
		env = "MONITOR_INTERNAL_TOKEN"
	}
	return os.Getenv(env)
}

func (c *Config) UsesStaticPeers() bool {
	return len(c.Cluster.Peers) > 0
}

func (c *Config) UsesDynamicMembership() bool {
	return !c.UsesStaticPeers()
}

func (c *Config) APIAddr() string {
	if apiAddr := strings.TrimSpace(c.Cluster.APIAddr); apiAddr != "" {
		return apiAddr
	}
	if peer, ok := c.PeerByID(c.Cluster.NodeID); ok && strings.TrimSpace(peer.APIAddr) != "" {
		return strings.TrimSpace(peer.APIAddr)
	}
	return strings.TrimSpace(c.Network.ListenAddr)
}

func (c *Config) DefaultDisplayName() string {
	if strings.TrimSpace(c.Cluster.DisplayName) != "" {
		return strings.TrimSpace(c.Cluster.DisplayName)
	}
	return c.Cluster.NodeID
}

func (c *Config) NormalizedRole() string {
	return model.NormalizeClusterMemberRole(c.Cluster.Role)
}

func (c *Config) NormalizedJoinSeeds() []string {
	out := make([]string, 0, len(c.Cluster.JoinSeeds))
	seen := map[string]struct{}{}
	for _, seed := range c.Cluster.JoinSeeds {
		seed = strings.TrimSpace(seed)
		if seed == "" {
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

func (p ClusterPeer) DisplayNameOrNodeID() string {
	if strings.TrimSpace(p.DisplayName) != "" {
		return strings.TrimSpace(p.DisplayName)
	}
	return p.NodeID
}

func (p ClusterPeer) IsIngressCandidate() bool {
	if p.IngressCandidate == nil {
		return true
	}
	return *p.IngressCandidate
}
