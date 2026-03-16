<template>
  <div class="cluster-detail-modal" v-if="clusterDetail" @click.self="close">
    <div class="modal-content">
      <div class="modal-header">
        <h2>{{ clusterDetail.cluster_name }}</h2>
        <button class="close-btn" @click="close">&times;</button>
      </div>
      
      <div class="modal-body">
        <div class="cluster-info-section">
          <div class="info-row">
            <span class="label">Cluster Type:</span>
            <span class="value">{{ clusterDetail.cluster_type }}</span>
          </div>
          <div class="info-row">
            <span class="label">Version:</span>
            <span class="value">{{ clusterDetail.cluster_version }}</span>
          </div>
          <div class="info-row">
            <span class="label">Deploy User:</span>
            <span class="value">{{ clusterDetail.deploy_user }}</span>
          </div>
          <div class="info-row" v-if="clusterDetail.dashboard_url">
            <span class="label">Dashboard:</span>
            <a :href="clusterDetail.dashboard_url" target="_blank" class="value link">
              {{ clusterDetail.dashboard_url }}
            </a>
          </div>
          <div class="info-row" v-if="clusterDetail.grafana_url">
            <span class="label">Grafana:</span>
            <a :href="clusterDetail.grafana_url" target="_blank" class="value link">
              {{ clusterDetail.grafana_url }}
            </a>
          </div>
        </div>

        <div class="components-section">
          <h3>Components ({{ clusterDetail.components.length }})</h3>
          <div class="table-scroll-wrapper">
            <div class="table-container">
              <table class="components-table">
                <thead>
                  <tr>
                    <th>ID</th>
                    <th>Role</th>
                    <th>Host</th>
                    <th>Ports</th>
                    <th>Status</th>
                    <th>Data Dir</th>
                    <th>Deploy Dir</th>
                    <th>Log</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="component in clusterDetail.components" :key="component.id">
                    <td class="id-cell">{{ component.id }}</td>
                    <td>
                      <span class="role-badge" :class="`role-${component.role}`">
                        {{ component.role }}
                      </span>
                    </td>
                    <td class="host-cell">{{ component.host }}</td>
                    <td>{{ component.ports }}</td>
                    <td>
                      <span class="status-badge" :class="getStatusClass(component.status)">
                        {{ component.status }}
                      </span>
                    </td>
                    <td class="path-cell" :title="component.data_dir">
                      <div class="path-scroll">{{ component.data_dir }}</div>
                    </td>
                    <td class="path-cell" :title="component.deploy_dir">
                      <div class="path-scroll">{{ component.deploy_dir }}</div>
                    </td>
                    <td class="log-cell">
                      <div class="log-files" v-if="component.log_files && component.log_files.length">
                        <div class="log-file-row" v-for="logFile in component.log_files" :key="logFile.filename">
                          <span class="log-filename" :title="logFile.filename">{{ logFile.filename }}</span>
                          <div class="log-actions">
                            <button
                              class="log-btn log-btn-view"
                              @click="viewLog(component, logFile.filename)"
                              title="View log in browser"
                            >View</button>
                            <button
                              class="log-btn log-btn-download"
                              @click="downloadLog(component, logFile.filename)"
                              title="Download log file"
                            >Download</button>
                            <button
                              class="log-btn log-btn-ai"
                              :class="{ 'log-btn-ai-copied': aiCopyStatus[`${component.id}-${logFile.filename}`] === 'copied' }"
                              @click="aiAnalysis(component, logFile.filename)"
                              :title="aiCopyStatus[`${component.id}-${logFile.filename}`] === 'copied' ? 'Prompt copied! Paste it in Knot AI' : 'Copy prompt + Download log + Open Knot AI'"
                            >{{ aiCopyStatus[`${component.id}-${logFile.filename}`] === 'copied' ? 'Copied!' : 'AI Analysis' }}</button>
                          </div>
                        </div>
                      </div>
                      <span v-else class="no-logs">-</span>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Toast notification -->
    <transition name="toast">
      <div v-if="toastVisible" class="ai-toast">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="18" height="18">
          <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" /><polyline points="22 4 12 14.01 9 11.01" />
        </svg>
        <span>{{ toastMessage }}</span>
      </div>
    </transition>
  </div>
</template>

<script>
import { clusterAPI } from '../services/api'
import { buildAIPrompt } from '../config/ai-prompt'

