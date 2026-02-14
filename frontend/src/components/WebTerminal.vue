<template>
  <!-- Float (overlay) mode -->
  <teleport to="body">
    <transition name="fade">
      <div v-if="mode === 'float' && visible" class="terminal-overlay" @click.self="handleClose">
        <div class="terminal-window" :class="{ maximized: isMaximized }">
          <div class="terminal-titlebar">
            <div class="terminal-title">
              <span class="terminal-icon">&#9002;</span>
              Web Terminal
            </div>
            <div class="terminal-controls">
              <button class="ctrl-btn maximize-btn" @click="toggleMaximize" :title="isMaximized ? 'Restore' : 'Maximize'">
                <span v-if="isMaximized">&#9645;</span>
                <span v-else>&#9633;</span>
              </button>
              <button class="ctrl-btn close-btn" @click="handleClose" title="Close">&times;</button>
            </div>
          </div>
          <div class="terminal-body" ref="floatBody"></div>
          <div class="terminal-statusbar">
            <span class="status-dot" :class="{ connected: wsConnected }"></span>
            <span>{{ wsConnected ? 'Connected' : 'Disconnected' }}</span>
          </div>
        </div>
      </div>
    </transition>
  </teleport>

  <!-- Panel (slide) modes -->
  <teleport to="body">
    <transition :name="slideTransitionName">
      <div v-if="mode !== 'float' && visible" class="terminal-panel" :class="panelClass">
        <div class="terminal-titlebar">
          <div class="terminal-title">
            <span class="terminal-icon">&#9002;</span>
            Web Terminal
          </div>
          <div class="terminal-controls">
            <button class="ctrl-btn close-btn" @click="handleClose" title="Close">&times;</button>
          </div>
        </div>
        <div class="terminal-body" ref="panelBody"></div>
        <div class="terminal-statusbar">
          <span class="status-dot" :class="{ connected: wsConnected }"></span>
          <span>{{ wsConnected ? 'Connected' : 'Disconnected' }}</span>
        </div>
      </div>
    </transition>
  </teleport>

  <!-- Hidden persistent container that keeps the xterm DOM alive -->
  <div ref="terminalHost" class="terminal-host-hidden"></div>
</template>

<script>
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'

