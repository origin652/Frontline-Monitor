package inventory

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"vps-monitor/internal/config"
	"vps-monitor/internal/model"
)

type Inventory struct {
	Cluster InventoryCluster `yaml:"cluster"`
	Shared  SharedConfig     `yaml:"shared"`
	Nodes   []Node           `yaml:"nodes"`
}

type InventoryCluster struct {
	Mode             string `yaml:"mode,omitempty"`
	InternalTokenEnv string `yaml:"internal_token_env"`
}

type SharedConfig struct {
	Network    config.NetworkConfig    `yaml:"network"`
	Cloudflare config.CloudflareConfig `yaml:"cloudflare"`
	Checks     config.ChecksConfig     `yaml:"checks"`
	Thresholds config.Thresholds       `yaml:"thresholds"`
	Alerts     config.AlertsConfig     `yaml:"alerts"`
	Storage    config.StorageConfig    `yaml:"storage"`
}

type Node struct {
	NodeID           string                `yaml:"node_id"`
	DisplayName      string                `yaml:"display_name"`
	APIAddr          string                `yaml:"api_addr"`
	RaftAddr         string                `yaml:"raft_addr"`
	RaftBindAddr     string                `yaml:"raft_bind_addr"`
	PublicIPv4       string                `yaml:"public_ipv4"`
	Priority         int                   `yaml:"priority"`
	IngressCandidate *bool                 `yaml:"ingress_candidate"`
	Role             string                `yaml:"role,omitempty"`
	Network          *config.NetworkConfig `yaml:"network"`
	Checks           *config.ChecksConfig  `yaml:"checks"`
	Storage          *config.StorageConfig `yaml:"storage"`
}

func Load(path string) (*Inventory, error) {
	if strings.TrimSpace(path) == "" {
		path = "cluster.inventory.yaml"
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read inventory: %w", err)
	}

	inv := defaultInventory()
	if err := yaml.Unmarshal(raw, inv); err != nil {
		return nil, fmt.Errorf("parse inventory: %w", err)
	}
	if err := inv.Validate(); err != nil {
		return nil, err
	}
	return inv, nil
}

func defaultInventory() *Inventory {
	return &Inventory{
		Shared: SharedConfig{
			Network: config.NetworkConfig{
				ListenAddr:      ":8443",
				PublicHTTPSPort: 443,
			},
			Cloudflare: config.CloudflareConfig{
				RequestTimout: "10s",
			},
			Thresholds: config.Thresholds{
				CPUWarn:  80,
				CPUCrit:  92,
				MemWarn:  85,
				MemCrit:  95,
				DiskWarn: 80,
				DiskCrit: 92,
			},
			Storage: config.StorageConfig{
				DataDir:       "/var/lib/vps-monitor",
				SQLitePath:    "/var/lib/vps-monitor/monitor.db",
				RaftDir:       "/var/lib/vps-monitor/raft",
				RetentionDays: 30,
			},
		},
	}
}