export default {
  name: 'ClusterDetailModal',
  props: {
    clusterDetail: {
      type: Object,
      default: null
    }
  },
  emits: ['close', 'openAIAnalysis'],
  data() {
    return {
      aiCopyStatus: {},
      toastMessage: '',
      toastVisible: false,
      toastTimer: null
    }
  },
  methods: {
    close() {
      this.$emit('close')
    },
    showToast(msg) {
      this.toastMessage = msg
      this.toastVisible = true
      if (this.toastTimer) clearTimeout(this.toastTimer)
      this.toastTimer = setTimeout(() => {
        this.toastVisible = false
      }, 4000)
    },
    getStatusClass(status) {
      if (status.includes('Up')) return 'status-up'
      if (status.includes('Down')) return 'status-down'
      return 'status-unknown'
    },
    viewLog(component, filename) {
      const url = clusterAPI.getLogFileUrl(
        this.clusterDetail.cluster_name,
        component.id,
        filename,
        'view'
      )
      window.open(url, '_blank')
    },
    downloadLog(component, filename) {
      const url = clusterAPI.getLogFileUrl(
        this.clusterDetail.cluster_name,
        component.id,
        filename,
        'download'
      )
      const link = document.createElement('a')
      link.href = url
      link.download = filename
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
    },
    async aiAnalysis(component, filename) {
      const key = `${component.id}-${filename}`

      // 1. Build and copy prompt to clipboard
      const prompt = buildAIPrompt({
        clusterName: this.clusterDetail.cluster_name,
        clusterVersion: this.clusterDetail.cluster_version,
        clusterType: this.clusterDetail.cluster_type,
        componentRole: component.role,
        componentId: component.id,
        componentHost: component.host,
        componentStatus: component.status,
        logFilename: filename,
        deployDir: component.deploy_dir,
        dataDir: component.data_dir,
        ports: component.ports,
      })

      try {
        await navigator.clipboard.writeText(prompt)
      } catch {
        const textarea = document.createElement('textarea')
        textarea.value = prompt
        textarea.style.position = 'fixed'
        textarea.style.opacity = '0'
        document.body.appendChild(textarea)
        textarea.select()
        document.execCommand('copy')
        document.body.removeChild(textarea)
      }

      this.aiCopyStatus = { ...this.aiCopyStatus, [key]: 'copied' }
      setTimeout(() => {
        const s = { ...this.aiCopyStatus }
        delete s[key]
        this.aiCopyStatus = s
      }, 3000)

      // 2. Download tail 1KB of log file
      const tailUrl = clusterAPI.getLogFileUrl(
        this.clusterDetail.cluster_name,
        component.id,
        filename,
        'download',
        1024
      )
      const link = document.createElement('a')
      link.href = tailUrl
      const baseName = filename.replace(/\.[^/.]+$/, '')
      const ext = filename.includes('.') ? filename.substring(filename.lastIndexOf('.')) : '.log'
      link.download = `${baseName}_for_ai${ext}`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)

      // 3. Show toast notification
      this.showToast('分析提示词已复制至剪贴板，粘贴到对话框即可使用')

      // 4. Emit event to open AI panel
      this.$emit('openAIAnalysis')
    }
  }
}
</script>

<style scoped>
.cluster-detail-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 20px;
}

.modal-content {
  background: white;
  border-radius: 12px;
  max-width: 1400px;
  width: 100%;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 24px;
  border-bottom: 1px solid #e5e7eb;
}

.modal-header h2 {
  margin: 0;
  font-size: 24px;
  color: #1f2937;
}

.close-btn {
  background: none;
  border: none;
  font-size: 32px;
  color: #6b7280;
  cursor: pointer;
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  transition: all 0.2s;
}

.close-btn:hover {
  background: #f3f4f6;
  color: #1f2937;
}

.modal-body {
  padding: 24px;
  overflow-y: auto;
}

.cluster-info-section {
  background: #f9fafb;
  padding: 16px;
  border-radius: 8px;
  margin-bottom: 24px;
}

.info-row {
  display: flex;
  padding: 8px 0;
  border-bottom: 1px solid #e5e7eb;
}

.info-row:last-child {
  border-bottom: none;
}

.label {
  font-weight: 600;
  color: #6b7280;
  width: 150px;
  flex-shrink: 0;
}

.value {
  color: #1f2937;
  flex: 1;
}

.value.link {
  color: #3b82f6;
  text-decoration: none;
}

.value.link:hover {
  text-decoration: underline;
}

.components-section h3 {
  margin: 0 0 16px 0;
  font-size: 18px;
  color: #1f2937;
}

.table-scroll-wrapper {
  overflow-x: auto;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  max-width: 100%;
}

/* Custom scrollbar styling */
.table-scroll-wrapper::-webkit-scrollbar {
  height: 10px;
}

