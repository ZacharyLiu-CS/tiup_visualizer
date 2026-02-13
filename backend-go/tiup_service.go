package main

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Mapping of component role to its expected log file names.
var roleLogFiles = map[string][]string{
	"tikv":         {"tikv.log", "tikv_stderr.log"},
	"pd":           {"pd.log", "pd_stderr.log"},
	"tidb":         {"tidb.log", "tidb_stderr.log"},
	"grafana":      {"grafana.log"},
	"prometheus":   {"prometheus.log"},
	"alertmanager": {"alertmanager.log"},
}

const cacheTTL = 30 * time.Second

// cacheEntry stores a cached value with timestamp.
type cacheEntry struct {
	value     interface{}
	timestamp time.Time
}

// ttlCache is a simple in-memory TTL cache.
type ttlCache struct {
	mu    sync.RWMutex
	store map[string]cacheEntry
	ttl   time.Duration
}

func newTTLCache(ttl time.Duration) *ttlCache {
	return &ttlCache{
		store: make(map[string]cacheEntry),
		ttl:   ttl,
	}
}

func (c *ttlCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	entry, ok := c.store[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Since(entry.timestamp) > c.ttl {
		c.mu.Lock()
		delete(c.store, key)
		c.mu.Unlock()
		return nil, false
	}
	return entry.value, true
}

func (c *ttlCache) Set(key string, value interface{}) {
	c.mu.Lock()
	c.store[key] = cacheEntry{value: value, timestamp: time.Now()}
	c.mu.Unlock()
}

// TiUPService provides methods to interact with tiup CLI.
type TiUPService struct {
	cache *ttlCache
}

func NewTiUPService() *TiUPService {
	return &TiUPService{
		cache: newTTLCache(cacheTTL),
	}
}

// ExecuteCommand runs a shell command with timeout and returns stdout.
func ExecuteCommand(command string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	out, err := cmd.Output()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("command execution timeout")
	}
	if err != nil {
		return "", fmt.Errorf("command execution failed: %w", err)
	}
	return string(out), nil
}

// GetLogFilesForRole returns expected log file names for a component role.
func GetLogFilesForRole(role string) []string {
	r := strings.ToLower(role)
	if files, ok := roleLogFiles[r]; ok {
		return files
	}
	return []string{r + ".log"}
}

var splitRegex = regexp.MustCompile(`\s{2,}`)

// ParseClusterList parses the output of `tiup cluster list`.
func ParseClusterList(output string) []ClusterInfo {
	var clusters []ClusterInfo
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Name") || strings.HasPrefix(line, "----") {
			continue
		}
		parts := splitRegex.Split(line, -1)
		if len(parts) >= 5 {
			clusters = append(clusters, ClusterInfo{
				Name:       parts[0],
				User:       parts[1],
				Version:    parts[2],
				Path:       parts[3],
				PrivateKey: parts[4],
				Status:     "unknown",
			})
		}
	}
	return clusters
}

// ParseClusterDisplay parses the output of `tiup cluster display <name>`.
func ParseClusterDisplay(output string, clusterName string) *ClusterDetail {
	lines := strings.Split(strings.TrimSpace(output), "\n")

	detail := &ClusterDetail{
		ClusterName: clusterName,
		Components:  []ComponentInfo{},
	}

	componentSection := false
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if !componentSection {
			if strings.HasPrefix(line, "Cluster type:") {
				detail.ClusterType = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
			} else if strings.HasPrefix(line, "Cluster version:") {
				detail.ClusterVersion = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
			} else if strings.HasPrefix(line, "Deploy user:") {
				detail.DeployUser = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
			} else if strings.HasPrefix(line, "SSH type:") {
				detail.SSHType = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
			} else if strings.HasPrefix(line, "Dashboard URL:") {
				v := strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
				detail.DashboardURL = &v
			} else if strings.HasPrefix(line, "Grafana URL:") {
				v := strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
				detail.GrafanaURL = &v
			} else if strings.HasPrefix(line, "ID") && strings.Contains(line, "Role") {
				componentSection = true
				continue
			}
		} else {
			if strings.HasPrefix(line, "--") || strings.HasPrefix(line, "Total nodes:") || line == "" {
				continue
			}
			parts := splitRegex.Split(line, -1)
			if len(parts) >= 8 {
				role := parts[1]
				logFilenames := GetLogFilesForRole(role)
				logFiles := make([]LogFileInfo, len(logFilenames))
				for i, f := range logFilenames {
					logFiles[i] = LogFileInfo{Filename: f, Exists: true}
				}
				detail.Components = append(detail.Components, ComponentInfo{
					ID:        parts[0],
					Role:      parts[1],
					Host:      parts[2],
					Ports:     parts[3],
					OSArch:    parts[4],
					Status:    parts[5],
					DataDir:   parts[6],
					DeployDir: parts[7],
					LogFiles:  logFiles,
				})
			}
		}
	}
	return detail
}