export default {
  name: 'WebTerminal',
  props: {
    visible: {
      type: Boolean,
      default: false
    },
    mode: {
      type: String,
      default: 'top',
      validator: (v) => ['top', 'right', 'bottom', 'left', 'float'].includes(v)
    }
  },
  emits: ['close'],
  data() {
    return {
      terminal: null,
      fitAddon: null,
      ws: null,
      wsConnected: false,
      isMaximized: false,
      resizeObserver: null,
      _initialized: false,
      _connectionDead: false  // Flag to indicate terminal needs full rebuild
    }
  },
  computed: {
    panelClass() {
      return `panel-${this.mode}`
    },
    slideTransitionName() {
      const map = {
        top: 'slide-top',
        right: 'slide-right',
        bottom: 'slide-bottom',
        left: 'slide-left'
      }
      return map[this.mode] || 'slide-top'
    }
  },
  watch: {
    visible(val) {
      if (val) {
        this.ensureInitialized()
        this.$nextTick(() => this.attachTerminal())
      } else {
        // Just detach — move xterm DOM back to the hidden host.
        // Do NOT destroy; we reuse on next open.
        this.$nextTick(() => this.detachTerminal())
      }
    },
    mode() {
      if (this.visible) {
        // Mode changed while open: move xterm DOM to the new container
        this.$nextTick(() => this.attachTerminal())
      }
    }
  },
  methods: {
    handleClose() {
      this.$emit('close')
    },

    /**
     * One-time initialization: create Terminal + WebSocket.
     * Called on first open only; subsequent opens skip this.
     */
    ensureInitialized() {
      // If connection is dead (bash exited), rebuild everything
      if (this._connectionDead) {
        this.rebuildTerminal()
        return
      }

      if (this._initialized) {
        // Already initialized - check if we need to reconnect WebSocket
        this.ensureWebSocketConnected()
        return
      }
      this._initialized = true

      this.createTerminal()
    },

    /**
     * Create a new Terminal instance (called once during init or rebuild).
     */
    createTerminal() {
      this.terminal = new Terminal({
        cursorBlink: true,
        fontSize: 14,
        fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', Menlo, Monaco, 'Courier New', monospace",
        theme: {
          background: '#1e1e2e',
          foreground: '#cdd6f4',
          cursor: '#f5e0dc',
          cursorAccent: '#1e1e2e',
          selectionBackground: '#585b70',
          black: '#45475a',
          red: '#f38ba8',
          green: '#a6e3a1',
          yellow: '#f9e2af',
          blue: '#89b4fa',
          magenta: '#f5c2e7',
          cyan: '#94e2d5',
          white: '#bac2de',
          brightBlack: '#585b70',
          brightRed: '#f38ba8',
          brightGreen: '#a6e3a1',
          brightYellow: '#f9e2af',
          brightBlue: '#89b4fa',
          brightMagenta: '#f5c2e7',
          brightCyan: '#94e2d5',
          brightWhite: '#a6adc8'
        },
        allowProposedApi: true
      })

      this.fitAddon = new FitAddon()
      this.terminal.loadAddon(this.fitAddon)
      this.terminal.loadAddon(new WebLinksAddon())

      // Open xterm into the hidden host first (keeps DOM alive)
      this.terminal.open(this.$refs.terminalHost)

      this.terminal.onData(data => {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
          this.ws.send(data)
        }
      })

      this.connectWebSocket()
    },

    /**
     * Rebuild the entire terminal (used when connection dies and user reopens).
     */
    rebuildTerminal() {
      // Clean up old resources
      if (this.resizeObserver) {
        this.resizeObserver.disconnect()
        this.resizeObserver = null
      }

      if (this.ws) {
        this.ws.onopen = null
        this.ws.onmessage = null
        this.ws.onclose = null
        this.ws.onerror = null
        if (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING) {
          this.ws.close()
        }
        this.ws = null
      }

      if (this.terminal) {
        this.terminal.dispose()
        this.terminal = null
      }

      this.fitAddon = null
      this.wsConnected = false
      this._connectionDead = false
      this._initialized = true

      // Recreate terminal
      this.createTerminal()

      // Re-attach to visible container if visible
      if (this.visible) {
        this.$nextTick(() => this.attachTerminal())
      }
    },

    /**
     * Ensure WebSocket is connected. Trigger rebuild if dead.
     */
    ensureWebSocketConnected() {
      // If connection is dead or ws is closed/closing, rebuild
      if (this._connectionDead || !this.ws || this.ws.readyState === WebSocket.CLOSED || this.ws.readyState === WebSocket.CLOSING) {
        this._connectionDead = true  // Ensure rebuild happens
        this.rebuildTerminal()
      }
    },

    /**
     * Move the xterm DOM element into the currently visible body container
     * and fit to the new size.
     */
    attachTerminal() {
      const target = this.mode === 'float' ? this.$refs.floatBody : this.$refs.panelBody
      if (!target || !this.terminal) return

      const xtermEl = this.$refs.terminalHost
      if (!xtermEl) return

      // Move the hidden host (which contains the xterm DOM) into the visible body
      target.appendChild(xtermEl)

      // Disconnect old observer if any
      if (this.resizeObserver) {
        this.resizeObserver.disconnect()
      }

      // Fit after the layout has settled
      this.$nextTick(() => {
        if (this.fitAddon) {
          try { this.fitAddon.fit() } catch (_) { /* ignore */ }
          this.sendResize()
        }
      })

      // Observe the target for future resizes
      this.resizeObserver = new ResizeObserver(() => {
        if (this.fitAddon && this.terminal && this.visible) {
          try { this.fitAddon.fit() } catch (_) { /* ignore */ }
          this.sendResize()
        }
      })
      this.resizeObserver.observe(target)
    },

    /**
     * Move xterm DOM back to be a direct child of body (hidden via CSS).
     * This keeps the DOM alive but invisible.
     */
    detachTerminal() {
      const xtermEl = this.$refs.terminalHost
      if (!xtermEl) return

      // If it's currently inside a floatBody/panelBody, move it back to body
      // so it stays alive but hidden. The CSS class .terminal-host-hidden hides it.
      if (xtermEl.parentNode && xtermEl.parentNode !== document.body) {
        document.body.appendChild(xtermEl)
      }

      if (this.resizeObserver) {
        this.resizeObserver.disconnect()
        this.resizeObserver = null
      }
    },

    connectWebSocket() {
      // Clean up old WebSocket if exists
      if (this.ws) {
        this.ws.onopen = null
        this.ws.onmessage = null
        this.ws.onclose = null
        this.ws.onerror = null
        if (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING) {
          this.ws.close()
        }
      }

      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const basePath = import.meta.env.BASE_URL.replace(/\/+$/, '')
      const token = localStorage.getItem('auth_token') || ''
      const wsUrl = `${protocol}//${window.location.host}${basePath}/ws/terminal?token=${encodeURIComponent(token)}`

      this.ws = new WebSocket(wsUrl)
      this.ws.binaryType = 'arraybuffer'

      this.ws.onopen = () => {
        this.wsConnected = true
        // Clear reconnecting message if present
        if (this.terminal) {
          this.terminal.write('\x1b[2K\r')  // Clear current line
        }
        this.sendResize()
      }

      this.ws.onmessage = (event) => {
        if (!this.terminal) return
        if (event.data instanceof ArrayBuffer) {
          this.terminal.write(new Uint8Array(event.data))
        } else {
          this.terminal.write(event.data)
        }
      }

      this.ws.onclose = () => {
        this.wsConnected = false
        // Mark terminal as dead - will be rebuilt on next open
        this._connectionDead = true
        // Show message to user
        if (this.terminal) {
          this.terminal.write('\r\n\x1b[31m[Session ended]\x1b[0m\r\n')
          this.terminal.write('\x1b[33m[Close and reopen terminal to start a new session]\x1b[0m\r\n')
        }
      }

      this.ws.onerror = () => {
        this.wsConnected = false
        if (this.terminal) {
          this.terminal.write('\r\n\x1b[31m[Connection error]\x1b[0m\r\n')
        }
      }
    },

    sendResize() {
      if (this.ws && this.ws.readyState === WebSocket.OPEN && this.terminal) {
        const rows = this.terminal.rows
        const cols = this.terminal.cols
        this.ws.send(`\x1bresize:${rows}:${cols}`)
      }
    },

    toggleMaximize() {
      this.isMaximized = !this.isMaximized
      this.$nextTick(() => {
        if (this.fitAddon && this.visible) {
          try { this.fitAddon.fit() } catch (_) { /* ignore */ }
          this.sendResize()
        }
      })
    },

    /**
     * Full cleanup — only called on page unload (beforeUnmount).
     */
    destroyTerminal() {
      if (this.resizeObserver) {
        this.resizeObserver.disconnect()
        this.resizeObserver = null
      }

      if (this.ws) {
        this.ws.onopen = null
        this.ws.onmessage = null
        this.ws.onclose = null
        this.ws.onerror = null
        this.ws.close()
        this.ws = null
      }

      if (this.terminal) {
        this.terminal.dispose()
        this.terminal = null
      }

      // Clean up the detached host element from body if it was moved there
      const xtermEl = this.$refs.terminalHost
      if (xtermEl && xtermEl.parentNode === document.body) {
        document.body.removeChild(xtermEl)
      }

      this.fitAddon = null
      this.wsConnected = false
      this.isMaximized = false
      this._initialized = false
      this._connectionDead = false
    }
  },
  mounted() {
    this._onBeforeUnload = () => this.destroyTerminal()
    window.addEventListener('beforeunload', this._onBeforeUnload)
  },
  beforeUnmount() {
    window.removeEventListener('beforeunload', this._onBeforeUnload)
    this.destroyTerminal()
  }
}
</script>

