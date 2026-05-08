<template>
  <teleport to="body">
    <transition name="slide-right">
      <div v-if="visible" class="ccfg-overlay" @click.self="handleClose">
        <div class="ccfg-panel">
          <!-- Title bar -->
          <div class="ccfg-titlebar">
            <div class="ccfg-title">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="18" height="18">
                <circle cx="12" cy="12" r="3"/>
                <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/>
              </svg>
              {{ clusterName }} - Config
            </div>
            <button class="ctrl-btn close-btn" @click="handleClose" title="Close">&times;</button>
          </div>

          <!-- Loading state -->
          <div v-if="loading" class="ccfg-loading">
            <div class="spinner"></div>
            <span>Loading config...</span>
          </div>

          <!-- Error state -->
          <div v-else-if="loadError" class="ccfg-error">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="20" height="20">
              <circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/>
            </svg>
            <span>Failed to load config: {{ loadError }}</span>
            <button class="btn-retry" @click="loadConfig">Retry</button>
          </div>

          <!-- Editor -->
          <div v-else class="ccfg-editor-area">
            <div class="ccfg-editor-header">
              <span class="ccfg-editor-label">meta.yaml</span>
              <span v-if="hasChanges" class="ccfg-modified-badge">Modified</span>
            </div>
            <textarea
              ref="editor"
              v-model="editedConfig"
              class="ccfg-textarea"
              spellcheck="false"
              :disabled="saving"
            ></textarea>
          </div>

          <!-- Action bar -->
          <div v-if="!loading && !loadError" class="ccfg-actions">
            <button
              class="ccfg-btn ccfg-btn-export"
              @click="handleExportTemplate"
              :disabled="saving || reloading"
              title="Export current config as a cluster creation template"
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                <path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/>
                <polyline points="7 10 12 15 17 10"/>
                <line x1="12" y1="15" x2="12" y2="3"/>
              </svg>
              Export as Template
            </button>
            <button
              class="ccfg-btn ccfg-btn-save"
              @click="handleSaveAndReload"
              :disabled="saving || reloading"
              title="Save config and reload cluster"
            >
              <svg v-if="saving || reloading" class="spin-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                <path d="M21 12a9 9 0 11-6.219-8.56"/>
              </svg>
              <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                <polyline points="20 6 9 17 4 12"/>
              </svg>
              {{ saving ? 'Saving...' : reloading ? 'Reloading...' : 'Save & Reload' }}
            </button>
            <button
              class="ccfg-btn ccfg-btn-discard"
              @click="handleDiscard"
              :disabled="!hasChanges || saving || reloading"
              title="Discard all changes and revert to original"
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                <polyline points="1 4 1 10 7 10"/>
                <path d="M3.51 15a9 9 0 1 0 .49-3.51L1 10"/>
              </svg>
              Discard Changes
            </button>
          </div>

          <!-- Status messages -->
          <div v-if="statusMessage" class="ccfg-status" :class="statusType">
            {{ statusMessage }}
          </div>
        </div>
      </div>
    </transition>
  </teleport>
</template>

<script>
import { clusterAPI } from '../services/api'

export default {
  name: 'ClusterConfigModal',
  props: {
    visible: {
      type: Boolean,
      default: false
    },
    clusterName: {
      type: String,
      default: ''
    }
  },
  emits: ['close', 'refresh'],
  data() {
    return {
      loading: false,
      loadError: '',
      originalConfig: '',
      editedConfig: '',
      saving: false,
      reloading: false,
      statusMessage: '',
      statusType: 'info', // 'info', 'success', 'error'
    }
  },
  computed: {
    hasChanges() {
      return this.editedConfig !== this.originalConfig
    }
  },
  watch: {
    visible(val) {
      if (val && this.clusterName) {
        this.loadConfig()
      }
      if (!val) {
        this.statusMessage = ''
        this.originalConfig = ''
        this.editedConfig = ''
      }
    }
  },
  methods: {
    async loadConfig() {
      this.loading = true
      this.loadError = ''
      this.statusMessage = ''
      try {
        const resp = await clusterAPI.getConfig(this.clusterName)
        this.originalConfig = resp.data.config || ''
        this.editedConfig = this.originalConfig
      } catch (e) {
        this.loadError = e.response?.data?.detail || e.message || 'Failed to load config'
      } finally {
        this.loading = false
      }
    },
    async handleSaveAndReload() {
      if (!this.hasChanges) {
        this.showStatus('No changes to save', 'info')
        return
      }

      // Confirm before save & reload
      if (!confirm(`Save config and reload cluster "${this.clusterName}"?\nThis will restart the cluster services to apply the new configuration.`)) {
        return
      }

      this.saving = true
      this.statusMessage = ''
      try {
        // Step 1: Save config
        await clusterAPI.saveConfig(this.clusterName, this.editedConfig)
        this.originalConfig = this.editedConfig
        this.showStatus('Config saved. Reloading cluster...', 'info')

        // Step 2: Reload cluster
        this.saving = false
        this.reloading = true
        const reloadResp = await clusterAPI.reloadCluster(this.clusterName)
        const output = reloadResp.data?.output || ''
        this.showStatus(`Cluster reloaded successfully.\n${output.substring(0, 200)}`, 'success')
        this.$emit('refresh')
      } catch (e) {
        const errMsg = e.response?.data?.detail || e.message || 'Operation failed'
        this.showStatus(errMsg, 'error')
      } finally {
        this.saving = false
        this.reloading = false
      }
    },
    async handleExportTemplate() {
      this.statusMessage = ''
      try {
        const resp = await clusterAPI.exportTemplate(this.clusterName)
        this.showStatus('Template exported! You can find it in the Create Cluster > History tab.', 'success')
      } catch (e) {
        const errMsg = e.response?.data?.detail || e.message || 'Export failed'
        this.showStatus(errMsg, 'error')
      }
    },
    handleDiscard() {
      if (!this.hasChanges) return
      this.editedConfig = this.originalConfig
      this.showStatus('Changes discarded', 'info')
    },
    handleClose() {
      if (this.saving || this.reloading) return
      this.$emit('close')
    },
    showStatus(message, type) {
      this.statusMessage = message
      this.statusType = type
    }
  }
}
</script>

