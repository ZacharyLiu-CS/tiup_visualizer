package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// ClusterConfigService handles cluster config read/write/reload operations.
type ClusterConfigService struct{}

// NewClusterConfigService creates a new service instance.
func NewClusterConfigService() *ClusterConfigService {
	return &ClusterConfigService{}
}

// GetConfig reads the current cluster topology config (meta.yaml topology section).
func (s *ClusterConfigService) GetConfig(clusterName string) (string, error) {
	metaPath := filepath.Join(os.Getenv("HOME"), ".tiup", "storage", "cluster", "clusters", clusterName, "meta.yaml")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return "", fmt.Errorf("failed to read meta.yaml for cluster %q: %w", clusterName, err)
	}
	return string(data), nil
}

// SaveConfig writes the updated meta.yaml directly to disk.
func (s *ClusterConfigService) SaveConfig(clusterName string, metaYAML string) (string, error) {
	metaPath := filepath.Join(os.Getenv("HOME"), ".tiup", "storage", "cluster", "clusters", clusterName, "meta.yaml")

	// Backup the current config
	backupPath := metaPath + ".backup"
	if err := copyFile(metaPath, backupPath); err != nil {
		slog.Warn("ClusterConfig: failed to backup meta.yaml", "error", err)
	}

	slog.Info("ClusterConfig: saving config", "cluster", clusterName, "path", metaPath)

	if err := os.WriteFile(metaPath, []byte(metaYAML), 0644); err != nil {
		return "", fmt.Errorf("failed to write meta.yaml: %w", err)
	}

	slog.Info("ClusterConfig: save success", "cluster", clusterName)
	return "Config saved", nil
}

// ReloadCluster runs `tiup cluster reload` for the given cluster.
func (s *ClusterConfigService) ReloadCluster(clusterName string) (string, error) {
	cmdStr := fmt.Sprintf("tiup cluster reload %s -y", clusterName)
	slog.Info("ClusterConfig: reloading cluster", "cluster", clusterName, "command", cmdStr)

	// Use exec.Command directly for longer timeout and streaming potential
	ctx := exec.Command("bash", "-c", cmdStr)
	out, err := ctx.CombinedOutput()
	output := string(out)

	if err != nil {
		slog.Error("ClusterConfig: reload failed", "cluster", clusterName, "error", err, "output", truncate(output, 500))
		return output, fmt.Errorf("reload failed: %w", err)
	}

	slog.Info("ClusterConfig: reload success", "cluster", clusterName)
	return output, nil
}

// ExportAsTemplate extracts the topology section from meta.yaml, suitable for use
// as a cluster creation template (tiup cluster deploy). The meta.yaml contains
// user/tidb_version/topology at top level, but deploy only needs the topology content.
// It also saves the topology to the cluster create history directory.
func (s *ClusterConfigService) ExportAsTemplate(clusterName string, historyDir string, operator string) (string, error) {
	// Read meta.yaml
	metaPath := filepath.Join(os.Getenv("HOME"), ".tiup", "storage", "cluster", "clusters", clusterName, "meta.yaml")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return "", fmt.Errorf("failed to read meta.yaml for cluster %q: %w", clusterName, err)
	}

	// Parse meta.yaml to extract topology section
	var meta struct {
		Topology interface{} `yaml:"topology"`
	}
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return "", fmt.Errorf("failed to parse meta.yaml: %w", err)
	}
	if meta.Topology == nil {
		return "", fmt.Errorf("no topology section found in meta.yaml for cluster %q", clusterName)
	}

	// Marshal topology back to YAML
	topoData, err := yaml.Marshal(meta.Topology)
	if err != nil {
		return "", fmt.Errorf("failed to marshal topology: %w", err)
	}
	topoYAML := string(topoData)

	// Save to history directory with timestamp to avoid overwriting previous exports
	ts := time.Now().Format("20060102_150405")
	safeName := sanitizeFileName(fmt.Sprintf("%s_%s", clusterName, ts))
	configPath := filepath.Join(historyDir, safeName+".yaml")
	if err := os.WriteFile(configPath, []byte(topoYAML), 0644); err != nil {
		slog.Warn("ClusterConfig: failed to save exported template", "path", configPath, "error", err)
	} else {
		slog.Info("ClusterConfig: template exported", "cluster", clusterName, "path", configPath, "user", operator)
	}

	return topoYAML, nil
}

// (truncate is defined in update_service.go)

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