.table-scroll-wrapper::-webkit-scrollbar-track {
  background: #f1f5f9;
  border-radius: 0 0 8px 8px;
}

.table-scroll-wrapper::-webkit-scrollbar-thumb {
  background: #94a3b8;
  border-radius: 5px;
}

.table-scroll-wrapper::-webkit-scrollbar-thumb:hover {
  background: #64748b;
}

.table-container {
  min-width: fit-content;
}

.components-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
  min-width: 1200px;
}

.components-table thead {
  background: #f9fafb;
}

.components-table th {
  padding: 12px;
  text-align: left;
  font-weight: 600;
  color: #6b7280;
  border-bottom: 2px solid #e5e7eb;
  white-space: nowrap;
  position: sticky;
  top: 0;
  background: #f9fafb;
}

.components-table td {
  padding: 12px;
  border-bottom: 1px solid #e5e7eb;
}

.components-table tbody tr:hover {
  background: #f9fafb;
}

.id-cell {
  font-family: monospace;
  font-size: 12px;
  color: #6b7280;
  white-space: nowrap;
}

.host-cell {
  font-family: monospace;
  font-weight: 500;
  color: #1f2937;
  white-space: nowrap;
}

.path-cell {
  max-width: 280px;
  min-width: 150px;
}

.path-scroll {
  font-family: monospace;
  font-size: 11px;
  color: #6b7280;
  white-space: nowrap;
  overflow-x: auto;
  overflow-y: hidden;
  max-width: 280px;
  padding-bottom: 4px;
}

.path-scroll::-webkit-scrollbar {
  height: 4px;
}

.path-scroll::-webkit-scrollbar-track {
  background: #f1f5f9;
  border-radius: 2px;
}

.path-scroll::-webkit-scrollbar-thumb {
  background: #cbd5e1;
  border-radius: 2px;
}

.path-scroll::-webkit-scrollbar-thumb:hover {
  background: #94a3b8;
}

.log-cell {
  min-width: 280px;
  white-space: nowrap;
}

.log-files {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.log-file-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.log-filename {
  font-family: monospace;
  font-size: 11px;
  color: #6b7280;
  width: 130px;
  min-width: 130px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.log-actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}

.log-btn {
  padding: 2px 8px;
  border: none;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
  min-width: 68px;
  text-align: center;
  box-sizing: border-box;
}

.log-btn-view {
  background: #dbeafe;
  color: #1e40af;
}

.log-btn-view:hover {
  background: #bfdbfe;
  color: #1e3a8a;
}

.log-btn-download {
  background: #d1fae5;
  color: #065f46;
}

.log-btn-download:hover {
  background: #a7f3d0;
  color: #064e3b;
}

.log-btn-ai {
  background: #ede9fe;
  color: #6d28d9;
}

.log-btn-ai:hover {
  background: #ddd6fe;
  color: #5b21b6;
}

.log-btn-ai-copied {
  background: #d1fae5;
  color: #065f46;
}

.no-logs {
  color: #d1d5db;
}

.role-badge {
  display: inline-block;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
}

.role-tikv {
  background: #dbeafe;
  color: #1e40af;
}

.role-pd {
  background: #fce7f3;
  color: #be185d;
}

.role-tidb {
  background: #d1fae5;
  color: #065f46;
}

.role-prometheus {
  background: #fed7aa;
  color: #92400e;
}

.role-grafana {
  background: #e0e7ff;
  color: #3730a3;
}

.role-alertmanager {
  background: #fecaca;
  color: #991b1b;
}

.status-badge {
  display: inline-block;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
}

.status-up {
  background: #d1fae5;
  color: #065f46;
}

.status-down {
  background: #fee2e2;
  color: #991b1b;
}

.status-unknown {
  background: #f3f4f6;
  color: #6b7280;
}

/* Toast notification */
.ai-toast {
  position: fixed;
  top: 24px;
  left: 50%;
  transform: translateX(-50%);
  background: #1e1e2e;
  color: #a5f3fc;
  padding: 12px 24px;
  border-radius: 10px;
  font-size: 14px;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 10px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.35);
  z-index: 9999;
  border: 1px solid #7c3aed;
}

.ai-toast svg {
  color: #34d399;
  flex-shrink: 0;
}

.toast-enter-active {
  transition: all 0.3s ease-out;
}

.toast-leave-active {
  transition: all 0.3s ease-in;
}

.toast-enter-from {
  opacity: 0;
  transform: translateX(-50%) translateY(-20px);
}

.toast-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(-20px);
}
</style>
