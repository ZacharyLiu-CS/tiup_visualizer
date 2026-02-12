<template>
  <div class="home-view">
    <header class="app-header">
      <h1>TiUP Cluster Visualizer</h1>
      <div class="header-actions">
        <button @click="showTerminal = true" class="terminal-btn" title="Open Terminal">
          &#9002; Terminal
        </button>
        <button @click="showServerLogs = true" class="server-log-btn" title="View Backend Logs">
          &#128196; Server Logs
        </button>
        <button v-if="selectedHost || selectedCluster" @click="clearSelection" class="clear-btn">
          Clear Selection
        </button>
        <div class="header-divider"></div>
        <div class="user-info">
          <svg viewBox="0 0 20 20" fill="currentColor" width="16" height="16">
            <path fill-rule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clip-rule="evenodd"/>
          </svg>
          <span>{{ username }}</span>
        </div>
        <button class="logout-btn" @click="$emit('logout')" title="Sign Out">
          <svg viewBox="0 0 20 20" fill="currentColor" width="14" height="14">
            <path fill-rule="evenodd" d="M3 3a1 1 0 00-1 1v12a1 1 0 001 1h12a1 1 0 001-1V4a1 1 0 00-1-1H3zm7.707 3.293a1 1 0 010 1.414L9.414 9H17a1 1 0 110 2H9.414l1.293 1.293a1 1 0 01-1.414 1.414l-3-3a1 1 0 010-1.414l3-3a1 1 0 011.414 0z" clip-rule="evenodd"/>
          </svg>
          Sign Out
        </button>
      </div>
    </header>

    <!-- Web Terminal -->
    <WebTerminal :visible="showTerminal" @close="showTerminal = false" />

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

.terminal-btn {
  background: rgba(255, 255, 255, 0.15);
  border: 1px solid rgba(255, 255, 255, 0.4);
  color: white;
  padding: 8px 16px;
  border-radius: 6px;
  cursor: pointer;
  font-weight: 600;
  font-size: 14px;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 6px;
}

.terminal-btn:hover {
  background: rgba(255, 255, 255, 0.25);
  border-color: white;
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
}

.server-log-btn {
  background: rgba(255, 255, 255, 0.15);
  border: 1px solid rgba(255, 255, 255, 0.4);
  color: white;
  padding: 8px 16px;
  border-radius: 6px;
  cursor: pointer;
  font-weight: 600;
  font-size: 14px;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 6px;
}

.server-log-btn:hover {
  background: rgba(255, 255, 255, 0.25);
  border-color: white;
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
}

.clear-btn {
  background: rgba(255, 255, 255, 0.2);
  border: 1px solid white;
  color: white;
  padding: 8px 16px;
  border-radius: 6px;
  cursor: pointer;
  font-weight: 600;
  transition: all 0.2s;
}

.clear-btn:hover {
  background: white;
  color: #667eea;
}

.header-divider {
  width: 1px;
  height: 24px;
  background: rgba(255, 255, 255, 0.3);
}

.user-info {
  display: flex;
  align-items: center;
  gap: 6px;
  color: rgba(255, 255, 255, 0.8);
  font-size: 13px;
}

.logout-btn {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 6px 12px;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.25);
  border-radius: 6px;
  color: rgba(255, 255, 255, 0.85);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.logout-btn:hover {
  background: rgba(239, 68, 68, 0.7);
  border-color: rgba(239, 68, 68, 0.8);
  color: #fff;
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