func (s *TiUPService) GetClusterDetail(clusterName string) (*ClusterDetail, error) {
	cacheKey := "detail:" + clusterName
	if v, ok := s.cache.Get(cacheKey); ok {
		return v.(*ClusterDetail), nil
	}

	output, err := ExecuteCommand(fmt.Sprintf("tiup cluster display %s", clusterName), 30*time.Second)
	if err != nil {
		return nil, err
	}
	detail := ParseClusterDisplay(output, clusterName)
	s.cache.Set(cacheKey, detail)
	return detail, nil
}

func (s *TiUPService) getClusterList() ([]ClusterInfo, error) {
	cacheKey := "cluster_list"
	if v, ok := s.cache.Get(cacheKey); ok {
		return v.([]ClusterInfo), nil
	}

	output, err := ExecuteCommand("tiup cluster list", 30*time.Second)
	if err != nil {
		return nil, err
	}
	clusters := ParseClusterList(output)
	s.cache.Set(cacheKey, clusters)
	return clusters, nil
}

func (s *TiUPService) getAllDetails() map[string]*ClusterDetail {
	clusters, err := s.getClusterList()
	if err != nil {
		slog.Warn("Error getting cluster list", "error", err)
		return nil
	}
	details := make(map[string]*ClusterDetail)
	for _, c := range clusters {
		detail, err := s.GetClusterDetail(c.Name)
		if err != nil {
			slog.Warn("Error getting cluster detail", "cluster", c.Name, "error", err)
			continue
		}
		details[c.Name] = detail
	}
	return details
}

// GetAllClusters returns all clusters with computed health status.
func (s *TiUPService) GetAllClusters() ([]ClusterInfo, error) {
	clusters, err := s.getClusterList()
	if err != nil {
		return nil, err
	}

	details := s.getAllDetails()
	result := make([]ClusterInfo, len(clusters))
	for i, c := range clusters {
		result[i] = ClusterInfo{
			Name:       c.Name,
			User:       c.User,
			Version:    c.Version,
			Path:       c.Path,
			PrivateKey: c.PrivateKey,
			Status:     "unknown",
		}

		if detail, ok := details[c.Name]; ok && len(detail.Components) > 0 {
			hasUp := false
			allUp := true
			for _, comp := range detail.Components {
				if strings.Contains(comp.Status, "Up") {
					hasUp = true
				} else {
					allUp = false
				}
			}
			if allUp {
				result[i].Status = "healthy"
			} else if hasUp {
				result[i].Status = "partial"
			} else {
				result[i].Status = "unhealthy"
			}
		}
	}
	return result, nil
}

// GetAllHosts aggregates components by physical host.
func (s *TiUPService) GetAllHosts() (map[string]*HostInfo, error) {
	details := s.getAllDetails()
	hostsMap := make(map[string]*HostInfo)

	for clusterName, detail := range details {
		for _, comp := range detail.Components {
			host, ok := hostsMap[comp.Host]
			if !ok {
				host = &HostInfo{
					Host:       comp.Host,
					Components: []ComponentInfo{},
					Clusters:   []string{},
				}
				hostsMap[comp.Host] = host
			}
			host.Components = append(host.Components, comp)
			// Add cluster name if not already present
			found := false
			for _, cn := range host.Clusters {
				if cn == clusterName {
					found = true
					break
				}
			}
			if !found {
				host.Clusters = append(host.Clusters, clusterName)
			}
		}
	}
	return hostsMap, nil
}

// GetLogFilePath returns the log file path and the component info.
func (s *TiUPService) GetLogFilePath(clusterName, componentID, filename string) (string, *ComponentInfo, error) {
	detail, err := s.GetClusterDetail(clusterName)
	if err != nil {
		return "", nil, err
	}

	var component *ComponentInfo
	for i := range detail.Components {
		if detail.Components[i].ID == componentID {
			component = &detail.Components[i]
			break
		}
	}
	if component == nil {
		return "", nil, fmt.Errorf("component %s not found in cluster %s", componentID, clusterName)
	}

	allowedFiles := GetLogFilesForRole(component.Role)
	allowed := false
	for _, f := range allowedFiles {
		if f == filename {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", nil, fmt.Errorf("log file %s not allowed for role %s", filename, component.Role)
	}

	logPath := filepath.Join(component.DeployDir, "log", filename)
	return logPath, component, nil
}
