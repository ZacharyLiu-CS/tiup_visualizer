<template>
  <div class="home-view" :class="terminalLayoutClass">
    <header class="app-header">
      <h1>TiUP Cluster Visualizer</h1>
      <div class="header-actions">
        <div class="terminal-group">
          <button @click="showTerminal = !showTerminal" class="header-btn" title="Toggle Terminal">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="16" height="16">
              <polyline points="4 17 10 11 4 5" /><line x1="12" y1="19" x2="20" y2="19" />
            </svg>
            Terminal
          </button>
          <div class="terminal-mode-select" ref="terminalModeSelect">
            <button class="mode-trigger" @click="showModeDropdown = !showModeDropdown" type="button">
              <span class="mode-label">{{ terminalModeLabel }}</span>
              <svg class="mode-arrow" :class="{ open: showModeDropdown }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" width="12" height="12">
                <polyline points="6 9 12 15 18 9" />
              </svg>
            </button>
            <transition name="dropdown">
              <ul v-show="showModeDropdown" class="mode-dropdown">
                <li v-for="opt in terminalModeOptions" :key="opt.value"
                    :class="{ active: terminalMode === opt.value }"
                    @click="selectTerminalMode(opt.value)">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="14" height="14">
                    <polyline v-if="opt.value === 'top'" points="12 5 12 19" /><polyline v-if="opt.value === 'top'" points="5 12 12 19 19 12" />
                    <polyline v-if="opt.value === 'bottom'" points="12 19 12 5" /><polyline v-if="opt.value === 'bottom'" points="5 12 12 5 19 12" />
                    <polyline v-if="opt.value === 'left'" points="19 12 5 12" /><polyline v-if="opt.value === 'left'" points="12 5 5 12 12 19" />
                    <polyline v-if="opt.value === 'right'" points="5 12 19 12" /><polyline v-if="opt.value === 'right'" points="12 5 19 12 12 19" />
                    <template v-if="opt.value === 'float'">
                      <rect x="3" y="3" width="18" height="18" rx="2" ry="2" /><rect x="7" y="7" width="10" height="10" rx="1" ry="1" />
                    </template>
                  </svg>
                  <span>{{ opt.label }}</span>
                  <svg v-if="terminalMode === opt.value" class="check-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" width="14" height="14">
                    <polyline points="20 6 9 17 4 12" />
                  </svg>
                </li>
              </ul>
            </transition>
          </div>
        </div>
        <button @click="showServerLogs = true" class="header-btn" title="View Backend Logs">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="16" height="16">
            <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z" /><polyline points="14 2 14 8 20 8" /><line x1="16" y1="13" x2="8" y2="13" /><line x1="16" y1="17" x2="8" y2="17" /><polyline points="10 9 9 9 8 9" />
          </svg>
          Server Logs
        </button>
        <button v-if="selectedHost || selectedCluster" @click="clearSelection" class="header-btn">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="16" height="16">
            <line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" />
          </svg>
          Clear
        </button>
        <div class="header-divider"></div>
        <div class="user-info">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="16" height="16">
            <path d="M20 21v-2a4 4 0 00-4-4H8a4 4 0 00-4 4v2" /><circle cx="12" cy="7" r="4" />
          </svg>
          <span>{{ username }}</span>
        </div>
        <button class="header-btn btn-danger" @click="$emit('logout')" title="Sign Out">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="16" height="16">
            <path d="M9 21H5a2 2 0 01-2-2V5a2 2 0 012-2h4" /><polyline points="16 17 21 12 16 7" /><line x1="21" y1="12" x2="9" y2="12" />
          </svg>
          Sign Out
        </button>
      </div>
    </header>

    <!-- Web Terminal -->
    <WebTerminal :visible="showTerminal" :mode="terminalMode" @close="showTerminal = false" />

    <!-- Server Log Modal -->
    <ServerLogModal :visible="showServerLogs" @close="showServerLogs = false" />

    <div class="loading-overlay" v-if="loading">
      <div class="spinner"></div>
      <p>Loading clusters...</p>
    </div>

    <div class="error-message" v-if="error">
      <p>Error: {{ error }}</p>
      <button @click="refresh">Retry</button>
    </div>

    <div class="main-container" ref="mainContainer" v-if="!loading && !error">
      <!-- Hosts Section (Top) -->
      <section class="hosts-section">
        <h2 class="section-title">Physical Hosts</h2>
        <div class="hosts-grid" ref="hostsGrid">
          <HostCard 
            v-for="(hostInfo, host) in hosts" 
            :key="host"
            :host="host"
            :hostInfo="hostInfo"
            :isSelected="selectedHost === host"
            :isHighlighted="highlightedHosts.includes(host)"
            :clusterIndexMap="clusterIndexMap"
            :allClusters="clusters"
            @select="handleHostSelect"
            :ref="el => { if (el) hostRefs[host] = el }"
          />
        </div>
      </section>

      <!-- Clusters Section (Bottom) -->
      <section class="clusters-section">
        <h2 class="section-title">TiKV Clusters</h2>
        <div class="clusters-grid" ref="clustersGrid">
          <ClusterCard 
            v-for="cluster in clusters" 
            :key="cluster.name"
            :cluster="cluster"
            :index="clusterIndexMap[cluster.name]"
            :isSelected="selectedCluster === cluster.name"
            :isHighlighted="highlightedClusters.includes(cluster.name)"
            @connect="handleClusterConnect"
            @detail="handleClusterDetail"
            :ref="el => { if (el) clusterRefs[cluster.name] = el }"
          />
        </div>
      </section>

      <!-- Connection Lines -->
      <ConnectionLines :lines="connectionLines" />

      <!-- Cluster Detail Modal -->
      <ClusterDetailModal 
        :clusterDetail="clusterDetail"
        @close="closeClusterDetail"
      />
    </div>
  </div>
