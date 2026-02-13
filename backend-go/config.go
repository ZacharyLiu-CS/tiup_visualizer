package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// AppConfig holds all application configuration.
type AppConfig struct {
	// From environment or defaults
	AppName     string
	Debug       bool
	APIPrefix   string
	RootPath    string
	CORSOrigins []string
	ListenAddr  string

	// From config.yaml
	Auth    AuthConfig    `yaml:"auth"`
	Logging LoggingConfig `yaml:"logging"`
}

type AuthConfig struct {
	Username         string `yaml:"username"`
	Password         string `yaml:"password"`
	SecretKey        string `yaml:"secret_key"`
	TokenExpireHours int    `yaml:"token_expire_hours"`
}

type LoggingConfig struct {
	LogDir         string `yaml:"log_dir"`
	LogLevel       string `yaml:"log_level"`
	MaxFileSizeMB  int    `yaml:"max_file_size_mb"`
	BackupCount    int    `yaml:"backup_count"`
}

func DefaultConfig() *AppConfig {
	return &AppConfig{
		AppName:     "TiUP Visualizer",
		Debug:       false,
		APIPrefix:   "/api/v1",
		RootPath:    "",
		CORSOrigins: []string{"http://localhost:5173", "http://localhost:3000"},
		ListenAddr:  ":8000",
		Auth: AuthConfig{
			Username:         "admin",
			Password:         "easygraph",
			SecretKey:        "tiup-visualizer-secret-key-change-me-in-production",
			TokenExpireHours: 24,
		},
		Logging: LoggingConfig{
			LogDir:        "./logs",
			LogLevel:      "INFO",
			MaxFileSizeMB: 10,
			BackupCount:   5,
		},
	}
}

func LoadConfig(execDir string) *AppConfig {
	cfg := DefaultConfig()

	// Load environment variable overrides
	if v := os.Getenv("APP_NAME"); v != "" {
		cfg.AppName = v
	}
	if v := os.Getenv("DEBUG"); strings.EqualFold(v, "true") {
		cfg.Debug = true
	}
	if v := os.Getenv("API_PREFIX"); v != "" {
		cfg.APIPrefix = v
	}
	if v := os.Getenv("ROOT_PATH"); v != "" {
		cfg.RootPath = v
	}
	if v := os.Getenv("LISTEN_ADDR"); v != "" {
		cfg.ListenAddr = v
	}
	if v := os.Getenv("CORS_ORIGINS"); v != "" {
		cfg.CORSOrigins = strings.Split(v, ",")
	}

	// Load config.yaml
	yamlPath := findConfigFile(execDir)
	if yamlPath != "" {
		data, err := os.ReadFile(yamlPath)
		if err == nil {
			var yamlCfg struct {
				Auth    AuthConfig    `yaml:"auth"`
				Logging LoggingConfig `yaml:"logging"`
			}
			if err := yaml.Unmarshal(data, &yamlCfg); err == nil {
				if yamlCfg.Auth.Username != "" {
					cfg.Auth = yamlCfg.Auth
				}
				if yamlCfg.Logging.LogDir != "" {
					cfg.Logging = yamlCfg.Logging
				}
			} else {
				slog.Warn("Failed to parse config.yaml", "error", err)
			}
		}
		slog.Info("Loaded config", "path", yamlPath)
	}

	// Apply defaults for zero values from yaml
	if cfg.Auth.TokenExpireHours <= 0 {
		cfg.Auth.TokenExpireHours = 24
	}
	if cfg.Logging.MaxFileSizeMB <= 0 {
		cfg.Logging.MaxFileSizeMB = 10
	}
	if cfg.Logging.BackupCount <= 0 {
		cfg.Logging.BackupCount = 5
	}
	if cfg.Logging.LogLevel == "" {
		cfg.Logging.LogLevel = "INFO"
	}
	if cfg.Logging.LogDir == "" {
		cfg.Logging.LogDir = "./logs"
	}

	return cfg
}

func findConfigFile(execDir string) string {
	candidates := []string{
		os.Getenv("TIUP_VISUALIZER_CONFIG"),
		filepath.Join(execDir, "config.yaml"),
		"/etc/tiup-visualizer/config.yaml",
	}
	for _, p := range candidates {
		if p == "" {
			continue
		}
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}
	return ""
}

// ResolveLogDir returns the absolute path to the log directory.
func (c *AppConfig) ResolveLogDir(execDir string) string {
	d := c.Logging.LogDir
	if !filepath.IsAbs(d) {
		d = filepath.Join(execDir, d)
	}
	return d
}

func (c *AppConfig) ParseLogLevel() slog.Level {
	switch strings.ToUpper(c.Logging.LogLevel) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR", "CRITICAL":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func (c *AppConfig) String() string {
	return fmt.Sprintf("AppConfig{AppName:%s, Debug:%v, APIPrefix:%s, RootPath:%s, ListenAddr:%s, Auth.Username:%s}",
		c.AppName, c.Debug, c.APIPrefix, c.RootPath, c.ListenAddr, c.Auth.Username)
}
