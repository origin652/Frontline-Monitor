package model

import "time"

const (
	StatusHealthy  = "healthy"
	StatusDegraded = "degraded"
	StatusCritical = "critical"
	StatusUnknown  = "unknown"
)

const (
	IncidentStatusActive   = "active"
	IncidentStatusResolved = "resolved"
)

type ServiceCheck struct {
	ID        string    `json:"id,omitempty"`
	Type      string    `json:"type,omitempty"`
	Name      string    `json:"name"`
	Target    string    `json:"target,omitempty"`
	Status    string    `json:"status"`
	Detail    string    `json:"detail"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DockerCheck struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Detail    string    `json:"detail"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PortResult struct {
	Port      int       `json:"port"`
	Open      bool      `json:"open"`
	LatencyMS int64     `json:"latency_ms"`
	CheckedAt time.Time `json:"checked_at"`
}

type HTTPCheckResult struct {
	Name       string    `json:"name"`
	URL        string    `json:"url"`
	OK         bool      `json:"ok"`
	StatusCode int       `json:"status_code"`
	LatencyMS  int64     `json:"latency_ms"`
	CheckedAt  time.Time `json:"checked_at"`
}

type NodeHeartbeat struct {
	NodeID          string            `json:"node_id"`
	CollectedAt     time.Time         `json:"collected_at"`
	CPUPct          float64           `json:"cpu_pct"`
	MemPct          float64           `json:"mem_pct"`
	DiskPct         float64           `json:"disk_pct"`
	Load1           float64           `json:"load1"`
	UptimeS         uint64            `json:"uptime_s"`
	Services        []ServiceCheck    `json:"services"`
	DockerChecks    []DockerCheck     `json:"docker_checks"`
	LocalHTTPChecks []HTTPCheckResult `json:"local_http_checks"`
	CPUModel        string            `json:"cpu_model,omitempty"`
	CPUCores        int               `json:"cpu_cores,omitempty"`
	MemTotalMB      uint64            `json:"mem_total_mb,omitempty"`
	DiskTotalMB     uint64            `json:"disk_total_mb,omitempty"`
	OS              string            `json:"os,omitempty"`
	Kernel          string            `json:"kernel,omitempty"`
}

type ProbeObservation struct {
	SourceNodeID   string            `json:"source_node_id"`
	SourceNodeName string            `json:"source_node_name,omitempty"`
	TargetNodeID   string            `json:"target_node_id"`
	TargetNodeName string            `json:"target_node_name,omitempty"`
	CollectedAt    time.Time         `json:"collected_at"`
	TCP22OK        bool              `json:"tcp_22_ok"`
	TCP443OK       bool              `json:"tcp_443_ok"`
	HTTPOK         bool              `json:"http_ok"`
	SSHBannerMS    int64             `json:"ssh_banner_ms"`
	Ports          []PortResult      `json:"ports"`
	HTTPChecks     []HTTPCheckResult `json:"http_checks"`
}

type ProbeSummary struct {
	SuccessfulPeers int      `json:"successful_peers"`
	TotalPeers      int      `json:"total_peers"`
	Reachable       bool     `json:"reachable"`
	LastSources     []string `json:"last_sources"`
}

type NodeState struct {
	NodeID           string         `json:"node_id"`
	NodeName         string         `json:"node_name,omitempty"`
	Status           string         `json:"status"`
	Reason           string         `json:"reason"`
	RuleKey          string         `json:"rule_key"`
	LastHeartbeatAt  time.Time      `json:"last_heartbeat_at"`
	LastProbeSummary ProbeSummary   `json:"last_probe_summary"`
	ReplicatedFresh  bool           `json:"replicated_fresh"`
	CPUPct           float64        `json:"cpu_pct"`
	MemPct           float64        `json:"mem_pct"`
	DiskPct          float64        `json:"disk_pct"`
	Load1            float64        `json:"load1"`
	UptimeS          uint64         `json:"uptime_s"`
	Services         []ServiceCheck `json:"services"`
	BadStreak        int            `json:"bad_streak"`
	GoodStreak       int            `json:"good_streak"`
	LastEvaluatedAt  time.Time      `json:"last_evaluated_at"`
	PrimaryEvidence  []string       `json:"primary_evidence"`
}

type Incident struct {
	ID             string     `json:"id"`
	NodeID         string     `json:"node_id"`
	NodeName       string     `json:"node_name,omitempty"`
	RuleKey        string     `json:"rule_key"`
	Severity       string     `json:"severity"`
	Status         string     `json:"status"`
	Summary        string     `json:"summary"`
	Detail         string     `json:"detail"`
	OpenedAt       time.Time  `json:"opened_at"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
	LastNotifiedAt *time.Time `json:"last_notified_at,omitempty"`
}

type AlertDelivery struct {
	IncidentID  string     `json:"incident_id"`
	Channel     string     `json:"channel"`
	DeliveryKey string     `json:"delivery_key"`
	Status      string     `json:"status"`
	Response    string     `json:"response"`
	CreatedAt   time.Time  `json:"created_at"`
	SentAt      *time.Time `json:"sent_at,omitempty"`
}

type AlertClaim struct {
	IncidentID  string    `json:"incident_id"`
	Channel     string    `json:"channel"`
	DeliveryKey string    `json:"delivery_key"`
	CreatedAt   time.Time `json:"created_at"`
}

type AlertCompletion struct {
	IncidentID  string    `json:"incident_id"`
	DeliveryKey string    `json:"delivery_key"`
	Status      string    `json:"status"`
	Response    string    `json:"response"`
	SentAt      time.Time `json:"sent_at"`
}

type Event struct {
	ID        string         `json:"id"`
	Kind      string         `json:"kind"`
	Severity  string         `json:"severity"`
	NodeID    string         `json:"node_id"`
	NodeName  string         `json:"node_name,omitempty"`
	Title     string         `json:"title"`
	Body      string         `json:"body"`
	Meta      map[string]any `json:"meta"`
	CreatedAt time.Time      `json:"created_at"`
}

type IngressState struct {
	ActiveNodeID   string    `json:"active_node_id"`
	ActiveNodeName string    `json:"active_node_name,omitempty"`
	DesiredIP      string    `json:"desired_ip"`
	DNSSynced      bool      `json:"dns_synced"`
	DNSSyncedAt    time.Time `json:"dns_synced_at"`
	LastDNSError   string    `json:"last_dns_error"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type NodeDetail struct {
	State     NodeState          `json:"state"`
	Heartbeat *NodeHeartbeat     `json:"heartbeat"`
	Probes    []ProbeObservation `json:"probes"`
	Incidents []Incident         `json:"incidents"`
	History   []MetricPoint      `json:"history"`
}

type MetricPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

type ClusterSnapshot struct {
	GeneratedAt time.Time    `json:"generated_at"`
	NodeID      string       `json:"node_id"`
	NodeName    string       `json:"node_name,omitempty"`
	LeaderID    string       `json:"leader_id"`
	LeaderName  string       `json:"leader_name,omitempty"`
	Ingress     IngressState `json:"ingress"`
	Nodes       []NodeState  `json:"nodes"`
	Incidents   []Incident   `json:"incidents"`
	Events      []Event      `json:"events"`
}