</template>

<script>
import { mapState, mapActions } from 'pinia'
import { useClusterStore } from '../stores/cluster'
import HostCard from '../components/HostCard.vue'
import ClusterCard from '../components/ClusterCard.vue'
import ClusterDetailModal from '../components/ClusterDetailModal.vue'
import ConnectionLines from '../components/ConnectionLines.vue'
import WebTerminal from '../components/WebTerminal.vue'
import ServerLogModal from '../components/ServerLogModal.vue'

export default {
  name: 'HomeView',
  components: {
    HostCard,
    ClusterCard,
    ClusterDetailModal,
    ConnectionLines,
    WebTerminal,
    ServerLogModal
  },
  props: {
    username: {
      type: String,
      default: ''
    }
  },
  emits: ['logout'],
  data() {
    return {
      hostRefs: {},
      clusterRefs: {},
      connectionLines: [],
      showTerminal: false,
      terminalMode: 'bottom',
      showModeDropdown: false,
      terminalModeOptions: [
        { value: 'top', label: 'Top' },
        { value: 'right', label: 'Right' },
        { value: 'bottom', label: 'Bottom' },
        { value: 'left', label: 'Left' },
        { value: 'float', label: 'Float' }
      ],
      showServerLogs: false
    }
  },
  computed: {
    ...mapState(useClusterStore, [
      'clusters',
      'hosts',
      'selectedHost',
      'selectedCluster',
      'clusterDetail',
      'loading',
      'error'
    ]),
    terminalLayoutClass() {
      if (!this.showTerminal || this.terminalMode === 'float') return ''
      return `terminal-push-${this.terminalMode}`
    },
    terminalModeLabel() {
      const opt = this.terminalModeOptions.find(o => o.value === this.terminalMode)
      return opt ? opt.label : 'Top'
    },
    clusterIndexMap() {
      const map = {}
      this.clusters.forEach((cluster, idx) => {
        map[cluster.name] = idx + 1
      })
      return map
    },
    highlightedHosts() {
      if (this.selectedCluster) {
        return this.getHostsForCluster(this.selectedCluster)
      }
      return []
    },
    highlightedClusters() {
      if (this.selectedHost && this.hosts[this.selectedHost]) {
        return this.hosts[this.selectedHost].clusters
      }
      return []
    }
  },
  async mounted() {
    await this.refresh()
    document.addEventListener('click', this.handleClickOutside)
  },
  beforeUnmount() {
    document.removeEventListener('click', this.handleClickOutside)
  },
  watch: {
    selectedHost() {
      this.$nextTick(() => this.updateConnectionLines())
    },
    selectedCluster() {
      this.$nextTick(() => this.updateConnectionLines())
    }
  },
  methods: {
    ...mapActions(useClusterStore, [
      'fetchOverview',
      'fetchClusters',
      'fetchHosts',
      'selectHost',
      'selectClusterForConnect',
      'selectClusterForDetail',
      'clearSelection',
      'getHostsForCluster'
    ]),
    async refresh() {
      await this.fetchOverview()
    },
    selectTerminalMode(value) {
      this.terminalMode = value
      this.showModeDropdown = false
    },
    handleClickOutside(e) {
      const el = this.$refs.terminalModeSelect
      if (el && !el.contains(e.target)) {
        this.showModeDropdown = false
      }
    },
    async handleHostSelect(host) {
      if (this.selectedHost === host) {
        this.clearSelection()
      } else {
        this.selectHost(host)
      }
    },
    handleClusterConnect(clusterName) {
      if (this.selectedCluster === clusterName) {
        this.clearSelection()
      } else {
        this.selectClusterForConnect(clusterName)
      }
    },
    async handleClusterDetail(clusterName) {
      await this.selectClusterForDetail(clusterName)
    },
    closeClusterDetail() {
      this.clearSelection()
    },
    updateConnectionLines() {
      this.connectionLines = []
      
      if (this.selectedHost) {
        const highlightedClusters = this.highlightedClusters
        const lines = highlightedClusters.map(clusterName => 
          this.calculateLine(this.selectedHost, clusterName, true)
        ).filter(Boolean)
        this.distributeLines(lines)
      } else if (this.selectedCluster) {
        const highlightedHosts = this.highlightedHosts
        const lines = highlightedHosts.map(host => 
          this.calculateLine(host, this.selectedCluster, false)
        ).filter(Boolean)
        this.distributeLines(lines)
      }
    },
    distributeLines(lines) {
      if (!lines.length) return
      const container = this.$refs.mainContainer
      if (!container) return
      const containerRect = container.getBoundingClientRect()
      const hostsGrid = this.$refs.hostsGrid
      const clustersGrid = this.$refs.clustersGrid
      if (!hostsGrid || !clustersGrid) {
        this.connectionLines = lines
        return
      }
      const hostsRect = hostsGrid.getBoundingClientRect()
      const clustersRect = clustersGrid.getBoundingClientRect()
      const gapTop = hostsRect.bottom - containerRect.top + 8
      const gapBottom = clustersRect.top - containerRect.top - 8
      const gapHeight = gapBottom - gapTop

      // Group lines by cluster row (same y2 = same row)
      const rowGroups = {}
      lines.forEach(line => {
        const rowKey = Math.round(line.y2)
        if (!rowGroups[rowKey]) rowGroups[rowKey] = []
        rowGroups[rowKey].push(line)
      })

      // Assign one midY per distinct row
      const rowKeys = Object.keys(rowGroups).sort((a, b) => Number(a) - Number(b))
      const rowCount = rowKeys.length
      rowKeys.forEach((rowKey, i) => {
        const midY = gapTop + (gapHeight * (i + 1)) / (rowCount + 1)
        rowGroups[rowKey].forEach(line => {
          line.path = `M${line.x1},${line.y1} V${midY} H${line.x2} V${line.y2}`
        })
      })
      this.connectionLines = lines
    },
    calculateLine(hostKey, clusterName, fromHost) {
      const hostEl = this.hostRefs[hostKey]?.$el
      const clusterEl = this.clusterRefs[clusterName]?.$el
      
      if (!hostEl || !clusterEl) return null

      const container = this.$refs.mainContainer
      if (!container) return null

      const hostRect = hostEl.getBoundingClientRect()
      const clusterRect = clusterEl.getBoundingClientRect()
      const containerRect = container.getBoundingClientRect()

      const x1 = hostRect.left + hostRect.width / 2 - containerRect.left
      const y1 = hostRect.bottom - containerRect.top
      const x2 = clusterRect.left + clusterRect.width / 2 - containerRect.left
      const y2 = clusterRect.top - containerRect.top

      return {
        x1, y1, x2, y2,
        color: fromHost ? '#3b82f6' : '#8b5cf6',
        path: ''
      }
    }
  }
}
</script>

