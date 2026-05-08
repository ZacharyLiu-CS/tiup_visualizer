package main

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// cleanOutputLine strips ANSI codes and handles \r (carriage return) based progress.
// tiup uses \r to overwrite the current line for spinners — we only keep the last segment.
func cleanOutputLine(raw string) string {
	cleaned := stripANSI(raw) // stripANSI defined in pdctl_service.go
	// Handle \r: keep only the text after the last \r
	if idx := strings.LastIndex(cleaned, "\r"); idx >= 0 {
		cleaned = cleaned[idx+1:]
	}
	return cleaned
}

// ClusterCreateService handles cluster creation operations.
type ClusterCreateService struct {
	historyDir string
	jobs       sync.Map // map[string]*DeployJob
}

// DeployJob tracks an async deploy operation.
type DeployJob struct {
	ID       string
	mu       sync.RWMutex
	lines    []string
	done     bool
	exitErr  error
	notify   chan struct{} // buffered(1): pinged when new lines or done
	finished chan struct{} // closed when job is fully done
}

func newDeployJob() *DeployJob {
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	return &DeployJob{
		ID:       id,
		notify:   make(chan struct{}, 1),
		finished: make(chan struct{}),
	}
}

func (j *DeployJob) appendLine(line string) {
	j.mu.Lock()
	j.lines = append(j.lines, line)
	j.mu.Unlock()
	select {
	case j.notify <- struct{}{}:
	default:
	}
}

func (j *DeployJob) finish(err error) {
	j.mu.Lock()
	j.done = true
	j.exitErr = err
	j.mu.Unlock()
	select {
	case j.notify <- struct{}{}:
	default:
	}
	close(j.finished)
}

