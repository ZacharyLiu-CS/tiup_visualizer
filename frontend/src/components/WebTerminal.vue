<template>
  <!--
    Hidden host: keeps xterm DOM alive when the panel is closed.
    Positioned off-screen so it doesn't affect layout.
  -->
  <div ref="hiddenHost" class="terminal-hidden-host"></div>

  <!-- Float (overlay) mode -->
  <teleport to="body">
    <transition name="fade" @after-leave="onTransitionLeave">
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
    <transition :name="slideTransitionName" @after-leave="onTransitionLeave">
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
      // Track whether session was ended by exit/disconnect (not user closing panel)
      sessionDead: false
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
    },
    /** Whether we have a living session that can be reused */
    hasLiveSession() {
      return (
        this.terminal &&
        this.ws &&
        this.ws.readyState === WebSocket.OPEN &&
        !this.sessionDead
      )
    }
  },
  watch: {
    visible(val) {
      if (val) {
        this._waitForTarget(() => {
          if (this.hasLiveSession) {
            // Reuse existing session: move xterm DOM into the visible container
            this.reattachTerminal()
          } else {
            // No live session: destroy stale resources and create fresh
            this.destroyTerminal()
            this.createAndAttach()
          }
        })
      } else {
        // Panel closing: park xterm DOM into hidden host so it survives v-if removal
        this.parkTerminal()
      }
    },
    mode() {
      if (this.visible && this.terminal) {
        this.$nextTick(() => {
          this._waitForTarget(() => {
            this.reattachTerminal()
          })
        })
      }
    }
  },
  methods: {
    handleClose() {
      this.$emit('close')
    },

    /**
     * Called by <transition @after-leave> — transition animation is fully done.
     * Terminal is already parked in hidden host, so nothing to destroy here
     * unless the session is dead.
     */
    onTransitionLeave() {
      if (this.sessionDead) {
        this.destroyTerminal()
      }
    },

    /**
     * Poll until the target container ref exists (transition may not have rendered yet).
     */
    _waitForTarget(callback, maxRetries = 30) {
      let retries = 0
      const poll = () => {
        const target = this._getTarget()
        if (target) {
          callback()
          return
        }
        if (retries < maxRetries && this.visible) {
          retries++
          setTimeout(poll, 30)
          return
        }
        callback()
      }
      this.$nextTick(poll)
    },

    _getTarget() {
      return this.mode === 'float' ? this.$refs.floatBody : this.$refs.panelBody
    },

    /**
     * Move xterm DOM from wherever it is into the hidden host div.
     * This keeps it alive when the panel v-if removes the container.
     */
    parkTerminal() {
      if (!this.terminal || !this.terminal.element) return
      const hidden = this.$refs.hiddenHost
      if (!hidden) return
      const el = this.terminal.element
      if (el.parentNode !== hidden) {
        hidden.appendChild(el)
      }
      // Disconnect resize observer since the panel is hidden
      if (this.resizeObserver) {
        this.resizeObserver.disconnect()
      }
    },

    /**
     * Move xterm DOM from hidden host back into the visible container.
     * Reuse existing terminal + WebSocket session.
     */
    reattachTerminal() {
      const target = this._getTarget()
      if (!target || !this.terminal || !this.terminal.element) return

      const el = this.terminal.element
      if (el.parentNode !== target) {
        target.appendChild(el)
      }

      // Re-setup resize observer on the new target
      if (this.resizeObserver) {
        this.resizeObserver.disconnect()
      }
      this.resizeObserver = new ResizeObserver(() => {
        if (this.fitAddon && this.terminal && this.visible) {
          try { this.fitAddon.fit() } catch (_) { /* ignore */ }
          this.sendResize()
        }
      })
      this.resizeObserver.observe(target)

      // Fit to the new container size
      this.$nextTick(() => {
        if (this.fitAddon) {
          try { this.fitAddon.fit() } catch (_) { /* ignore */ }
          this.sendResize()
        }
        // Ensure terminal gets focus
        if (this.terminal) {
          this.terminal.focus()
        }
      })
    },

    /**
     * Create a brand-new terminal + WebSocket and attach into the visible container.
     * Only called when there's no live session to reuse.
     */
    createAndAttach() {
      this.sessionDead = false

      const target = this._getTarget()
      if (!target) return

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

      this.terminal.open(target)

      this.terminal.onData(data => {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
          this.ws.send(data)
        }
      })

      this.connectWebSocket()

      this.$nextTick(() => {
        if (this.fitAddon) {
          try { this.fitAddon.fit() } catch (_) { /* ignore */ }
          this.sendResize()
        }
      })

      this.resizeObserver = new ResizeObserver(() => {
        if (this.fitAddon && this.terminal && this.visible) {
          try { this.fitAddon.fit() } catch (_) { /* ignore */ }
          this.sendResize()
        }
      })
      this.resizeObserver.observe(target)
    },

    connectWebSocket() {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const basePath = import.meta.env.BASE_URL.replace(/\/+$/, '')
      const token = localStorage.getItem('auth_token') || ''
      const wsUrl = `${protocol}//${window.location.host}${basePath}/ws/terminal?token=${encodeURIComponent(token)}`

      this.ws = new WebSocket(wsUrl)
      this.ws.binaryType = 'arraybuffer'

      this.ws.onopen = () => {
        this.wsConnected = true
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
        this.sessionDead = true
        // Clean up ws reference
        if (this.ws) {
          this.ws.onopen = null
          this.ws.onmessage = null
          this.ws.onclose = null
          this.ws.onerror = null
          this.ws = null
        }
        // Session ended (e.g. user typed exit) — auto-close the terminal panel
        if (this.visible) {
          setTimeout(() => {
            if (this.visible) {
              this.$emit('close')
            }
          }, 0)
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
     * Full cleanup — destroy terminal + WebSocket completely.
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
        try {
          if (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING) {
            this.ws.close()
          }
        } catch (_) { /* ignore */ }
        this.ws = null
      }

      if (this.terminal) {
        try { this.terminal.dispose() } catch (_) { /* ignore */ }
        this.terminal = null
      }

      this.fitAddon = null
      this.wsConnected = false
      this.sessionDead = false
    }
  },
  beforeUnmount() {
    this.destroyTerminal()
  }
}
</script>

<style scoped>
/* ===== Hidden host: keeps xterm DOM alive off-screen ===== */
.terminal-hidden-host {
  position: fixed;
  left: -9999px;
  top: -9999px;
  width: 1px;
  height: 1px;
  overflow: hidden;
  pointer-events: none;
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
