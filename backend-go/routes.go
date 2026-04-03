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
	"strconv"
	"strings"
	"time"
)

// Server holds all dependencies for HTTP handlers.
type Server struct {
	cfg     *AppConfig
	auth    *AuthService
	tiup    *TiUPService
	tikv    *TiKVService
	update  *UpdateService
	execDir string
	version string
	mux     *http.ServeMux
}

func NewServer(cfg *AppConfig, execDir string) *Server {
	s := &Server{
		cfg:     cfg,
		auth:    NewAuthService(cfg),
		tiup:    NewTiUPService(),
		tikv:    NewTiKVService(),
		update:  NewUpdateService(execDir),
		execDir: execDir,
		version: loadVersion(execDir),
		mux:     http.NewServeMux(),
	}
	slog.Info("Build version", "version", s.version)
	s.registerRoutes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) registerRoutes() {
	prefix := s.cfg.APIPrefix

	// Health check (no auth)
	s.mux.HandleFunc("GET /health", s.handleHealth)

	// Version (no auth)
	s.mux.HandleFunc("GET "+prefix+"/version", s.handleVersion)

	// Update routes (auth required)
	s.mux.HandleFunc("GET "+prefix+"/update/check", s.requireAuth(s.handleUpdateCheck))
	s.mux.HandleFunc("POST "+prefix+"/update/apply", s.requireAuth(s.handleUpdateApply))

	// Auth routes (no auth required)
	s.mux.HandleFunc("POST "+prefix+"/auth/login", s.handleLogin)
	s.mux.HandleFunc("GET "+prefix+"/auth/verify", s.handleVerify)

	// Protected routes (auth required)
	s.mux.HandleFunc("GET "+prefix+"/overview", s.requireAuth(s.handleOverview))
	s.mux.HandleFunc("GET "+prefix+"/clusters", s.requireAuth(s.handleClusters))
	s.mux.HandleFunc("GET "+prefix+"/clusters/{clusterName}", s.requireAuth(s.handleClusterDetail))
	s.mux.HandleFunc("GET "+prefix+"/hosts", s.requireAuth(s.handleHosts))
	s.mux.HandleFunc("GET "+prefix+"/hosts/{hostIP}/clusters", s.requireAuth(s.handleHostClusters))
	s.mux.HandleFunc("GET "+prefix+"/logs/{clusterName}/{componentID}/{filename}", s.requireAuth(s.handleLogFile))
	s.mux.HandleFunc("GET "+prefix+"/server-logs", s.requireAuth(s.handleServerLogs))
	s.mux.HandleFunc("GET "+prefix+"/server-logs/{filename}", s.requireAuth(s.handleServerLogFile))
	s.mux.HandleFunc("DELETE "+prefix+"/server-logs/{filename}", s.requireAuth(s.handleServerLogClean))

	// TiKV data access routes (auth required)
	s.mux.HandleFunc("GET "+prefix+"/tikv/{clusterName}/key", s.requireAuth(s.handleTiKVGetKey))
	s.mux.HandleFunc("GET "+prefix+"/tikv/{clusterName}/scan", s.requireAuth(s.handleTiKVScan))
	// TiKV direct PD address routes (auth required)
	s.mux.HandleFunc("GET "+prefix+"/tikv-direct/key", s.requireAuth(s.handleTiKVDirectGetKey))
	s.mux.HandleFunc("GET "+prefix+"/tikv-direct/scan", s.requireAuth(s.handleTiKVDirectScan))

	// WebSocket terminal (GET only, must be before catch-all)
	s.mux.HandleFunc("GET /ws/terminal", s.handleTerminal)

	// Static file serving (SPA) - must be last as it has catch-all
	s.registerStaticRoutes()
}

// requireAuth wraps a handler with JWT authentication.
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenStr := ExtractToken(r)
		if tokenStr == "" {
			writeError(w, http.StatusUnauthorized, "Not authenticated")
			return
		}
		username, ok := s.auth.VerifyToken(tokenStr)
		if !ok {
			writeError(w, http.StatusUnauthorized, "Not authenticated")
			return
		}
		// Store username in header for downstream use
		r.Header.Set("X-Username", username)
		next(w, r)
	}
}

