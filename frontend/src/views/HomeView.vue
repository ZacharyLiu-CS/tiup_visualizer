<template>
  <div class="home-view">
    <header class="app-header">
      <h1>TiUP Cluster Visualizer</h1>
      <button v-if="selectedHost || selectedCluster" @click="clearSelection" class="clear-btn">
        Clear Selection
      </button>
    </header>

    <div class="loading-overlay" v-if="loading">
      <div class="spinner"></div>
      <p>Loading clusters...</p>
    </div>

    <div class="error-message" v-if="error">
      <p>Error: {{ error }}</p>
      <button @click="refresh">Retry</button>
    </div>

    <div class="main-container" v-if="!loading && !error">
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
            @select="handleClusterSelect"
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

export default {
  name: 'HomeView',
  components: {
    HostCard,
    ClusterCard,
    ClusterDetailModal,
    ConnectionLines
  },
  data() {
    return {
      hostRefs: {},
      clusterRefs: {},
      connectionLines: []
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
      'fetchClusters',
      'fetchHosts',
      'selectHost',
      'selectCluster',
      'clearSelection',
      'getHostsForCluster'
    ]),
    async refresh() {
      await Promise.all([
        this.fetchClusters(),
        this.fetchHosts()
      ])
    },
    async handleHostSelect(host) {
      if (this.selectedHost === host) {
        this.clearSelection()
      } else {
        this.selectHost(host)
      }
    },
    async handleClusterSelect(clusterName) {
      if (this.selectedCluster === clusterName) {
        this.clearSelection()
      } else {
        await this.selectCluster(clusterName)
      }
    },
    closeClusterDetail() {
      this.clearSelection()
    },
    updateConnectionLines() {
      this.connectionLines = []
      
      if (this.selectedHost) {
        // Draw lines from selected host to its clusters
        const highlightedClusters = this.highlightedClusters
        highlightedClusters.forEach(clusterName => {
          const line = this.calculateLine(this.selectedHost, clusterName, true)
          if (line) this.connectionLines.push(line)
        })
      } else if (this.selectedCluster) {
        // Draw lines from selected cluster to its hosts
        const highlightedHosts = this.highlightedHosts
        highlightedHosts.forEach(host => {
          const line = this.calculateLine(host, this.selectedCluster, false)
          if (line) this.connectionLines.push(line)
        })
      }
    },
    calculateLine(hostKey, clusterName, fromHost) {
      const hostEl = this.hostRefs[hostKey]?.$el
      const clusterEl = this.clusterRefs[clusterName]?.$el
      
      if (!hostEl || !clusterEl) return null

      const hostRect = hostEl.getBoundingClientRect()
      const clusterRect = clusterEl.getBoundingClientRect()
      const container = this.$el.getBoundingClientRect()

      return {
        x1: hostRect.left + hostRect.width / 2 - container.left,
        y1: hostRect.bottom - container.top,
        x2: clusterRect.left + clusterRect.width / 2 - container.left,
        y2: clusterRect.top - container.top,
        color: fromHost ? '#3b82f6' : '#8b5cf6'
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
  z-index: 10;
}

.clusters-section {
  margin-bottom: 24px;
}

.clusters-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  position: relative;
  z-index: 10;
}
</style>