// snapshot returns lines starting from fromOffset, plus done/error state.
func (j *DeployJob) snapshot(fromOffset int) (lines []string, done bool, err error) {
	j.mu.RLock()
	defer j.mu.RUnlock()
	if fromOffset < len(j.lines) {
		lines = make([]string, len(j.lines)-fromOffset)
		copy(lines, j.lines[fromOffset:])
	}
	return lines, j.done, j.exitErr
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
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"created_at"`
	Username  string `json:"username"`
	MD5       string `json:"md5"`
}

// StartDeployJob validates params, saves config, and starts an async deploy job.
// Returns the job immediately; caller should stream output via the job's SSE endpoint.
func (s *ClusterCreateService) StartDeployJob(req DeployRequest) (*DeployJob, error) {
	clusterName := strings.TrimSpace(req.ClusterName)
	version := strings.TrimSpace(req.Version)
	username := strings.TrimSpace(req.Username)
	config := req.Config

	if clusterName == "" {
		return nil, fmt.Errorf("cluster_name is required")
	}
	if version == "" {
		return nil, fmt.Errorf("version is required")
	}
	if username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if config == "" {
		return nil, fmt.Errorf("config is required")
	}

	// Sanitize cluster name for filename
	safeName := sanitizeFileName(clusterName)
	configPath := filepath.Join(s.historyDir, safeName+".yaml")

	// Save config file
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		return nil, fmt.Errorf("failed to save config file: %w", err)
	}
	slog.Info("ClusterCreate: config saved", "path", configPath)

	job := newDeployJob()
	s.jobs.Store(job.ID, job)

	go s.runDeployJob(job, clusterName, version, configPath, username)

	return job, nil
}

// runDeployJob executes tiup cluster deploy and streams output to the job line by line.
func (s *ClusterCreateService) runDeployJob(job *DeployJob, clusterName, version, configPath, username string) {
	// Clean up job from map after 30 minutes
	defer time.AfterFunc(30*time.Minute, func() {
		s.jobs.Delete(job.ID)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	cmdStr := fmt.Sprintf("tiup cluster deploy %s %s %s --user %s -y",
		clusterName, version, configPath, username)

	slog.Info("ClusterCreate: deploy job started",
		"job", job.ID, "cluster", clusterName, "version", version, "username", username)

	job.appendLine("$ " + cmdStr)
	job.appendLine("")

	cmd := exec.CommandContext(ctx, "bash", "-c", cmdStr)
	// TERM=dumb tells tiup to skip spinner/progress-bar animations.
	// NO_COLOR=1 suppresses color codes.
	cmd.Env = append(os.Environ(), "TERM=dumb", "NO_COLOR=1")

	// Use cmd.StdoutPipe + StderrPipe for more reliable streaming
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		job.appendLine("Error: failed to create stdout pipe: " + err.Error())
		job.finish(err)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		job.appendLine("Error: failed to create stderr pipe: " + err.Error())
		job.finish(err)
		return
	}

	if err := cmd.Start(); err != nil {
		job.appendLine("Error: failed to start command: " + err.Error())
		job.finish(err)
		return
	}

	// Read stdout and stderr concurrently, merge into job lines
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		s.streamReader(job, stdout)
	}()
	go func() {
		defer wg.Done()
		s.streamReader(job, stderr)
	}()

	// Wait for both readers to drain, then wait for process
	wg.Wait()
	waitErr := cmd.Wait()

	if ctx.Err() == context.DeadlineExceeded {
		slog.Error("ClusterCreate: deploy timed out", "job", job.ID, "cluster", clusterName)
		job.finish(fmt.Errorf("command timed out after 30 minutes"))
		return
	}

	if waitErr != nil {
		slog.Error("ClusterCreate: deploy failed", "job", job.ID, "cluster", clusterName, "error", waitErr)
		job.finish(waitErr)
		return
	}

	slog.Info("ClusterCreate: deploy success", "job", job.ID, "cluster", clusterName)
	job.finish(nil)
}

// streamReader reads from r, handles \r and \n, strips ANSI codes,
// and appends cleaned lines to the job.
func (s *ClusterCreateService) streamReader(job *DeployJob, r io.Reader) {
	buf := make([]byte, 4096)
	var lineBuf strings.Builder

	flush := func() {
		if lineBuf.Len() > 0 {
			raw := lineBuf.String()
			lineBuf.Reset()
			line := cleanOutputLine(raw)
			line = strings.TrimSpace(line)
			if line != "" {
				job.appendLine(line)
			}
		}
	}

	for {
		n, err := r.Read(buf)
		if n > 0 {
			chunk := string(buf[:n])
			for _, ch := range chunk {
				switch ch {
				case '\n':
					flush()
				case '\r':
					// Carriage return: treat as overwrite — flush current content
					// then start fresh (simulates terminal \r overwrite behavior)
					flush()
				default:
					lineBuf.WriteRune(ch)
				}
			}
		}
		if err != nil {
			flush() // flush any remaining content
			break
		}
	}
}

// DeployCluster saves the config and runs tiup cluster deploy synchronously.
// Deprecated: use StartDeployJob for async/streaming. Kept for reference.
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

	return output, nil
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

		// Compute MD5 of file content
		fileMD5 := ""
		if data, err := os.ReadFile(filepath.Join(s.historyDir, entry.Name())); err == nil {
			fileMD5 = fmt.Sprintf("%x", md5.Sum(data))
		}

		result = append(result, HistoryEntry{
			Name:      name,
			Size:      info.Size(),
			CreatedAt: info.ModTime().Format("2006-01-02 15:04:05"),
			MD5:       fileMD5,
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
	operator := r.Header.Get("X-Username")

	var job *DeployJob
	var clusterName string

	if strings.Contains(contentType, "multipart/form-data") {
		// File upload mode
		if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max
			writeError(w, http.StatusBadRequest, fmt.Sprintf("failed to parse form: %v", err))
			return
		}
		clusterName = r.FormValue("cluster_name")
		version := r.FormValue("version")
		username := r.FormValue("username")
		file, fileHeader, err := r.FormFile("config_file")
		if err != nil {
			writeError(w, http.StatusBadRequest, "config_file is required")
			return
		}
		defer file.Close()

		slog.Info("ClusterCreate: upload deploy request",
			"cluster", clusterName,
			"version", version,
			"username", username,
			"filename", fileHeader.Filename,
			"size", fileHeader.Size,
			"operator", operator,
		)

		configBytes, err := io.ReadAll(file)
		if err != nil {
			writeError(w, http.StatusBadRequest, "failed to read uploaded file")
			return
		}
		if len(configBytes) == 0 {
			writeError(w, http.StatusBadRequest, "uploaded file is empty")
			return
		}

		job, err = s.clusterCreate.StartDeployJob(DeployRequest{
			ClusterName: clusterName,
			Version:     version,
			Username:    username,
			Config:      string(configBytes),
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	} else {
		// JSON mode (pasted config)
		var req DeployRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		clusterName = req.ClusterName

		slog.Info("ClusterCreate: deploy request",
			"cluster", req.ClusterName,
			"version", req.Version,
			"username", req.Username,
			"operator", operator,
		)

		var err error
		job, err = s.clusterCreate.StartDeployJob(req)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	slog.Info("ClusterCreate: deploy job created", "job", job.ID, "cluster", clusterName)
	writeJSON(w, http.StatusOK, map[string]any{
		"job_id":       job.ID,
		"cluster_name": clusterName,
		"message":      "deploy job started",
	})
}

// handleClusterCreateDeployStream streams deploy output as Server-Sent Events.
func (s *Server) handleClusterCreateDeployStream(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		writeError(w, http.StatusBadRequest, "job_id is required")
		return
	}

	val, ok := s.clusterCreate.jobs.Load(jobID)
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("job %q not found or expired", jobID))
		return
	}
	job := val.(*DeployJob)

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering

	slog.Info("ClusterCreate: SSE client connected", "job", jobID, "user", r.Header.Get("X-Username"))

	ctx := r.Context()
	offset := 0

	// Keepalive ticker: send SSE comment every 10s to keep Nginx from timing out
	keepalive := time.NewTicker(10 * time.Second)
	defer keepalive.Stop()

	for {
		lines, done, exitErr := job.snapshot(offset)
		for _, line := range lines {
			fmt.Fprintf(w, "data: %s\n\n", line)
		}
		offset += len(lines)
		if len(lines) > 0 || done {
			flusher.Flush()
		}

		if done {
			if exitErr != nil {
				errJSON, _ := json.Marshal(map[string]string{"error": exitErr.Error()})
				fmt.Fprintf(w, "event: done\ndata: %s\n\n", errJSON)
			} else {
				fmt.Fprintf(w, "event: done\ndata: {\"success\":true}\n\n")
			}
			flusher.Flush()
			slog.Info("ClusterCreate: SSE stream complete", "job", jobID)
			return
		}

		// Wait for more data, job completion, keepalive tick, or client disconnect
		select {
		case <-ctx.Done():
			slog.Info("ClusterCreate: SSE client disconnected", "job", jobID)
			return
		case <-job.notify:
			// New lines available or job done — loop to drain
		case <-job.finished:
			// Job finished — one final pass to send remaining lines
		case <-keepalive.C:
			// Send SSE comment as keepalive to prevent proxy timeout
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

func (s *Server) handleClusterCreateHistory(w http.ResponseWriter, r *http.Request) {
	slog.Info("ClusterCreate: list history request", "user", r.Header.Get("X-Username"))
	entries, err := s.clusterCreate.ListHistory()
	if err != nil {
		slog.Error("ClusterCreate: list history failed", "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Info("ClusterCreate: list history success", "count", len(entries))
	writeJSON(w, http.StatusOK, map[string]any{
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

	slog.Info("ClusterCreate: get config request", "name", name, "action", action, "user", r.Header.Get("X-Username"))

	content, err := s.clusterCreate.GetConfigContent(name)
	if err != nil {
		slog.Warn("ClusterCreate: get config not found", "name", name, "error", err)
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