// --- Health ---

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

// --- Version ---

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"version": s.version})
}

// --- Update ---

func (s *Server) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	latest, err := s.update.FetchLatestRelease()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	needUpdate := latest.Version > s.version
	writeJSON(w, http.StatusOK, CheckResult{
		CurrentVersion: s.version,
		LatestRelease:  latest,
		NeedUpdate:     needUpdate,
	})
}

func (s *Server) handleUpdateApply(w http.ResponseWriter, r *http.Request) {
	latest, err := s.update.FetchLatestRelease()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if latest.Version <= s.version {
		writeJSON(w, http.StatusOK, map[string]string{"message": "already up-to-date"})
		return
	}
	// Run update asynchronously so we can return a response before the process restarts
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "update started",
		"version": latest.Version,
	})
	// Flush response before applying
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	go func() {
		if err := s.update.DownloadAndApply(latest); err != nil {
			slog.Error("Update failed", "error", err)
		}
	}()
}

// loadVersion reads the version file next to the binary (or CWD).
func loadVersion(execDir string) string {
	for _, p := range []string{
		filepath.Join(execDir, "version"),
		"version",
	} {
		if data, err := os.ReadFile(p); err == nil {
			return strings.TrimSpace(string(data))
		}
	}
	return "unknown"
}

// --- Auth ---

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if !s.auth.Authenticate(req.Username, req.Password) {
		slog.Warn("Failed login attempt", "username", req.Username)
		writeError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	token, expiresIn, err := s.auth.CreateToken(req.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create token")
		return
	}

	slog.Info("User logged in", "username", req.Username)
	writeJSON(w, http.StatusOK, TokenResponse{
		AccessToken: token,
		TokenType:   "bearer",
		ExpiresIn:   expiresIn,
	})
}

func (s *Server) handleVerify(w http.ResponseWriter, r *http.Request) {
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		writeError(w, http.StatusUnauthorized, "Token required")
		return
	}
	username, ok := s.auth.VerifyToken(tokenStr)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Invalid or expired token")
		return
	}
	writeJSON(w, http.StatusOK, UserInfoResponse{Username: username})
}

// --- Cluster routes ---

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	clusters, err := s.tiup.GetAllClusters()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	hosts, err := s.tiup.GetAllHosts()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"clusters": clusters,
		"hosts":    hosts,
	})
}

func (s *Server) handleClusters(w http.ResponseWriter, r *http.Request) {
	clusters, err := s.tiup.GetAllClusters()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, clusters)
}

func (s *Server) handleClusterDetail(w http.ResponseWriter, r *http.Request) {
	clusterName := r.PathValue("clusterName")
	detail, err := s.tiup.GetClusterDetail(clusterName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) handleHosts(w http.ResponseWriter, r *http.Request) {
	hosts, err := s.tiup.GetAllHosts()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, hosts)
}

func (s *Server) handleHostClusters(w http.ResponseWriter, r *http.Request) {
	hostIP := r.PathValue("hostIP")
	hosts, err := s.tiup.GetAllHosts()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if host, ok := hosts[hostIP]; ok {
		writeJSON(w, http.StatusOK, host.Clusters)
	} else {
		writeJSON(w, http.StatusOK, []string{})
	}
}

// --- Log routes ---

