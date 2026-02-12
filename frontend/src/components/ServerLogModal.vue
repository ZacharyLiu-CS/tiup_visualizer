<template>
  <div class="server-log-modal" v-if="visible" @click.self="close">
    <div class="modal-content">
      <div class="modal-header">
        <h2>Backend Service Logs</h2>
        <button class="close-btn" @click="close">&times;</button>
      </div>

      <div class="modal-body">
        <div class="loading" v-if="loading">
          <div class="spinner"></div>
          <span>Loading log files...</span>
        </div>

        <div class="error" v-else-if="error">
          <p>{{ error }}</p>
          <button class="retry-btn" @click="fetchLogs">Retry</button>
        </div>

        <div class="empty" v-else-if="!logFiles.length">
          <p>No log files found.</p>
        </div>

        <table class="log-table" v-else>
          <thead>
            <tr>
              <th>Filename</th>
              <th>Size</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="file in logFiles" :key="file.filename">
              <td class="filename-cell">{{ file.filename }}</td>
              <td class="size-cell">{{ formatSize(file.size) }}</td>
              <td class="actions-cell">
                <button class="log-btn log-btn-view" @click="viewLog(file.filename)" title="View log in browser">View</button>
                <button class="log-btn log-btn-download" @click="downloadLog(file.filename)" title="Download log file">Download</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script>
import { serverLogAPI } from '../services/api'

export default {
  name: 'ServerLogModal',
  props: {
    visible: {
      type: Boolean,
      default: false
    }
  },
  emits: ['close'],
  data() {
    return {
      logFiles: [],
      loading: false,
      error: null
    }
  },
  watch: {
    visible(val) {
      if (val) {
        this.fetchLogs()
      }
    }
  },
  methods: {
    close() {
      this.$emit('close')
    },
    async fetchLogs() {
      this.loading = true
      this.error = null
      try {
        const response = await serverLogAPI.listLogs()
        this.logFiles = response.data.files
      } catch (e) {
        this.error = e.message || 'Failed to load log files'
      } finally {
        this.loading = false
      }
    },
    viewLog(filename) {
      const url = serverLogAPI.getLogUrl(filename, 'view')
      window.open(url, '_blank')
    },
    downloadLog(filename) {
      const url = serverLogAPI.getLogUrl(filename, 'download')
      const link = document.createElement('a')
      link.href = url
      link.download = filename
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
    },
    formatSize(bytes) {
      if (bytes < 1024) return bytes + ' B'
      if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
      return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
    }
  }
}
</script>

<style scoped>
.server-log-modal {
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
  max-width: 600px;
  width: 100%;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-bottom: 1px solid #e5e7eb;
}

.modal-header h2 {
  margin: 0;
  font-size: 20px;
  color: #1f2937;
}

.close-btn {
  background: none;
  border: none;
  font-size: 28px;
  color: #6b7280;
  cursor: pointer;
  width: 36px;
  height: 36px;
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

.loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 32px 0;
  color: #6b7280;
}

.spinner {
  width: 24px;
  height: 24px;
  border: 3px solid #e5e7eb;
  border-top-color: #3b82f6;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.error {
  text-align: center;
  color: #dc2626;
  padding: 20px 0;
}

.retry-btn {
  margin-top: 12px;
  padding: 6px 16px;
  background: #3b82f6;
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-weight: 600;
}

.empty {
  text-align: center;
  color: #9ca3af;
  padding: 32px 0;
}

.log-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.log-table thead {
  background: #f9fafb;
}

.log-table th {
  padding: 10px 12px;
  text-align: left;
  font-weight: 600;
  color: #6b7280;
  border-bottom: 2px solid #e5e7eb;
  white-space: nowrap;
}

.log-table td {
  padding: 10px 12px;
  border-bottom: 1px solid #e5e7eb;
}

.log-table tbody tr:hover {
  background: #f9fafb;
}

.filename-cell {
  font-family: monospace;
  font-size: 13px;
  color: #1f2937;
  font-weight: 500;
}

.size-cell {
  font-family: monospace;
  font-size: 12px;
  color: #6b7280;
  white-space: nowrap;
}

.actions-cell {
  white-space: nowrap;
}

.log-btn {
  padding: 3px 10px;
  border: none;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
  width: 68px;
  text-align: center;
  box-sizing: border-box;
  display: inline-block;
}

.log-btn + .log-btn {
  margin-left: 4px;
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
</style>
