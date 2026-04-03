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
	slog.Info("[update] Fetching release list", "url", mirrorBaseURL)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(mirrorBaseURL)
	if err != nil {
		slog.Error("[update] Failed to fetch release list", "error", err)
		return nil, fmt.Errorf("failed to fetch release list: %w", err)
	}
	defer resp.Body.Close()
	slog.Info("[update] Release list response", "status", resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("[update] Failed to read release list body", "error", err)
		return nil, fmt.Errorf("failed to read release list: %w", err)
	}

	re := regexp.MustCompile(versionPattern)
	matches := re.FindAllStringSubmatch(string(body), -1)
	if len(matches) == 0 {
		slog.Error("[update] No release packages found in response", "body_preview", truncate(string(body), 200))
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
	slog.Info("[update] Available versions", "count", len(versions), "versions", strings.Join(versions, ", "))

	latest := versions[0]
	filename := fmt.Sprintf("tiup-visualizer-%s.tar.gz", latest)
	slog.Info("[update] Latest release identified", "version", latest, "filename", filename)
	return &LatestRelease{
		Version:  latest,
		Filename: filename,
		URL:      mirrorBaseURL + filename,
	}, nil
}

// DownloadAndApply downloads the latest package to /tmp, extracts it, and runs deploy-nginx.sh.
func (u *UpdateService) DownloadAndApply(release *LatestRelease) error {
	slog.Info("[update] ===== Update process started =====", "target_version", release.Version)

	// Step 1: prepare temp dir
	tmpDir := filepath.Join("/tmp", "tiup-visualizer-update")
	slog.Info("[update] Step 1/5: Preparing temp directory", "path", tmpDir)
	if err := os.RemoveAll(tmpDir); err != nil {
		slog.Warn("[update] Failed to remove old temp dir (non-fatal)", "error", err)
	}
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		slog.Error("[update] Failed to create temp directory", "path", tmpDir, "error", err)
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	slog.Info("[update] Temp directory ready", "path", tmpDir)

	// Step 2: download
	archivePath := filepath.Join(tmpDir, release.Filename)
	slog.Info("[update] Step 2/5: Downloading package", "url", release.URL, "dest", archivePath)
	startTime := time.Now()
	if err := downloadFile(release.URL, archivePath); err != nil {
		slog.Error("[update] Download failed", "url", release.URL, "error", err)
		return fmt.Errorf("download failed: %w", err)
	}
	fi, _ := os.Stat(archivePath)
	var fileSize int64
	if fi != nil {
		fileSize = fi.Size()
	}
	slog.Info("[update] Download completed",
		"dest", archivePath,
		"size_bytes", fileSize,
		"elapsed", time.Since(startTime).Round(time.Millisecond),
	)

	// Step 3: extract
	extractedDir := filepath.Join(tmpDir, strings.TrimSuffix(release.Filename, ".tar.gz"))
	if err := os.MkdirAll(extractedDir, 0755); err != nil {
		slog.Error("[update] Failed to create extract directory", "path", extractedDir, "error", err)
		return fmt.Errorf("failed to create extract dir: %w", err)
	}
	slog.Info("[update] Step 3/5: Extracting archive", "archive", archivePath, "dest", extractedDir)
	extractCmd := fmt.Sprintf("tar -xzf %s -C %s --strip-components=1", archivePath, extractedDir)
	slog.Info("[update] Running command", "cmd", extractCmd)
	out, err := ExecuteCommand(extractCmd, 5*time.Minute)
	if out != "" {
		slog.Info("[update] tar output", "output", out)
	}
	if err != nil {
		slog.Error("[update] Extract failed", "error", err, "output", out)
		return fmt.Errorf("extract failed: %w\n%s", err, out)
	}
	slog.Info("[update] Archive extracted successfully", "dest", extractedDir)

	// Verify extracted directory
	if _, err := os.Stat(extractedDir); err != nil {
		slog.Error("[update] Extracted directory not found", "expected_path", extractedDir, "error", err)
		return fmt.Errorf("extracted directory not found at %s", extractedDir)
	}
	slog.Info("[update] Extracted directory verified", "path", extractedDir)

	// Verify version in the new package
	newVersionPath := filepath.Join(extractedDir, "version")
	if data, err := os.ReadFile(newVersionPath); err == nil {
		slog.Info("[update] Step 4/5: New package version", "version", strings.TrimSpace(string(data)))
	} else {
		// Package has no version file — write the target version so deploy installs it
		slog.Warn("[update] Step 4/5: version file not found in package, writing target version")
		_ = os.WriteFile(newVersionPath, []byte(release.Version+"\n"), 0644)
	}

	// Verify deploy script exists
	deployScript := filepath.Join(extractedDir, "deploy-nginx.sh")
	if _, err := os.Stat(deployScript); err != nil {
		slog.Error("[update] deploy-nginx.sh not found in package", "expected_path", deployScript)
		return fmt.Errorf("deploy-nginx.sh not found in package")
	}
	if err := os.Chmod(deployScript, 0755); err != nil {
		slog.Error("[update] chmod deploy-nginx.sh failed", "error", err)
		return fmt.Errorf("chmod deploy-nginx.sh failed: %w", err)
	}

	// Step 5: launch deploy via detached runner script
	// We MUST NOT call deploy-nginx.sh directly in this process because it will
	// restart the systemd service (killing us mid-execution).
	// Instead, write a standalone runner shell script and launch it with setsid/nohup
	// so it runs in a completely separate process group and survives our exit.
	deployArgs := u.buildDeployArgs()
	logFile := filepath.Join(u.execDir, "logs", "tiup-visualizer.log")
	runnerScript := filepath.Join(tmpDir, "run-deploy.sh")
	runnerContent := fmt.Sprintf(`#!/bin/bash
# Auto-generated update runner — do not edit
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:$PATH"
DEPLOY_DIR=%s
DEPLOY_ARGS=%q
LOG_FILE=%s
TMPDIR=%s

log() { echo "[$(date '+%%Y-%%m-%%dT%%H:%%M:%%S')] [update-runner] $*" | tee -a "$LOG_FILE"; }

log "===== Runner started (pid=$$) ====="
log "Deploy dir: $DEPLOY_DIR"
log "Deploy args: $DEPLOY_ARGS"

cd "$DEPLOY_DIR" || { log "ERROR: cannot cd to $DEPLOY_DIR"; exit 1; }

log "Running deploy-nginx.sh ..."
bash deploy-nginx.sh $DEPLOY_ARGS >> "$LOG_FILE" 2>&1
EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
    log "===== Deploy completed successfully ====="
else
    log "ERROR: deploy-nginx.sh exited with code $EXIT_CODE"
fi

log "Cleaning up temp directory: $TMPDIR"
rm -rf "$TMPDIR"
log "Done."
exit $EXIT_CODE
`, extractedDir, deployArgs, logFile, tmpDir)

	if err := os.WriteFile(runnerScript, []byte(runnerContent), 0755); err != nil {
		slog.Error("[update] Failed to write runner script", "path", runnerScript, "error", err)
		return fmt.Errorf("failed to write runner script: %w", err)
	}
	slog.Info("[update] Step 5/5: Launching detached deploy runner",
		"runner", runnerScript,
		"deploy_args", deployArgs,
		"log", logFile,
	)

	// systemd stop kills the entire service cgroup, so we need the runner in its own cgroup.
	// Strategy (in order of preference):
	//   1. systemd-run with sudo (separate transient unit, separate cgroup)
	//   2. at(1) via atd daemon (completely independent scheduler)
	//   3. double-fork via sh -c so child is reparented to init/PID1
	var launchCmd string
	switch {
	case cmdExists("systemd-run"):
		launchCmd = fmt.Sprintf(
			"sudo systemd-run --no-ask-password --unit=tiup-visualizer-update --description='TiUP Visualizer self-update' bash %s",
			runnerScript,
		)
	case cmdExists("at"):
		launchCmd = fmt.Sprintf("echo 'bash %s' | at now", runnerScript)
	default:
		// Double-fork: sh spawns a subshell that spawns the runner and exits,
		// leaving the runner reparented to init (PID 1), outside our cgroup.
		launchCmd = fmt.Sprintf(
			`sh -c 'bash %s </dev/null >/dev/null 2>&1 & disown $!'`,
			runnerScript,
		)
	}
	slog.Info("[update] Launch command", "cmd", launchCmd)
	if out, err := ExecuteCommand(launchCmd, 10*time.Second); err != nil {
		slog.Error("[update] Failed to launch runner", "error", err, "output", out)
		return fmt.Errorf("failed to launch deploy runner: %w", err)
	}

	slog.Info("[update] Deploy runner launched — service will restart shortly",
		"version", release.Version,
	)
	// tmpDir cleanup is handled by the runner script itself after deploy completes
	return nil
}