func (s *Server) handleLogFile(w http.ResponseWriter, r *http.Request) {
	clusterName := r.PathValue("clusterName")
	componentID := r.PathValue("componentID")
	filename := r.PathValue("filename")
	action := r.URL.Query().Get("action")
	if action == "" {
		action = "view"
	}
	tailBytes := int64(0)
	if tb := r.URL.Query().Get("tail"); tb != "" {
		if v, err := strconv.ParseInt(tb, 10, 64); err == nil && v > 0 {
			tailBytes = v
		}
	}

	logPath, component, err := s.tiup.GetLogFilePath(clusterName, componentID, filename)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	host := component.Host

	// Get local IPs
	localIPs := getLocalIPs()

	if contains(localIPs, host) || host == "127.0.0.1" || host == "localhost" {
		// Local file
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, fmt.Sprintf("Log file not found: %s", logPath))
			return
		}
		if tailBytes > 0 {
			serveLocalFileTail(w, logPath, filename, action, tailBytes)
		} else {
			serveLocalFile(w, logPath, filename, action)
		}
	} else {
		// Remote file via SSH
		detail, err := s.tiup.GetClusterDetail(clusterName)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		deployUser := detail.DeployUser
		var sshCmd string
		if tailBytes > 0 {
			sshCmd = fmt.Sprintf("ssh -o StrictHostKeyChecking=no -o ConnectTimeout=5 %s@%s 'tail -c %d %s'", deployUser, host, tailBytes, logPath)
		} else {
			sshCmd = fmt.Sprintf("ssh -o StrictHostKeyChecking=no -o ConnectTimeout=5 %s@%s 'cat %s'", deployUser, host, logPath)
		}
		content, err := ExecuteCommand(sshCmd, 30*time.Second)
		if err != nil {
			writeError(w, http.StatusNotFound, fmt.Sprintf("Failed to read log file from %s:%s - %v", host, logPath, err))
			return
		}
		dlName := filename
		if tailBytes > 0 {
			dlName = tailFilename(filename)
		}
		if action == "download" {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", dlName))
			w.Write([]byte(content))
		} else {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Write([]byte(content))
		}
	}
}


func (s *Server) handleServerLogs(w http.ResponseWriter, r *http.Request) {
	logDir := s.cfg.ResolveLogDir(s.execDir)
	if info, err := os.Stat(logDir); err != nil || !info.IsDir() {
		writeJSON(w, http.StatusOK, map[string]interface{}{"files": []interface{}{}})
		return
	}

	entries, err := os.ReadDir(logDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	type fileInfo struct {
		Filename string `json:"filename"`
		Size     int64  `json:"size"`
	}

	var files []fileInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, fileInfo{Filename: entry.Name(), Size: info.Size()})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Filename < files[j].Filename
	})

	if files == nil {
		files = []fileInfo{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"files": files})
}

func (s *Server) handleServerLogFile(w http.ResponseWriter, r *http.Request) {
	filename := r.PathValue("filename")
	action := r.URL.Query().Get("action")
	if action == "" {
		action = "view"
	}

	// Security check
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		writeError(w, http.StatusBadRequest, "Invalid filename")
		return
	}

	logDir := s.cfg.ResolveLogDir(s.execDir)
	logPath := filepath.Join(logDir, filename)

	if info, err := os.Stat(logPath); err != nil || info.IsDir() {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Log file not found: %s", filename))
		return
	}

	serveLocalFile(w, logPath, filename, action)
}

func (s *Server) handleServerLogClean(w http.ResponseWriter, r *http.Request) {
	filename := r.PathValue("filename")

	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		writeError(w, http.StatusBadRequest, "Invalid filename")
		return
	}

	logDir := s.cfg.ResolveLogDir(s.execDir)
	logPath := filepath.Join(logDir, filename)

	if info, err := os.Stat(logPath); err != nil || info.IsDir() {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Log file not found: %s", filename))
		return
	}

	// Truncate file to 0 bytes (keeps the file, clears content)
	if err := os.Truncate(logPath, 0); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to clean log: %v", err))
		return
	}
	slog.Info("Server log cleaned", "filename", filename, "user", r.Header.Get("X-Username"))
	writeJSON(w, http.StatusOK, map[string]string{"message": "log cleared", "filename": filename})
}

// --- Helpers ---

func serveLocalFile(w http.ResponseWriter, logPath, filename, action string) {
	if action == "download" {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		f, err := os.Open(logPath)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer f.Close()
		io.Copy(w, f)
	} else {
		content, err := os.ReadFile(logPath)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write(content)
	}
}

func tailFilename(filename string) string {
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	if ext == "" {
		ext = ".log"
	}
	return base + "_for_ai" + ext
}

