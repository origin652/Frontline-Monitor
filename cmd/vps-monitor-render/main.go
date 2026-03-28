package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"

	"vps-monitor/internal/config"
	"vps-monitor/internal/inventory"
)

func main() {
	var inventoryPath string
	var outputDir string
	var nodeID string

	flag.StringVar(&inventoryPath, "inventory", "cluster.inventory.yaml", "path to cluster inventory yaml")
	flag.StringVar(&outputDir, "out", "generated-configs", "output directory for rendered monitor.yaml files")
	flag.StringVar(&nodeID, "node", "", "render only the specified node_id")
	flag.Parse()

	inv, err := inventory.Load(inventoryPath)
	if err != nil {
		log.Fatalf("load inventory failed: %v", err)
	}

	rendered := map[string]*config.Config{}
	if nodeID != "" {
		cfg, err := inv.RenderNode(nodeID)
		if err != nil {
			log.Fatalf("render node failed: %v", err)
		}
		rendered[nodeID] = cfg
	} else {
		rendered, err = inv.RenderAll()
		if err != nil {
			log.Fatalf("render inventory failed: %v", err)
		}
	}

	if err := writeRenderedConfigs(outputDir, rendered); err != nil {
		log.Fatalf("write rendered configs failed: %v", err)
	}

	fmt.Printf("rendered %d config(s) into %s\n", len(rendered), outputDir)
}

func writeRenderedConfigs(outputDir string, rendered map[string]*config.Config) error {
	nodeIDs := make([]string, 0, len(rendered))
	for nodeID := range rendered {
		nodeIDs = append(nodeIDs, nodeID)
	}
	sort.Strings(nodeIDs)

	for _, nodeID := range nodeIDs {
		cfg := rendered[nodeID]
		dir := filepath.Join(outputDir, nodeID)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}

		body, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}
		content := append([]byte("# Generated from cluster inventory. Manual edits will be overwritten.\n"), body...)
		target := filepath.Join(dir, "monitor.yaml")
		if err := os.WriteFile(target, content, 0o644); err != nil {
			return err
		}
		fmt.Printf("- %s\n", target)
	}
	return nil
}
