package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ClusterCreateService handles cluster creation operations.
type ClusterCreateService struct {
	historyDir string
}

// NewClusterCreateService creates a new service instance.
func NewClusterCreateService(execDir string) *ClusterCreateService {
	dir := filepath.Join(execDir, "create_cluster_history")
	if err := os.MkdirAll(dir, 0755); err != nil {
		slog.Error("Failed to create cluster history directory", "dir", dir, "error", err)
	}
	return &ClusterCreateService{historyDir: dir}
}

// DeployRequest is the JSON request body for deploying a cluster.
type DeployRequest struct {
	ClusterName string `json:"cluster_name"`
	Version     string `json:"version"`
	Username    string `json:"username"`
	Config      string `json:"config"` // YAML content (pasted)
}

// HistoryEntry represents one saved config file entry.
type HistoryEntry struct {
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	CreatedAt  string `json:"created_at"`
	Username   string `json:"username"`
}

// DeployCluster saves the config and runs tiup cluster deploy.
func (s *ClusterCreateService) DeployCluster(req DeployRequest) (string, error) {
	clusterName := strings.TrimSpace(req.ClusterName)
	version := strings.TrimSpace(req.Version)
	username := strings.TrimSpace(req.Username)
	config := req.Config

	if clusterName == "" {
		return "", fmt.Errorf("cluster_name is required")
	}
	if version == "" {
		return "", fmt.Errorf("version is required")
	}
	if username == "" {
		return "", fmt.Errorf("username is required")
	}
	if config == "" {
		return "", fmt.Errorf("config is required")
	}

	// Sanitize cluster name for filename
	safeName := sanitizeFileName(clusterName)
	configPath := filepath.Join(s.historyDir, safeName+".yaml")

	// Save config file
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		return "", fmt.Errorf("failed to save config file: %w", err)
	}
	slog.Info("ClusterCreate: config saved", "path", configPath)

	// Build and execute tiup command
	cmd := fmt.Sprintf("tiup cluster deploy %s %s %s --user %s -y",
		clusterName, version, configPath, username)
	slog.Info("ClusterCreate: deploying cluster", "cluster", clusterName,
		"version", version, "username", username, "command", cmd)

	output, err := ExecuteCommand(cmd, 30*time.Minute)
	if err != nil {
		slog.Error("ClusterCreate: deploy failed", "cluster", clusterName,
			"error", err, "output", truncate(output, 500))
		return output, err
	}

	slog.Info("ClusterCreate: deploy success", "cluster", clusterName,
		"output", truncate(output, 200))

	// Invalidate tiup cache after deploy
	return output, nil
}

// UploadAndDeploy handles file upload and deploys the cluster.
func (s *ClusterCreateService) UploadAndDeploy(clusterName, version, username string, file io.Reader) (string, error) {
	clusterName = strings.TrimSpace(clusterName)
	version = strings.TrimSpace(version)
	username = strings.TrimSpace(username)

	if clusterName == "" {
		return "", fmt.Errorf("cluster_name is required")
	}
	if version == "" {
		return "", fmt.Errorf("version is required")
	}
	if username == "" {
		return "", fmt.Errorf("username is required")
	}

	// Read file content
	configBytes, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read uploaded file: %w", err)
	}
	if len(configBytes) == 0 {
		return "", fmt.Errorf("uploaded file is empty")
	}

	return s.DeployCluster(DeployRequest{
		ClusterName: clusterName,
		Version:     version,
		Username:    username,
		Config:      string(configBytes),
	})
}