<style scoped>
.ccfg-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  background: rgba(0, 0, 0, 0.4);
  z-index: 2000;
  display: flex;
  justify-content: flex-end;
}

.ccfg-panel {
  width: 58vw;
  min-width: 500px;
  max-width: 900px;
  height: 100vh;
  background: #fff;
  display: flex;
  flex-direction: column;
  box-shadow: -4px 0 20px rgba(0, 0, 0, 0.15);
}

.ccfg-titlebar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 20px;
  border-bottom: 1px solid #e5e7eb;
  background: #f9fafb;
}

.ccfg-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 700;
  color: #1f2937;
}

.ctrl-btn {
  background: none;
  border: none;
  cursor: pointer;
  font-size: 22px;
  line-height: 1;
  padding: 4px 8px;
  border-radius: 4px;
  color: #6b7280;
  transition: all 0.15s;
}

.ctrl-btn:hover {
  background: #e5e7eb;
  color: #374151;
}

.ccfg-loading {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: #6b7280;
  font-size: 14px;
}

.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid #e5e7eb;
  border-top-color: #8b5cf6;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.ccfg-error {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: #dc2626;
  font-size: 14px;
  text-align: center;
  padding: 20px;
}

.btn-retry {
  margin-top: 8px;
  padding: 6px 16px;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  background: white;
  color: #374151;
  cursor: pointer;
  font-size: 13px;
  font-weight: 600;
}

.btn-retry:hover {
  background: #f3f4f6;
}

.ccfg-editor-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.ccfg-editor-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: #f9fafb;
  border-bottom: 1px solid #e5e7eb;
}

.ccfg-editor-label {
  font-size: 12px;
  font-weight: 600;
  color: #6b7280;
  font-family: monospace;
}

.ccfg-modified-badge {
  font-size: 10px;
  font-weight: 700;
  color: #d97706;
  background: #fef3c7;
  padding: 2px 8px;
  border-radius: 10px;
}

.ccfg-textarea {
  flex: 1;
  width: 100%;
  padding: 12px 16px;
  border: none;
  outline: none;
  resize: none;
  font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', 'Consolas', monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #1f2937;
  background: #fafafa;
  tab-size: 2;
}

.ccfg-textarea:focus {
  background: #fff;
}

.ccfg-actions {
  display: flex;
  gap: 8px;
  padding: 12px 16px;
  border-top: 1px solid #e5e7eb;
  background: #f9fafb;
  flex-wrap: wrap;
}

.ccfg-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  font-size: 13px;
  font-weight: 600;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
  background: white;
  color: #374151;
}

.ccfg-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.ccfg-btn-export {
  color: #0891b2;
  border-color: #a5f3fc;
  background: #ecfeff;
}

.ccfg-btn-export:hover:not(:disabled) {
  background: #cffafe;
  border-color: #67e8f9;
}

.ccfg-btn-save {
  color: #059669;
  border-color: #a7f3d0;
  background: #ecfdf5;
}

.ccfg-btn-save:hover:not(:disabled) {
  background: #d1fae5;
  border-color: #6ee7b7;
}

.ccfg-btn-discard {
  color: #dc2626;
  border-color: #fecaca;
  background: #fef2f2;
}

.ccfg-btn-discard:hover:not(:disabled) {
  background: #fee2e2;
  border-color: #fca5a5;
}

.ccfg-status {
  padding: 10px 16px;
  font-size: 13px;
  white-space: pre-wrap;
  word-break: break-word;
  border-top: 1px solid #e5e7eb;
  max-height: 120px;
  overflow-y: auto;
}

.ccfg-status.info {
  color: #2563eb;
  background: #eff6ff;
}

.ccfg-status.success {
  color: #059669;
  background: #ecfdf5;
}

.ccfg-status.error {
  color: #dc2626;
  background: #fef2f2;
}

.spin-icon {
  animation: spin 0.8s linear infinite;
}

/* Slide-right transition */
.slide-right-enter-active,
.slide-right-leave-active {
  transition: transform 0.3s ease;
}

.slide-right-enter-from,
.slide-right-leave-to {
  transform: translateX(100%);
}
</style>
