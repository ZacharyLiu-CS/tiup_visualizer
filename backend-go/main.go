package main

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

func main() {
	// Determine executable directory for relative paths
	execPath, err := os.Executable()
	if err != nil {
		execPath = "."
	}
	execDir := filepath.Dir(execPath)

	// If running with `go run`, use CWD instead
	if cwd, err := os.Getwd(); err == nil {
		execDir = cwd
	}

	// Load configuration
	cfg := LoadConfig(execDir)

	// Setup logging
	setupLogging(cfg, execDir)

	slog.Info("Starting TiUP Visualizer", "config", cfg.String())

	// Create server
	srv := NewServer(cfg, execDir)

	// Apply middleware
	var handler http.Handler = srv
	handler = CORSMiddleware(cfg.CORSOrigins, handler)
	handler = LoggingMiddleware(handler)

	slog.Info("Listening", "addr", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, handler); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}

func setupLogging(cfg *AppConfig, execDir string) {
	logDir := cfg.ResolveLogDir(execDir)
	os.MkdirAll(logDir, 0755)

	logFile := filepath.Join(logDir, "tiup-visualizer.log")

	lj := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    cfg.Logging.MaxFileSizeMB,
		MaxBackups: cfg.Logging.BackupCount,
		Compress:   false,
	}

	level := cfg.ParseLogLevel()

	multiWriter := io.MultiWriter(os.Stdout, lj)
	handler := slog.NewTextHandler(multiWriter, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))
}
