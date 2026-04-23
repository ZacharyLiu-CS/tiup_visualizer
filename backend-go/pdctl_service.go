package main

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// PDCtlService provides execution of `tiup ctl:<version> pd --pd <addr> ...` commands.
type PDCtlService struct{}

// NewPDCtlService creates a PDCtlService.
func NewPDCtlService() *PDCtlService {
	return &PDCtlService{}
}

// PDCtlExecRequest describes a pd-ctl execution request.
type PDCtlExecRequest struct {
	PDAddr      string `json:"pd_addr"`      // direct PD address (comma separated); optional if ClusterName is set
	ClusterName string `json:"cluster_name"` // cluster name to resolve PD addresses; optional if PDAddr is set
	TiUPVersion string `json:"tiup_version"` // e.g. v8.1.0
	Command     string `json:"command"`      // top-level pd-ctl command, e.g. "scheduler"
	SubCommand  string `json:"sub_command"`  // extra args, e.g. "show" or "region topread 5"
	Help        bool   `json:"help"`         // if true, append --help
}

// PDCtlExecResult is the structured output of a pd-ctl execution.
// Preamble holds the `Starting component ctl: ...` line emitted by tiup.
// Output holds the actual command output (pd-ctl's own stdout/stderr).
type PDCtlExecResult struct {
	Command    string `json:"command"`
	Preamble   string `json:"preamble"`
	Output     string `json:"output"`
	Raw        string `json:"raw"`
	ExitCode   int    `json:"exit_code"`
	DurationMs int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

// splitArgs does a simple whitespace split (no shell quoting). It keeps things
// predictable — users can type `scheduler show` or `region topread 5`.
func splitArgs(s string) []string {
	fields := strings.Fields(s)
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}

// ANSI CSI escape sequences (e.g. "\x1b[1m", "\x1b[0m", "\x1b[31;1m").
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// stripANSI removes ANSI escape sequences from a string.
func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// splitTiUPOutput separates the `Starting component ...` preamble line emitted
// by the tiup loader from the actual command output. The preamble is always on
// the first line of stdout when invoking `tiup ctl:<ver> pd ...`.
func splitTiUPOutput(combined string) (preamble, output string) {
	s := strings.TrimLeft(combined, "\r\n")
	if !strings.HasPrefix(s, "Starting component") {
		return "", combined
	}
	// Find end of first line.
	idx := strings.IndexByte(s, '\n')
	if idx < 0 {
		return strings.TrimRight(s, "\r\n"), ""
	}
	preamble = strings.TrimRight(s[:idx], "\r\n")
	rest := s[idx+1:]
	// Some tiup versions leave a lone space or leading whitespace on the next
	// line after the ANSI reset is stripped. Trim only leading whitespace/newlines,
	// preserve trailing content verbatim.
	rest = strings.TrimLeft(rest, " \t\r\n")
	return preamble, rest
}

// BuildCommand returns the full command string (for display) and the argv used for exec.
func (s *PDCtlService) BuildCommand(req PDCtlExecRequest) (display string, argv []string, err error) {
	tiupVersion := strings.TrimSpace(req.TiUPVersion)
	if tiupVersion == "" {
		tiupVersion = "v8.1.0"
	}
	pdAddr := strings.TrimSpace(req.PDAddr)
	if pdAddr == "" {
		return "", nil, fmt.Errorf("pd_addr is required")
	}

	argv = []string{
		"tiup",
		fmt.Sprintf("ctl:%s", tiupVersion),
		"pd",
		"--pd", pdAddr,
	}

	command := strings.TrimSpace(req.Command)
	if command != "" {
		argv = append(argv, command)
	}

	subArgs := splitArgs(req.SubCommand)
	argv = append(argv, subArgs...)

	if req.Help {
		argv = append(argv, "--help")
	}

	display = strings.Join(argv, " ")
	return display, argv, nil
}

// Execute runs the pd-ctl command and returns the cleaned output.
// Sequence: run → strip ANSI from combined stdout/stderr → split preamble from body.
func (s *PDCtlService) Execute(req PDCtlExecRequest) (*PDCtlExecResult, error) {
	display, argv, err := s.BuildCommand(req)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	start := time.Now()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	out, runErr := cmd.CombinedOutput()
	duration := time.Since(start)

	raw := string(out)
	cleaned := stripANSI(raw)
	preamble, body := splitTiUPOutput(cleaned)

	result := &PDCtlExecResult{
		Command:    display,
		Preamble:   preamble,
		Output:     body,
		Raw:        raw,
		ExitCode:   0,
		DurationMs: duration.Milliseconds(),
	}

	if ctx.Err() == context.DeadlineExceeded {
		result.Error = "command execution timeout (60s)"
		result.ExitCode = -1
		return result, nil
	}
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			// Do not treat non-zero exit as a fatal error; surface output for user.
		} else {
			result.Error = runErr.Error()
			result.ExitCode = -1
		}
	}
	return result, nil
}
