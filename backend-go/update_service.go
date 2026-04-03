package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	mirrorBaseURL  = "https://mirrors.tencent.com/repository/generic/easygraph-tiup-visualizer/"
	versionPattern = `tiup-visualizer-(\d{8}_\d{6})\.tar\.gz`
)

// UpdateService handles version checking and self-update.
type UpdateService struct {
	execDir string
}

func NewUpdateService(execDir string) *UpdateService {
	return &UpdateService{execDir: execDir}
}

// LatestRelease holds info about the latest available release.
type LatestRelease struct {
	Version  string `json:"version"`
	Filename string `json:"filename"`
	URL      string `json:"url"`
}

// CheckResult is returned by the check endpoint.
type CheckResult struct {
	CurrentVersion string         `json:"current_version"`
	LatestRelease  *LatestRelease `json:"latest_release"`
	NeedUpdate     bool           `json:"need_update"`
}

// FetchLatestRelease queries the mirror index page and returns the latest package.
func (u *UpdateService) FetchLatestRelease() (*LatestRelease, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(mirrorBaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release list: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read release list: %w", err)
	}

	re := regexp.MustCompile(versionPattern)
	matches := re.FindAllStringSubmatch(string(body), -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no release packages found at %s", mirrorBaseURL)
	}

	// Collect unique versions and sort descending (newest first)
	seen := map[string]bool{}
	var versions []string
	for _, m := range matches {
		if !seen[m[1]] {
			seen[m[1]] = true
			versions = append(versions, m[1])
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(versions)))

	latest := versions[0]
	filename := fmt.Sprintf("tiup-visualizer-%s.tar.gz", latest)
	return &LatestRelease{
		Version:  latest,
		Filename: filename,
		URL:      mirrorBaseURL + filename,
	}, nil
}

// DownloadAndApply downloads the latest package to /tmp, extracts it, and runs deploy-nginx.sh.
func (u *UpdateService) DownloadAndApply(release *LatestRelease) error {
	tmpDir := filepath.Join("/tmp", "tiup-visualizer-update")
	os.RemoveAll(tmpDir)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}

	archivePath := filepath.Join(tmpDir, release.Filename)

	// Download
	slog.Info("Downloading update", "url", release.URL, "dest", archivePath)
	if err := downloadFile(release.URL, archivePath); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Extract
	slog.Info("Extracting archive", "path", archivePath)
	out, err := ExecuteCommand(
		fmt.Sprintf("tar -xzf %s -C %s", archivePath, tmpDir),
		5*time.Minute,
	)
	if err != nil {
		return fmt.Errorf("extract failed: %w\n%s", err, out)
	}

	// Find extracted directory (tiup-visualizer/)
	extractedDir := filepath.Join(tmpDir, "tiup-visualizer")
	if _, err := os.Stat(extractedDir); err != nil {
		return fmt.Errorf("extracted directory not found at %s", extractedDir)
	}

	// Copy version file into extracted dir so deploy can reference it
	versionSrc := filepath.Join(u.execDir, "version")
	if data, err := os.ReadFile(versionSrc); err == nil {
		_ = os.WriteFile(filepath.Join(extractedDir, "version"), data, 0644)
	}

	// Run deploy-nginx.sh (non-interactive, preserve existing flags)
	deployScript := filepath.Join(extractedDir, "deploy-nginx.sh")
	if _, err := os.Stat(deployScript); err != nil {
		return fmt.Errorf("deploy-nginx.sh not found in package")
	}
	if err := os.Chmod(deployScript, 0755); err != nil {
		return fmt.Errorf("chmod deploy-nginx.sh failed: %w", err)
	}

	// Read current config to preserve port/prefix
	deployArgs := u.buildDeployArgs()
	cmd := fmt.Sprintf("cd %s && bash deploy-nginx.sh %s", extractedDir, deployArgs)
	slog.Info("Running deploy script", "cmd", cmd)
	out, err = ExecuteCommand(cmd, 5*time.Minute)
	if err != nil {
		return fmt.Errorf("deploy failed: %w\n%s", err, out)
	}
	slog.Info("Deploy completed", "output", out)
	return nil
}

// buildDeployArgs reads config to pass the same port/prefix to the new deploy.
func (u *UpdateService) buildDeployArgs() string {
	args := []string{}
	// Try to read running config
	cfgPath := filepath.Join(u.execDir, "config.yaml")
	if data, err := os.ReadFile(cfgPath); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "root_path:") {
				val := strings.TrimSpace(strings.TrimPrefix(line, "root_path:"))
				val = strings.Trim(val, `"'`)
				if val != "" {
					args = append(args, "--prefix", val)
				}
			}
			if strings.HasPrefix(line, "listen_addr:") {
				val := strings.TrimSpace(strings.TrimPrefix(line, "listen_addr:"))
				val = strings.Trim(val, `"'`)
				// Extract port from "host:port"
				parts := strings.Split(val, ":")
				if len(parts) == 2 && parts[1] != "" {
					args = append(args, "--port", parts[1])
				}
			}
		}
	}
	return strings.Join(args, " ")
}

func downloadFile(url, dest string) error {
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}
