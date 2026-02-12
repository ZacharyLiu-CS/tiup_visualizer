<template>
  <!-- Use v-show instead of v-if to keep DOM alive for proper cleanup -->
  <div class="terminal-overlay" v-show="visible" @click.self="handleClose">
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
      <div class="terminal-body" ref="terminalContainer"></div>
      <div class="terminal-statusbar">
        <span class="status-dot" :class="{ connected: wsConnected }"></span>
        <span>{{ wsConnected ? 'Connected' : 'Disconnected' }}</span>
      </div>
    </div>
  </div>
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
      _destroyed: false,
      _initTimer: null
    }
  },
  watch: {
    visible(val) {
      if (val) {
        this._destroyed = false
        this.$nextTick(() => this.initTerminal())
      } else {
        this.destroyTerminal()
      }
    }
  },
  methods: {
    handleClose() {
      this.destroyTerminal()
      this.$emit('close')
    },

    initTerminal() {
      if (this.terminal || this._destroyed) return

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

      const container = this.$refs.terminalContainer
      this.terminal.open(container)

      // Fit after a short delay to ensure DOM is laid out; track the timer for cleanup
      this._initTimer = setTimeout(() => {
        this._initTimer = null
        if (this._destroyed || !this.fitAddon) return
        this.fitAddon.fit()
        this.connectWebSocket()
      }, 100)

      // Handle user input
      this.terminal.onData(data => {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
          this.ws.send(data)
        }
      })

      // Watch for container resize
      this.resizeObserver = new ResizeObserver(() => {
        if (this._destroyed) return
        if (this.fitAddon && this.terminal) {
          this.fitAddon.fit()
          this.sendResize()
        }
      })
      this.resizeObserver.observe(container)
    },

    connectWebSocket() {
      if (this._destroyed) return

      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const basePath = import.meta.env.BASE_URL.replace(/\/+$/, '')
      const wsUrl = `${protocol}//${window.location.host}${basePath}/ws/terminal`

      this.ws = new WebSocket(wsUrl)
      this.ws.binaryType = 'arraybuffer'

      this.ws.onopen = () => {
        if (this._destroyed) return
        this.wsConnected = true
        this.sendResize()
      }

      this.ws.onmessage = (event) => {
        if (this._destroyed || !this.terminal) return
        if (event.data instanceof ArrayBuffer) {
          this.terminal.write(new Uint8Array(event.data))
        } else {
          this.terminal.write(event.data)
        }
      }

      this.ws.onclose = () => {
        if (this._destroyed) return
        this.wsConnected = false
        if (this.terminal) {
          this.terminal.write('\r\n\x1b[31m[Connection closed]\x1b[0m\r\n')
        }
      }

      this.ws.onerror = () => {
        if (this._destroyed) return
        this.wsConnected = false
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
        if (this.fitAddon && !this._destroyed) {
          this.fitAddon.fit()
          this.sendResize()
        }
      })
    },

    destroyTerminal() {
      this._destroyed = true

      // Cancel pending init timer
      if (this._initTimer) {
        clearTimeout(this._initTimer)
        this._initTimer = null
      }

      // Disconnect resize observer first
      if (this.resizeObserver) {
        this.resizeObserver.disconnect()
        this.resizeObserver = null
      }

      // Close WebSocket — nullify callbacks to prevent post-destroy side effects
      if (this.ws) {
        this.ws.onopen = null
        this.ws.onmessage = null
        this.ws.onclose = null
        this.ws.onerror = null
        this.ws.close()
        this.ws = null
      }

      // Dispose xterm instance
      if (this.terminal) {
        this.terminal.dispose()
        this.terminal = null
      }

      this.fitAddon = null
      this.wsConnected = false
      this.isMaximized = false
    }
  },
  beforeUnmount() {
    this.destroyTerminal()
  }
}
</script>

<style scoped>
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
