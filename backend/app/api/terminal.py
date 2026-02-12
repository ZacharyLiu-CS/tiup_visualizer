import asyncio
import fcntl
import logging
import os
import pty
import select
import signal
import struct
import termios
import time

from fastapi import APIRouter, WebSocket, WebSocketDisconnect, Query
from app.core.auth import verify_ws_token

logger = logging.getLogger("tiup_visualizer")

router = APIRouter()

# Idle timeout in seconds: if no data flows in either direction for this long,
# the server closes the PTY and WebSocket automatically.
IDLE_TIMEOUT = 600  # 10 minutes


@router.websocket("/ws/terminal")
async def terminal_websocket(websocket: WebSocket, token: str = Query(default=None)):
    """WebSocket endpoint that provides an interactive bash terminal via PTY."""
    # Verify authentication token
    username = verify_ws_token(token)
    if username is None:
        await websocket.close(code=4001, reason="Authentication required")
        return

    await websocket.accept()
    logger.info(f"WebSocket terminal opened by user: {username}")

    # Create a pseudo-terminal
    master_fd, slave_fd = pty.openpty()

    # Fork a child process running bash
    pid = os.fork()
    if pid == 0:
        # Child process
        os.close(master_fd)
        os.setsid()

        # Set the slave as the controlling terminal
        fcntl.ioctl(slave_fd, termios.TIOCSCTTY, 0)

        # Redirect stdio to the slave PTY
        os.dup2(slave_fd, 0)
        os.dup2(slave_fd, 1)
        os.dup2(slave_fd, 2)
        if slave_fd > 2:
            os.close(slave_fd)

        # Set environment variables for a nicer terminal experience
        env = os.environ.copy()
        env["TERM"] = "xterm-256color"
        env["COLORTERM"] = "truecolor"

        os.execvpe("/bin/bash", ["/bin/bash", "--login"], env)
    else:
        # Parent process
        os.close(slave_fd)

        # Set master_fd to non-blocking
        flag = fcntl.fcntl(master_fd, fcntl.F_GETFL)
        fcntl.fcntl(master_fd, fcntl.F_SETFL, flag | os.O_NONBLOCK)

        # Set default terminal size
        _set_terminal_size(master_fd, 24, 80)

        # Track whether cleanup has been done to avoid double-cleanup
        cleaned_up = False
        # Shared mutable timestamp for idle detection
        last_activity = time.monotonic()

        def touch_activity():
            nonlocal last_activity
            last_activity = time.monotonic()

        def cleanup():
            nonlocal cleaned_up
            if cleaned_up:
                return
            cleaned_up = True

            # Close master fd
            try:
                os.close(master_fd)
            except OSError:
                pass

            # Terminate child process gracefully, then force-kill if needed
            try:
                os.kill(pid, signal.SIGTERM)
            except OSError:
                return
            # Non-blocking wait with timeout — avoid hanging forever
            for _ in range(10):
                try:
                    result = os.waitpid(pid, os.WNOHANG)
                    if result[0] != 0:
                        return
                except (OSError, ChildProcessError):
                    return
                time.sleep(0.1)
            # Force kill if SIGTERM didn't work within 1 second
            try:
                os.kill(pid, signal.SIGKILL)
                os.waitpid(pid, 0)
            except (OSError, ChildProcessError):
                pass

        async def read_from_pty():
            """Read output from PTY and send to WebSocket."""
            try:
                while True:
                    await asyncio.sleep(0.01)
                    if cleaned_up:
                        break
                    try:
                        r, _, _ = select.select([master_fd], [], [], 0)
                        if r:
                            data = os.read(master_fd, 4096)
                            if not data:
                                # PTY closed (child exited)
                                break
                            touch_activity()
                            await websocket.send_bytes(data)
                    except OSError:
                        break
            except (WebSocketDisconnect, Exception):
                pass

        async def idle_watchdog():
            """Periodically check for idle timeout and force-close if exceeded."""
            try:
                while not cleaned_up:
                    await asyncio.sleep(30)  # check every 30s
                    if cleaned_up:
                        break
                    elapsed = time.monotonic() - last_activity
                    if elapsed >= IDLE_TIMEOUT:
                        logger.info(
                            f"Terminal idle timeout ({IDLE_TIMEOUT}s) for user: {username}, closing"
                        )
                        # Notify client before closing
                        try:
                            await websocket.send_text(
                                "\r\n\x1b[33m[Session timed out due to inactivity]\x1b[0m\r\n"
                            )
                        except Exception:
                            pass
                        # Close websocket to trigger the finally cleanup
                        try:
                            await websocket.close(code=4002, reason="Idle timeout")
                        except Exception:
                            pass
                        break
            except (asyncio.CancelledError, Exception):
                pass

        read_task = asyncio.ensure_future(read_from_pty())
        idle_task = asyncio.ensure_future(idle_watchdog())

        try:
            while True:
                message = await websocket.receive()
                if message.get("type") == "websocket.disconnect":
                    break

                touch_activity()

                if "text" in message:
                    text = message["text"]
                    # Handle resize command
                    if text.startswith("\x1bresize:"):
                        parts = text.split(":")
                        if len(parts) == 3:
                            try:
                                rows = int(parts[1])
                                cols = int(parts[2])
                                _set_terminal_size(master_fd, rows, cols)
                            except (ValueError, OSError):
                                pass
                        continue
                    try:
                        os.write(master_fd, text.encode("utf-8"))
                    except OSError:
                        break
                elif "bytes" in message:
                    try:
                        os.write(master_fd, message["bytes"])
                    except OSError:
                        break
        except WebSocketDisconnect:
            pass
        except Exception:
            pass
        finally:
            read_task.cancel()
            idle_task.cancel()
            try:
                await read_task
            except (asyncio.CancelledError, Exception):
                pass
            try:
                await idle_task
            except (asyncio.CancelledError, Exception):
                pass
            cleanup()
            logger.info(f"WebSocket terminal closed for user: {username}")
            # Close websocket if still open
            try:
                await websocket.close()
            except Exception:
                pass


def _set_terminal_size(fd: int, rows: int, cols: int):
    """Set the terminal size of a PTY."""
    size = struct.pack("HHHH", rows, cols, 0, 0)
    fcntl.ioctl(fd, termios.TIOCSWINSZ, size)