func serveLocalFileTail(w http.ResponseWriter, logPath, filename, action string, tailBytes int64) {
	f, err := os.Open(logPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	offset := info.Size() - tailBytes
	if offset < 0 {
		offset = 0
	}

	buf := make([]byte, info.Size()-offset)
	_, err = f.ReadAt(buf, offset)
	if err != nil && err != io.EOF {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	dlName := tailFilename(filename)
	if action == "download" {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", dlName))
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
	w.Write(buf)
}

func getLocalIPs() []string {
	out, err := ExecuteCommand("hostname -I", 5*time.Second)
	if err != nil {
		return nil
	}
	return strings.Fields(strings.TrimSpace(out))
}

func contains(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Detail: detail})
}

// --- TiKV Data Access ---

func (s *Server) handleTiKVGetKey(w http.ResponseWriter, r *http.Request) {
	clusterName := r.PathValue("clusterName")
	key := r.URL.Query().Get("key")
	parseType := r.URL.Query().Get("parse-type")
	cf := r.URL.Query().Get("cf")

	if key == "" {
		writeError(w, http.StatusBadRequest, "missing required query parameter: key")
		return
	}
	if parseType == "" {
		parseType = "graph_meta"
	}
	if cf == "" {
		cf = "default"
	}

	// Get PD addresses from cluster detail
	pdAddrs, err := s.getClusterPDAddrs(clusterName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result, err := s.tikv.GetKey(pdAddrs, cf, key, parseType)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleTiKVScan(w http.ResponseWriter, r *http.Request) {
	clusterName := r.PathValue("clusterName")
	prefix := r.URL.Query().Get("prefix")
	parseType := r.URL.Query().Get("parse-type")
	cf := r.URL.Query().Get("cf")
	limit := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}

	if prefix == "" {
		writeError(w, http.StatusBadRequest, "missing required query parameter: prefix")
		return
	}
	if parseType == "" {
		parseType = "graph_meta"
	}
	if cf == "" {
		cf = "default"
	}

	// Get PD addresses from cluster detail
	pdAddrs, err := s.getClusterPDAddrs(clusterName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	results, err := s.tikv.ScanPrefix(pdAddrs, cf, prefix, limit, parseType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":   len(results),
		"entries": results,
	})
}

// getClusterPDAddrs extracts PD addresses from cluster components.
func (s *Server) getClusterPDAddrs(clusterName string) (string, error) {
	detail, err := s.tiup.GetClusterDetail(clusterName)
	if err != nil {
		return "", fmt.Errorf("failed to get cluster %q detail: %w", clusterName, err)
	}
	pdAddrs := ExtractPDAddrs(detail)
	if pdAddrs == "" {
		return "", fmt.Errorf("no PD component found in cluster %q", clusterName)
	}
	return pdAddrs, nil
}

// --- TiKV Direct (custom PD address) ---

func (s *Server) handleTiKVDirectGetKey(w http.ResponseWriter, r *http.Request) {
	pd := r.URL.Query().Get("pd")
	key := r.URL.Query().Get("key")
	parseType := r.URL.Query().Get("parse-type")
	cf := r.URL.Query().Get("cf")

	if pd == "" {
		writeError(w, http.StatusBadRequest, "missing required query parameter: pd")
		return
	}
	if key == "" {
		writeError(w, http.StatusBadRequest, "missing required query parameter: key")
		return
	}
	if parseType == "" {
		parseType = "graph_meta"
	}
	if cf == "" {
		cf = "default"
	}

	result, err := s.tikv.GetKey(pd, cf, key, parseType)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleTiKVDirectScan(w http.ResponseWriter, r *http.Request) {
	pd := r.URL.Query().Get("pd")
	prefix := r.URL.Query().Get("prefix")
	parseType := r.URL.Query().Get("parse-type")
	cf := r.URL.Query().Get("cf")
	limit := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}

	if pd == "" {
		writeError(w, http.StatusBadRequest, "missing required query parameter: pd")
		return
	}
	if prefix == "" {
		writeError(w, http.StatusBadRequest, "missing required query parameter: prefix")
		return
	}
	if parseType == "" {
		parseType = "graph_meta"
	}
	if cf == "" {
		cf = "default"
	}

	results, err := s.tikv.ScanPrefix(pd, cf, prefix, limit, parseType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":   len(results),
		"entries": results,
	})
}