// ListHistory returns all saved config files.
func (s *ClusterCreateService) ListHistory() ([]HistoryEntry, error) {
	entries, err := os.ReadDir(s.historyDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []HistoryEntry{}, nil
		}
		return nil, fmt.Errorf("failed to read history directory: %w", err)
	}

	var result []HistoryEntry
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		result = append(result, HistoryEntry{
			Name:      name,
			Size:      info.Size(),
			CreatedAt: info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	// Sort by creation time descending
	sort.Slice(result, func(i, j int) bool {
		ti, _ := time.Parse("2006-01-02 15:04:05", result[i].CreatedAt)
		tj, _ := time.Parse("2006-01-02 15:04:05", result[j].CreatedAt)
		return tj.Before(ti)
	})

	if result == nil {
		result = []HistoryEntry{}
	}
	return result, nil
}

// GetConfigContent returns the content of a saved config file.
func (s *ClusterCreateService) GetConfigContent(name string) (string, error) {
	safeName := sanitizeFileName(name)
	// Try .yaml first then .yml
	for _, ext := range []string{".yaml", ".yml"} {
		path := filepath.Join(s.historyDir, safeName+ext)
		data, err := os.ReadFile(path)
		if err == nil {
			return string(data), nil
		}
	}
	return "", fmt.Errorf("config file not found for cluster %q", name)
}

// DeleteConfig removes a saved config file.
func (s *ClusterCreateService) DeleteConfig(name string) error {
	safeName := sanitizeFileName(name)
	deleted := false
	for _, ext := range []string{".yaml", ".yml"} {
		path := filepath.Join(s.historyDir, safeName+ext)
		if err := os.Remove(path); err == nil {
			deleted = true
			slog.Info("ClusterCreate: config deleted", "name", name, "path", path)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete config %q: %w", name, err)
		}
	}
	if !deleted {
		return fmt.Errorf("config file not found for cluster %q", name)
	}
	return nil
}

// GetConfigPath returns the full filesystem path for a config name.
func (s *ClusterCreateService) GetConfigPath(name string) string {
	safeName := sanitizeFileName(name)
	for _, ext := range []string{".yaml", ".yml"} {
		path := filepath.Join(s.historyDir, safeName+ext)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// --- HTTP Handlers ---

func (s *Server) handleClusterCreateDeploy(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "multipart/form-data") {
		// File upload mode
		if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max
			writeError(w, http.StatusBadRequest, fmt.Sprintf("failed to parse form: %v", err))
			return
		}
		clusterName := r.FormValue("cluster_name")
		version := r.FormValue("version")
		username := r.FormValue("username")
		file, _, err := r.FormFile("config_file")
		if err != nil {
			writeError(w, http.StatusBadRequest, "config_file is required")
			return
		}
		defer file.Close()

		output, err := s.clusterCreate.UploadAndDeploy(clusterName, version, username, file)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("deploy failed: %s\nOutput: %s", err.Error(), output))
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message":      "cluster deployed successfully",
			"cluster_name": clusterName,
			"output":       output,
		})
		return
	}

	// JSON mode (pasted config)
	var req DeployRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user := r.Header.Get("X-Username")
	slog.Info("ClusterCreate: deploy request", "cluster", req.ClusterName,
		"version", req.Version, "username", req.Username, "operator", user)

	output, err := s.clusterCreate.DeployCluster(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("deploy failed: %s\nOutput: %s", err.Error(), output))
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "cluster deployed successfully",
		"cluster_name": req.ClusterName,
		"output":       output,
	})
}

func (s *Server) handleClusterCreateHistory(w http.ResponseWriter, r *http.Request) {
	entries, err := s.clusterCreate.ListHistory()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"configs": entries,
	})
}

func (s *Server) handleClusterGetConfig(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	action := r.URL.Query().Get("action")
	if action == "" {
		action = "view"
	}

	// Security check
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		writeError(w, http.StatusBadRequest, "Invalid name")
		return
	}

	content, err := s.clusterCreate.GetConfigContent(name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	if action == "download" {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.yaml", name))
		w.Write([]byte(content))
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Write([]byte(content))
	}
}

func (s *Server) handleClusterDeleteConfig(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	// Security check
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		writeError(w, http.StatusBadRequest, "Invalid name")
		return
	}

	if err := s.clusterCreate.DeleteConfig(name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Info("ClusterCreate: config deleted", "name", name, "user", r.Header.Get("X-Username"))
	writeJSON(w, http.StatusOK, map[string]string{"message": "config deleted", "name": name})
}

func sanitizeFileName(name string) string {
	// Remove path separators and dangerous characters
	replacer := strings.NewReplacer(
		"..", "",
		"/", "",
		"\\", "",
		" ", "_",
	)
	s := replacer.Replace(name)
	// Only keep alphanumeric, dash, underscore, dot
	var result strings.Builder
	for _, ch := range s {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '-' || ch == '_' || ch == '.' {
			result.WriteRune(ch)
		}
	}
	return result.String()
}