<style scoped>
/* Hidden host that keeps xterm DOM alive when terminal panel/float is closed */
.terminal-host-hidden {
  position: absolute;
  width: 0;
  height: 0;
  overflow: hidden;
  pointer-events: none;
}

/* When attached inside a visible body, show it properly */
.terminal-body .terminal-host-hidden {
  position: static;
  width: 100%;
  height: 100%;
  overflow: hidden;
  pointer-events: auto;
}

/* ===== Fade transition for float overlay ===== */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.25s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

/* ===== Float (overlay) mode ===== */
.terminal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  backdrop-filter: blur(4px);
}

.terminal-window {
  width: 80%;
  height: 70%;
  min-width: 600px;
  min-height: 400px;
  background: #1e1e2e;
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 25px 50px rgba(0, 0, 0, 0.5);
  border: 1px solid #313244;
  transition: all 0.3s ease;
}

.terminal-window.maximized {
  width: 98%;
  height: 95%;
  border-radius: 8px;
}

/* ===== Panel (slide) modes ===== */
.terminal-panel {
  position: fixed;
  z-index: 999;
  background: #1e1e2e;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.4);
  border: 1px solid #313244;
}

.terminal-panel.panel-top {
  top: 0;
  left: 0;
  right: 0;
  height: 50vh;
  border-bottom: 2px solid #89b4fa;
}