// cmdExists reports whether a command is available in PATH.
func cmdExists(cmd string) bool {
	out, err := ExecuteCommand("command -v "+cmd, 3*time.Second)
	return err == nil && len(out) > 0
}

// cleanupTmpDir removes the entire update temp directory and logs the result.
func cleanupTmpDir(tmpDir string) {
	slog.Info("[update] Cleaning up temp directory", "path", tmpDir)
	if err := os.RemoveAll(tmpDir); err != nil {
		slog.Warn("[update] Failed to remove temp directory", "path", tmpDir, "error", err)
	} else {
		slog.Info("[update] Temp directory removed", "path", tmpDir)
	}
}

// buildDeployArgs reads config to pass the same port/prefix/user to the new deploy.
func (u *UpdateService) buildDeployArgs() string {
	args := []string{}
	cfgPath := filepath.Join(u.execDir, "config.yaml")
	slog.Info("[update] Reading config for deploy args", "path", cfgPath)
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		slog.Warn("[update] Could not read config.yaml, using deploy defaults", "error", err)
	} else {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "root_path:") {
				val := strings.TrimSpace(strings.TrimPrefix(line, "root_path:"))
				val = strings.Trim(val, `"'`)
				if val != "" {
					args = append(args, "--prefix", val)
					slog.Info("[update] Deploy arg: prefix", "value", val)
				}
			}
			if strings.HasPrefix(line, "listen_addr:") {
				val := strings.TrimSpace(strings.TrimPrefix(line, "listen_addr:"))
				val = strings.Trim(val, `"'`)
				parts := strings.Split(val, ":")
				if len(parts) == 2 && parts[1] != "" {
					args = append(args, "--port", parts[1])
					slog.Info("[update] Deploy arg: port", "value", parts[1])
				}
			}
		}
	}

	// Detect service user from systemd unit or current process owner
	serviceUser := u.detectServiceUser()
	if serviceUser != "" && serviceUser != "root" {
		args = append(args, "--user", serviceUser)
		slog.Info("[update] Deploy arg: user", "value", serviceUser)
	}

	return strings.Join(args, " ")
}

// detectServiceUser returns the User= configured in the systemd unit, or the current process owner.
func (u *UpdateService) detectServiceUser() string {
	// Try reading from systemd unit
	out, err := ExecuteCommand("systemctl show tiup-visualizer --property=User --value 2>/dev/null", 5*time.Second)
	if err == nil {
		user := strings.TrimSpace(out)
		if user != "" && user != "(null)" {
			return user
		}
	}
	// Fallback: current process owner
	return os.Getenv("USER")
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

// truncate truncates a string to max n chars for log preview.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