<style scoped>
.home-view {
  min-height: 100vh;
  background: #f9fafb;
  position: relative;
  transition: margin 0.35s cubic-bezier(0.4, 0, 0.2, 1),
              padding 0.35s cubic-bezier(0.4, 0, 0.2, 1);
}

/* Push layouts when terminal panel is open */
.home-view.terminal-push-top {
  margin-top: 50vh;
}

.home-view.terminal-push-bottom {
  margin-bottom: 50vh;
}

.home-view.terminal-push-left {
  margin-left: 50vw;
}

.home-view.terminal-push-right {
  margin-right: 50vw;
}

.app-header {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  padding: 24px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.app-header h1 {
  margin: 0;
  font-size: 28px;
  font-weight: 700;
}

.header-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}

/* Unified header button */
.header-btn {
  background: rgba(255, 255, 255, 0.12);
  border: 1px solid rgba(255, 255, 255, 0.35);
  color: white;
  padding: 0 14px;
  height: 36px;
  border-radius: 6px;
  cursor: pointer;
  font-weight: 600;
  font-size: 13px;
  transition: all 0.2s;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
  line-height: 1;
}

.header-btn:hover {
  background: rgba(255, 255, 255, 0.22);
  border-color: rgba(255, 255, 255, 0.6);
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

.header-btn svg {
  flex-shrink: 0;
}

.header-btn.btn-danger:hover {
  background: rgba(239, 68, 68, 0.65);
  border-color: rgba(239, 68, 68, 0.8);
}

.terminal-group {
  display: flex;
  align-items: stretch;
  gap: 0;
}

.terminal-group .header-btn {
  border-radius: 6px 0 0 6px;
  border-right: none;
}

.terminal-mode-select {
  position: relative;
  display: flex;
  align-items: stretch;
}

.mode-trigger {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: rgba(255, 255, 255, 0.12);
  border: 1px solid rgba(255, 255, 255, 0.35);
  border-left: 1px solid rgba(255, 255, 255, 0.2);
  color: white;
  padding: 0 10px;
  height: 36px;
  border-radius: 0 6px 6px 0;
  cursor: pointer;
  font-weight: 600;
  font-size: 13px;
  transition: all 0.2s;
  line-height: 1;
  white-space: nowrap;
}

.mode-trigger:hover {
  background: rgba(255, 255, 255, 0.22);
  border-color: rgba(255, 255, 255, 0.6);
}

.mode-label {
  min-width: 38px;
  text-align: left;
}

.mode-arrow {
  transition: transform 0.2s;
  opacity: 0.8;
}

.mode-arrow.open {
  transform: rotate(180deg);
}

.mode-dropdown {
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  min-width: 150px;
  background: #1e1e2e;
  border: 1px solid #313244;
  border-radius: 8px;
  padding: 4px;
  list-style: none;
  margin: 0;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
  z-index: 1100;
}

.mode-dropdown li {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  color: #cdd6f4;
  font-size: 13px;
  font-weight: 500;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s;
  white-space: nowrap;
}

.mode-dropdown li:hover {
  background: #313244;
}

.mode-dropdown li.active {
  background: #89b4fa;
  color: #1e1e2e;
}

.mode-dropdown li.active svg {
  stroke: #1e1e2e;
}

.mode-dropdown li .check-icon {
  margin-left: auto;
}

.mode-dropdown li svg {
  flex-shrink: 0;
  stroke: #cdd6f4;
}

/* dropdown transition */
.dropdown-enter-active,
.dropdown-leave-active {
  transition: opacity 0.15s, transform 0.15s;
}
.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

.header-divider {
  width: 1px;
  height: 24px;
  background: rgba(255, 255, 255, 0.3);
}

.user-info {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: rgba(255, 255, 255, 0.85);
  font-size: 13px;
  font-weight: 600;
  height: 36px;
}

.user-info svg {
  flex-shrink: 0;
  opacity: 0.75;
}

.loading-overlay {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
}

.spinner {
  width: 50px;
  height: 50px;
  border: 4px solid #e5e7eb;
  border-top-color: #3b82f6;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.error-message {
  text-align: center;
  padding: 40px;
  color: #dc2626;
}

.error-message button {
  margin-top: 16px;
  padding: 8px 24px;
  background: #3b82f6;
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-weight: 600;
}

.main-container {
  position: relative;
  padding: 24px;
}

.section-title {
  font-size: 20px;
  font-weight: 700;
  color: #1f2937;
  margin-bottom: 16px;
  padding-left: 8px;
  border-left: 4px solid #3b82f6;
}

.hosts-section {
  margin-bottom: 48px;
}

.hosts-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  position: relative;
}

.clusters-section {
  margin-bottom: 24px;
}

.clusters-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  position: relative;
}
</style>