.terminal-panel.panel-bottom {
  bottom: 0;
  left: 0;
  right: 0;
  height: 50vh;
  border-top: 2px solid #89b4fa;
}

.terminal-panel.panel-left {
  top: 0;
  left: 0;
  bottom: 0;
  width: 50vw;
  border-right: 2px solid #89b4fa;
}

.terminal-panel.panel-right {
  top: 0;
  right: 0;
  bottom: 0;
  width: 50vw;
  border-left: 2px solid #89b4fa;
}

/* ===== Slide transitions ===== */
.slide-top-enter-active,
.slide-top-leave-active {
  transition: transform 0.35s cubic-bezier(0.4, 0, 0.2, 1);
}
.slide-top-enter-from,
.slide-top-leave-to {
  transform: translateY(-100%);
}

.slide-bottom-enter-active,
.slide-bottom-leave-active {
  transition: transform 0.35s cubic-bezier(0.4, 0, 0.2, 1);
}
.slide-bottom-enter-from,
.slide-bottom-leave-to {
  transform: translateY(100%);
}

.slide-left-enter-active,
.slide-left-leave-active {
  transition: transform 0.35s cubic-bezier(0.4, 0, 0.2, 1);
}
.slide-left-enter-from,
.slide-left-leave-to {
  transform: translateX(-100%);
}

.slide-right-enter-active,
.slide-right-leave-active {
  transition: transform 0.35s cubic-bezier(0.4, 0, 0.2, 1);
}
.slide-right-enter-from,
.slide-right-leave-to {
  transform: translateX(100%);
}

/* ===== Shared styles ===== */
.terminal-titlebar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 16px;
  background: #181825;
  border-bottom: 1px solid #313244;
  user-select: none;
  flex-shrink: 0;
}

.terminal-title {
  color: #cdd6f4;
  font-size: 13px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 8px;
}

.terminal-icon {
  font-size: 16px;
  color: #a6e3a1;
}

.terminal-controls {
  display: flex;
  gap: 8px;
}

.ctrl-btn {
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  transition: all 0.15s;
  background: #313244;
  color: #cdd6f4;
}

.ctrl-btn:hover {
  background: #45475a;
}

.close-btn:hover {
  background: #f38ba8;
  color: #1e1e2e;
}

.terminal-body {
  flex: 1;
  padding: 4px;
  overflow: hidden;
}

.terminal-body :deep(.xterm) {
  height: 100%;
}

.terminal-body :deep(.xterm-viewport) {
  overflow-y: auto !important;
}

.terminal-statusbar {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 16px;
  background: #181825;
  border-top: 1px solid #313244;
  font-size: 12px;
  color: #6c7086;
  flex-shrink: 0;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #f38ba8;
  transition: background 0.3s;
}

.status-dot.connected {
  background: #a6e3a1;
}
</style>
