package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

const idleTimeout = 600 * time.Second // 10 minutes

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (s *Server) handleTerminal(w http.ResponseWriter, r *http.Request) {
	tokenStr := r.URL.Query().Get("token")
	username, ok := s.auth.VerifyToken(tokenStr)
	if !ok {
		slog.Warn("Terminal auth failed", "remote", r.RemoteAddr)
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("WebSocket upgrade failed", "error", err, "user", username, "remote", r.RemoteAddr)
		return
	}
	defer conn.Close()

	slog.Info("WebSocket terminal opened", "user", username, "remote", r.RemoteAddr)

	// Start bash with PTY
	cmd := exec.Command("/bin/bash", "--login")
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		slog.Error("Failed to start PTY", "error", err, "user", username)
		conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "Failed to start terminal"))
		return
	}
	defer ptmx.Close()

	slog.Info("PTY started", "user", username, "pid", cmd.Process.Pid)

	// Set default terminal size
	setTerminalSize(ptmx, 24, 80)

	var (
		cleanedUp    bool
		cleanupMu    sync.Mutex
		lastActivity = time.Now()
		activityMu   sync.Mutex
	)

	touchActivity := func() {
		activityMu.Lock()
		lastActivity = time.Now()
		activityMu.Unlock()
	}

	cleanup := func() {
		cleanupMu.Lock()
		defer cleanupMu.Unlock()
		if cleanedUp {
			slog.Debug("Cleanup already done", "user", username)
			return
		}
		cleanedUp = true

		slog.Info("Cleaning up terminal", "user", username, "pid", cmd.Process.Pid)

		ptmx.Close()

		if cmd.Process != nil {
			cmd.Process.Signal(syscall.SIGTERM)
			done := make(chan struct{})
			go func() {
				err := cmd.Wait()
				slog.Info("Process exited", "user", username, "pid", cmd.Process.Pid, "wait_err", err)
				close(done)
			}()
			select {
			case <-done:
				slog.Debug("Process exited gracefully", "user", username, "pid", cmd.Process.Pid)
			case <-time.After(1 * time.Second):
				slog.Warn("Force killing process", "user", username, "pid", cmd.Process.Pid)
				cmd.Process.Signal(syscall.SIGKILL)
				cmd.Wait()
			}
		}
	}
	defer cleanup()

	// Read from PTY, send to WebSocket
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				slog.Info("PTY read ended", "user", username, "error", err)
				// PTY closed (bash exited) - close WebSocket to notify frontend
				conn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Process exited"))
				return
			}
			if n > 0 {
				touchActivity()
				if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
					slog.Info("WebSocket write failed", "user", username, "error", err)
					return
				}
			}
		}
	}()

	// Idle watchdog
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				activityMu.Lock()
				elapsed := time.Since(lastActivity)
				activityMu.Unlock()

				if elapsed >= idleTimeout {
					slog.Info("Terminal idle timeout", "user", username, "timeout", idleTimeout)
					conn.WriteMessage(websocket.TextMessage,
						[]byte("\r\n\x1b[33m[Session timed out due to inactivity]\x1b[0m\r\n"))
					conn.WriteMessage(websocket.CloseMessage,
						websocket.FormatCloseMessage(4002, "Idle timeout"))
					return
				}
			}
		}
	}()

	// Read from WebSocket, write to PTY
	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			slog.Info("WebSocket read ended", "user", username, "error", err, "msgType", msgType)
			break
		}

		touchActivity()

		switch msgType {
		case websocket.TextMessage:
			text := string(msg)
			// Handle resize command: \x1bresize:rows:cols
			if strings.HasPrefix(text, "\x1bresize:") {
				parts := strings.Split(text, ":")
				if len(parts) == 3 {
					var rows, cols int
					if _, err := fmt.Sscanf(parts[1]+":"+parts[2], "%d:%d", &rows, &cols); err == nil {
						setTerminalSize(ptmx, rows, cols)
						slog.Debug("Terminal resized", "user", username, "rows", rows, "cols", cols)
					}
				}
				continue
			}
			if _, err := ptmx.Write(msg); err != nil {
				slog.Info("PTY write failed", "user", username, "error", err)
				goto exit
			}
		case websocket.BinaryMessage:
			if _, err := ptmx.Write(msg); err != nil {
				slog.Info("PTY write failed", "user", username, "error", err)
				goto exit
			}
		}
	}

exit:
	slog.Info("WebSocket terminal closed", "user", username, "pid", cmd.Process.Pid)
}

func setTerminalSize(f *os.File, rows, cols int) {
	ws := struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}{
		Row: uint16(rows),
		Col: uint16(cols),
	}
	syscall.Syscall(
		syscall.SYS_IOCTL,
		f.Fd(),
		uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(&ws)),
	)
}