func (i *Inventory) Validate() error {
	var problems []string
	if len(i.Nodes) == 0 {
		problems = append(problems, "nodes must contain at least one node")
	}

	seenNodeIDs := map[string]struct{}{}
	for idx, node := range i.Nodes {
		prefix := fmt.Sprintf("nodes[%d]", idx)
		if strings.TrimSpace(node.NodeID) == "" {
			problems = append(problems, prefix+".node_id is required")
		}
		if strings.TrimSpace(node.APIAddr) == "" {
			problems = append(problems, prefix+".api_addr is required")
		}
		if strings.TrimSpace(node.RaftAddr) == "" {
			problems = append(problems, prefix+".raft_addr is required")
		}
		if strings.TrimSpace(node.PublicIPv4) == "" {
			problems = append(problems, prefix+".public_ipv4 is required")
		}
		if _, exists := seenNodeIDs[node.NodeID]; exists && strings.TrimSpace(node.NodeID) != "" {
			problems = append(problems, "duplicate node_id "+node.NodeID)
		}
		seenNodeIDs[node.NodeID] = struct{}{}
		if !model.IsValidClusterMemberRole(node.Role) {
			problems = append(problems, prefix+".role must be voter or nonvoter")
		}
	}

	if mode := i.membershipMode(); mode != inventoryModeStatic && mode != inventoryModeDynamic {
		problems = append(problems, "cluster.mode must be static or dynamic")
	}

	for _, node := range i.Nodes {
		cfg, err := i.renderNode(node.NodeID)
		if err != nil {
			problems = append(problems, fmt.Sprintf("render %s failed: %v", node.NodeID, err))
			continue
		}
		if err := cfg.ValidateForRender(); err != nil {
			problems = append(problems, fmt.Sprintf("rendered config for %s is invalid: %v", node.NodeID, err))
		}
	}

	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

func (i *Inventory) RenderNode(nodeID string) (*config.Config, error) {
	cfg, err := i.renderNode(nodeID)
	if err != nil {
		return nil, err
	}
	if err := cfg.ValidateForRender(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (i *Inventory) RenderAll() (map[string]*config.Config, error) {
	out := make(map[string]*config.Config, len(i.Nodes))
	for _, node := range i.Nodes {
		cfg, err := i.RenderNode(node.NodeID)
		if err != nil {
			return nil, err
		}
		out[node.NodeID] = cfg
	}
	return out, nil
}

func (i *Inventory) renderNode(nodeID string) (*config.Config, error) {
	node, ok := i.nodeByID(nodeID)
	if !ok {
		return nil, fmt.Errorf("unknown node %q", nodeID)
	}

	clusterCfg := config.ClusterConfig{
		NodeID:           node.NodeID,
		RaftAddr:         node.RaftAddr,
		RaftBindAddr:     node.RaftBindAddr,
		Priority:         node.Priority,
		InternalTokenEnv: strings.TrimSpace(i.Cluster.InternalTokenEnv),
	}
	if i.membershipMode() == inventoryModeDynamic {
		clusterCfg.APIAddr = strings.TrimSpace(node.APIAddr)
		clusterCfg.DisplayName = strings.TrimSpace(node.DisplayName)
		clusterCfg.IngressCandidate = node.IngressCandidate
		clusterCfg.Role = model.NormalizeClusterMemberRole(node.Role)
		clusterCfg.Bootstrap = i.bootstrapNodeID() == node.NodeID
		if !clusterCfg.Bootstrap {
			clusterCfg.JoinSeeds = i.joinSeeds(node.NodeID)
		}
	} else {
		clusterCfg.Peers = i.peers()
	}

	cfg := &config.Config{
		Cluster:    clusterCfg,
		Network:    mergeNetwork(i.Shared.Network, node.Network, node.PublicIPv4),
		Cloudflare: i.Shared.Cloudflare,
		Checks:     mergeChecks(i.Shared.Checks, node.Checks),
		Thresholds: i.Shared.Thresholds,
		Alerts:     i.Shared.Alerts,
		Storage:    mergeStorage(i.Shared.Storage, node.Storage),
	}
	return cfg, nil
}

const (
	inventoryModeStatic  = "static"
	inventoryModeDynamic = "dynamic"
)

func (i *Inventory) membershipMode() string {
	switch strings.ToLower(strings.TrimSpace(i.Cluster.Mode)) {
	case "", inventoryModeStatic:
		return inventoryModeStatic
	case inventoryModeDynamic:
		return inventoryModeDynamic
	default:
		return strings.ToLower(strings.TrimSpace(i.Cluster.Mode))
	}
}

func (i *Inventory) bootstrapNodeID() string {
	if len(i.Nodes) == 0 {
		return ""
	}
	return strings.TrimSpace(i.Nodes[0].NodeID)
}

func (i *Inventory) nodeByID(nodeID string) (Node, bool) {
	for _, node := range i.Nodes {
		if node.NodeID == nodeID {
			return node, true
		}
	}
	return Node{}, false
}

func (i *Inventory) peers() []config.ClusterPeer {
	peers := make([]config.ClusterPeer, 0, len(i.Nodes))
	for _, node := range i.Nodes {
		peers = append(peers, config.ClusterPeer{
			NodeID:           node.NodeID,
			DisplayName:      strings.TrimSpace(node.DisplayName),
			APIAddr:          strings.TrimSpace(node.APIAddr),
			RaftAddr:         strings.TrimSpace(node.RaftAddr),
			PublicIPv4:       strings.TrimSpace(node.PublicIPv4),
			Priority:         node.Priority,
			IngressCandidate: node.IngressCandidate,
		})
	}
	return peers
}

func (i *Inventory) joinSeeds(nodeID string) []string {
	out := make([]string, 0, len(i.Nodes)-1)
	seen := map[string]struct{}{}
	for _, node := range i.Nodes {
		if strings.TrimSpace(node.NodeID) == strings.TrimSpace(nodeID) {
			continue
		}
		apiAddr := strings.TrimSpace(node.APIAddr)
		if apiAddr == "" {
			continue
		}
		if _, ok := seen[apiAddr]; ok {
			continue
		}
		seen[apiAddr] = struct{}{}
		out = append(out, apiAddr)
	}
	return out
}

func mergeNetwork(base config.NetworkConfig, override *config.NetworkConfig, publicIPv4 string) config.NetworkConfig {
	out := base
	if override != nil {
		if listenAddr := strings.TrimSpace(override.ListenAddr); listenAddr != "" {
			out.ListenAddr = listenAddr
		}
		if override.PublicHTTPSPort > 0 {
			out.PublicHTTPSPort = override.PublicHTTPSPort
		}
		if certFile := strings.TrimSpace(override.TLSCertFile); certFile != "" {
			out.TLSCertFile = certFile
		}
		if keyFile := strings.TrimSpace(override.TLSKeyFile); keyFile != "" {
			out.TLSKeyFile = keyFile
		}
	}
	out.PublicIPv4 = strings.TrimSpace(publicIPv4)
	return out
}

func mergeChecks(base config.ChecksConfig, override *config.ChecksConfig) config.ChecksConfig {
	out := config.ChecksConfig{
		Services:     slices.Clone(base.Services),
		TCPPorts:     slices.Clone(base.TCPPorts),
		HTTPChecks:   slices.Clone(base.HTTPChecks),
		DockerChecks: slices.Clone(base.DockerChecks),
	}
	if override == nil {
		return out
	}
	if override.Services != nil {
		out.Services = slices.Clone(override.Services)
	}
	if override.TCPPorts != nil {
		out.TCPPorts = slices.Clone(override.TCPPorts)
	}
	if override.HTTPChecks != nil {
		out.HTTPChecks = slices.Clone(override.HTTPChecks)
	}
	if override.DockerChecks != nil {
		out.DockerChecks = slices.Clone(override.DockerChecks)
	}
	return out
}

func mergeStorage(base config.StorageConfig, override *config.StorageConfig) config.StorageConfig {
	out := base
	if override == nil {
		return out
	}
	if dataDir := strings.TrimSpace(override.DataDir); dataDir != "" {
		out.DataDir = dataDir
	}
	if sqlitePath := strings.TrimSpace(override.SQLitePath); sqlitePath != "" {
		out.SQLitePath = sqlitePath
	}
	if raftDir := strings.TrimSpace(override.RaftDir); raftDir != "" {
		out.RaftDir = raftDir
	}
	if override.RetentionDays > 0 {
		out.RetentionDays = override.RetentionDays
	}
	return out
}
