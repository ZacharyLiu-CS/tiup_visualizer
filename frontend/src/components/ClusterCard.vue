<template>
  <div class="cluster-card"
       :class="{ 
         'selected': isSelected, 
         'highlighted': isHighlighted 
       }">
    <span 
      v-if="index" 
      class="cluster-index-badge"
    >{{ index }}</span>
    <div class="cluster-top" @click="handleConnect" title="Click to show host connections">
      <div class="cluster-icon">
        <svg width="48" height="48" viewBox="0 0 48 48" fill="none">
          <circle cx="24" cy="12" r="6" stroke="currentColor" stroke-width="2"/>
          <circle cx="12" cy="32" r="6" stroke="currentColor" stroke-width="2"/>
          <circle cx="36" cy="32" r="6" stroke="currentColor" stroke-width="2"/>
          <line x1="24" y1="18" x2="16" y2="26" stroke="currentColor" stroke-width="2"/>
          <line x1="24" y1="18" x2="32" y2="26" stroke="currentColor" stroke-width="2"/>
        </svg>
      </div>
      <div class="connect-hint" v-if="!isSelected">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M8 6h13M8 12h13M8 18h13M3 6h.01M3 12h.01M3 18h.01"/>
        </svg>
      </div>
      <div class="connect-hint active" v-else>
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>
          <polyline points="15 3 21 3 21 9"/>
          <line x1="10" y1="14" x2="21" y2="3"/>
        </svg>
      </div>
    </div>
    <div class="cluster-bottom" @click="handleDetail">
      <div class="cluster-info">
        <div class="cluster-name">{{ cluster.name }}</div>
        <div class="cluster-version">{{ cluster.version }}</div>
        <div class="cluster-user">User: {{ cluster.user }}</div>
      </div>
      <div class="detail-link">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/>
          <line x1="12" y1="16" x2="12" y2="12"/>
          <line x1="12" y1="8" x2="12.01" y2="8"/>
        </svg>
        <span>Details</span>
      </div>
    </div>
    <div class="cluster-actions">
      <button class="action-btn action-start" @click="handleStart" :disabled="operating !== ''" title="Start Cluster">
        <svg v-if="operating === 'start'" class="spin-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 12a9 9 0 11-6.219-8.56" /></svg>
        <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3" /></svg>
        Start
      </button>
      <button class="action-btn action-stop" @click="handleStop" :disabled="operating !== ''" title="Stop Cluster">
        <svg v-if="operating === 'stop'" class="spin-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 12a9 9 0 11-6.219-8.56" /></svg>
        <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="6" y="4" width="4" height="16" /><rect x="14" y="4" width="4" height="16" /></svg>
        Stop
      </button>
      <div class="ddl-dropdown" ref="ddlDropdown">
        <button class="action-btn action-ddl" @click="toggleDDL" :disabled="operating !== ''" title="DDL Operations">
          <svg v-if="operating === 'clean' || operating === 'destroy'" class="spin-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 12a9 9 0 11-6.219-8.56" /></svg>
          <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6" /><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2" /></svg>
          DDL
          <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="6 9 12 15 18 9" /></svg>
        </button>
        <transition name="dropdown">
          <div v-show="showDDL" class="ddl-menu">
            <div class="ddl-item" @click="handleClean">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 21H3l9-18 9 18z" /><line x1="12" y1="9" x2="12" y2="13" /><line x1="12" y1="17" x2="12.01" y2="17" /></svg>
              Clean (--all)
            </div>
            <div class="ddl-item ddl-danger" @click="handleDestroy">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6" /><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2" /><line x1="10" y1="11" x2="10" y2="17" /><line x1="14" y1="11" x2="14" y2="17" /></svg>
              Destroy
            </div>
          </div>
        </transition>
      </div>
    </div>
    <div class="status-bar" :class="statusClass"></div>
    <!-- Custom Confirm Dialog -->
    <ConfirmDialog
      :visible="confirmDialog.visible"
      :title="confirmDialog.title"
      :message="confirmDialog.message"
      :type="confirmDialog.type"
      :confirmText="confirmDialog.confirmText"
      :cancelText="confirmDialog.cancelText"
      @confirm="onConfirmDialog"
      @cancel="onCancelDialog"
      @update:visible="confirmDialog.visible = $event"
    />
    <!-- Custom Alert Dialog -->
    <ConfirmDialog
      :visible="alertDialog.visible"
      :title="alertDialog.title"
      :message="alertDialog.message"
      :type="alertDialog.type"
      :singleButton="true"
      :cancelText="'OK'"
      @confirm="alertDialog.visible = false"
      @update:visible="alertDialog.visible = $event"
    />
  </div>
</template>

<script>
import { clusterAPI } from '../services/api'
import ConfirmDialog from './ConfirmDialog.vue'

export default {
  name: 'ClusterCard',
  components: {
    ConfirmDialog
  },
  props: {
    cluster: {
      type: Object,
      required: true
    },
    index: {
      type: Number,
      default: null
    },
    isSelected: {
      type: Boolean,
      default: false
    },
    isHighlighted: {
      type: Boolean,
      default: false
    }
  },
  emits: ['connect', 'detail', 'refresh'],
  data() {
    return {
      operating: '',  // 'start', 'stop', 'clean', 'destroy', or ''
      showDDL: false,
      errorMsg: '',
      confirmDialog: {
        visible: false,
        title: '',
        message: '',
        type: 'default', // 'default' | 'warning' | 'danger'
        confirmText: 'Confirm',
        cancelText: 'Cancel',
        action: '', // 待确认的操作: 'start' | 'stop' | 'clean' | 'destroy'
      },
      alertDialog: {
        visible: false,
        title: '',
        message: '',
        type: 'default',
      }
    }
  },
  computed: {
    statusClass() {
      const status = this.cluster.status
      if (status === 'healthy') return 'status-healthy'
      if (status === 'partial') return 'status-partial'
      if (status === 'unhealthy') return 'status-unknown'
      return 'status-unhealthy'
    }
  },
  mounted() {
    document.addEventListener('click', this.handleClickOutside)
  },
  beforeUnmount() {
    document.removeEventListener('click', this.handleClickOutside)
  },
  methods: {
    handleConnect() {
      this.$emit('connect', this.cluster.name)
    },
    handleDetail() {
      this.$emit('detail', this.cluster.name)
    },
    toggleDDL() {
      this.showDDL = !this.showDDL
    },
    handleClickOutside(e) {
      if (this.$refs.ddlDropdown && !this.$refs.ddlDropdown.contains(e.target)) {
        this.showDDL = false
      }
    },
    // ---- 操作入口：显示确认弹窗 ----
    handleStart() {
      if (this.operating) return
      console.log('[ClusterCard] handleStart called for:', this.cluster.name)
      this.confirmDialog = {
        visible: true,
        title: 'Start Cluster',
        message: `Are you sure to start cluster "${this.cluster.name}"?`,
        type: 'default',
        confirmText: 'Start',
        cancelText: 'Cancel',
        action: 'start',
      }
    },
    handleStop() {
      if (this.operating) return
      console.log('[ClusterCard] handleStop called for:', this.cluster.name)
      this.confirmDialog = {
        visible: true,
        title: 'Stop Cluster',
        message: `Are you sure to stop cluster "${this.cluster.name}"?`,
        type: 'warning',
        confirmText: 'Stop',
        cancelText: 'Cancel',
        action: 'stop',
      }
    },
    handleClean() {
      this.showDDL = false
      if (this.operating) return
      console.log('[ClusterCard] handleClean called for:', this.cluster.name)
      this.confirmDialog = {
        visible: true,
        title: '⚠️ Clean Cluster Data',
        message: `This will clean ALL data of cluster "${this.cluster.name}" (tiup cluster clean --all).\n\nAre you absolutely sure?`,
        type: 'warning',
        confirmText: 'Clean --all',
        cancelText: 'Cancel',
        action: 'clean',
      }
    },
    handleDestroy() {
      this.showDDL = false
      if (this.operating) return
      console.log('[ClusterCard] handleDestroy called for:', this.cluster.name)
      // 第一步确认
      this.confirmDialog = {
        visible: true,
        title: '🚨 DESTROY Cluster',
        message: `This will DESTROY cluster "${this.cluster.name}" and DELETE ALL DATA!\n\nThis action CANNOT be undone.`,
        type: 'danger',
        confirmText: 'I understand, proceed',
        cancelText: 'Cancel',
        action: 'destroy-confirm',
      }
    },
    // ---- 弹窗回调 ----
    async onConfirmDialog() {
      const action = this.confirmDialog.action
      console.log('[ClusterCard] onConfirmDialog called, action:', action)
      this.confirmDialog.visible = false

      if (action === 'destroy-confirm') {
        // 第二次确认
        console.log('[ClusterCard] Showing second confirmation dialog')
        this.$nextTick(() => {
          this.confirmDialog = {
            visible: true,
            title: 'FINAL WARNING',
            message: `Cluster "${this.cluster.name}" will be permanently destroyed!\n\nClick "Destroy" only if you are absolutely certain.`,
            type: 'danger',
            confirmText: 'Destroy',
            cancelText: 'Cancel',
            action: 'destroy',
          }
        })
        return
      }

      // 执行实际操作
      const actionMap = {
        start: { api: 'startCluster', running: 'start', successMsg: 'started' },
        stop:  { api: 'stopCluster',  running: 'stop',  successMsg: 'stopped' },
        clean: { api: 'cleanCluster', running: 'clean', successMsg: 'cleaned' },
        destroy: { api: 'destroyCluster', running: 'destroy', successMsg: 'destroyed' },
      }
      const info = actionMap[action]
      if (!info) return

      this.operating = info.running
      this.errorMsg = ''
      try {
        const response = await clusterAPI[info.api](this.cluster.name)
        const output = response.data?.output || ''
        // 构建消息，包含命令输出
        let message = `Cluster "${this.cluster.name}" ${info.successMsg} successfully!`
        if (output.trim()) {
          message += `\n\n--- Output ---\n${output}`
        }
        this.showAlert('Success', message, 'default')
        this.$emit('refresh')
      } catch (e) {
        const errMsg = e.response?.data?.detail || e.message || `${info.running} failed`
        this.showAlert('Error', `${info.running} cluster failed: ${errMsg}`, 'danger')
      } finally {
        this.operating = ''
      }
    },
    onCancelDialog() {
      this.confirmDialog.visible = false
    },
    // ---- Alert 弹窗 ----
    showAlert(title, message, type = 'default') {
      this.alertDialog = {
        visible: true,
        title,
        message,
        type,
      }
    },
  }
}
</script>

<style scoped>
.cluster-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  background: white;
  border: 2px solid #e5e7eb;
  border-radius: 8px;
  transition: all 0.3s ease;
  min-width: 160px;
  overflow: hidden;
  position: relative;
}

.cluster-index-badge {
  position: absolute;
  top: 6px;
  left: 6px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  color: #6b7280;
  font-size: 11px;
  font-weight: 700;
  line-height: 1;
  z-index: 1;
  background: transparent;
  border: 2px solid #d1d5db;
}

.cluster-top {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100%;
  padding: 16px 16px 8px;
  cursor: pointer;
  border-bottom: 1px dashed #e5e7eb;
  transition: background 0.2s;
  position: relative;
}

.cluster-top:hover {
  background: #f5f3ff;
}

.connect-hint {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 10px;
  color: #9ca3af;
  margin-top: 2px;
  transition: color 0.2s;
}

.cluster-top:hover .connect-hint {
  color: #8b5cf6;
}

.connect-hint.active {
  color: #8b5cf6;
}

.cluster-bottom {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100%;
  padding: 8px 16px 0;
  cursor: pointer;
  transition: background 0.2s;
}

.cluster-bottom:hover {
  background: #eff6ff;
}

.cluster-card:hover {
  border-color: #8b5cf6;
  box-shadow: 0 4px 12px rgba(139, 92, 246, 0.2);
  transform: translateY(-2px);
}

.cluster-card.selected {
  border-color: #8b5cf6;
  background: #f5f3ff;
}

.cluster-card.highlighted {
  border-color: #10b981;
  background: #f0fdf4;
}

.cluster-icon {
  color: #6b7280;
  margin-bottom: 4px;
}

.cluster-card.selected .cluster-icon,
.cluster-card.highlighted .cluster-icon {
  color: #8b5cf6;
}

.cluster-info {
  text-align: center;
  width: 100%;
}

.cluster-name {
  font-weight: 700;
  font-size: 14px;
  color: #1f2937;
  margin-bottom: 4px;
  word-break: break-all;
}

.cluster-version {
  font-size: 12px;
  color: #8b5cf6;
  margin-bottom: 2px;
  font-weight: 600;
}

.cluster-user {
  font-size: 11px;
  color: #6b7280;
  padding: 2px 6px;
  background: #f3f4f6;
  border-radius: 4px;
  display: inline-block;
}

.detail-link {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 6px;
  padding: 3px 10px;
  font-size: 11px;
  color: #3b82f6;
  cursor: pointer;
  border-radius: 4px;
  transition: all 0.2s;
  font-weight: 600;
}

.detail-link:hover {
  background: #dbeafe;
  color: #2563eb;
}

.status-bar {
  width: calc(100% + 4px);
  height: 6px;
  margin-top: 8px;
  margin-left: -2px;
  margin-right: -2px;
  border-radius: 0 0 6px 6px;
}

.status-healthy {
  background: #22c55e;
}

.status-partial {
  background: #eab308;
}

.status-unhealthy {
  background: #ef4444;
}

.status-unknown {
  background: #9ca3af;
}

/* Cluster Action Buttons */
.cluster-actions {
  display: flex;
  gap: 4px;
  padding: 8px 8px 4px;
  width: 100%;
  flex-wrap: wrap;
  justify-content: center;
  position: relative;
  z-index: 10;
  pointer-events: auto;
}

.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  font-size: 11px;
  font-weight: 600;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.15s;
  background: white;
  color: #374151;
  line-height: 1.2;
}

.action-btn:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.action-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.action-start {
  color: #16a34a;
  border-color: #bbf7d0;
  background: #f0fdf4;
}
.action-start:hover:not(:disabled) {
  background: #dcfce7;
  border-color: #86efac;
}

.action-stop {
  color: #dc2626;
  border-color: #fecaca;
  background: #fef2f2;
}
.action-stop:hover:not(:disabled) {
  background: #fee2e2;
  border-color: #fca5a5;
}

.action-ddl {
  color: #d97706;
  border-color: #fde68a;
  background: #fffbeb;
}
.action-ddl:hover:not(:disabled) {
  background: #fef3c7;
  border-color: #fbbf24;
}

/* DDL Dropdown */
.ddl-dropdown {
  position: relative;
  display: inline-flex;
}

.ddl-menu {
  position: absolute;
  bottom: calc(100% + 4px);
  left: 50%;
  transform: translateX(-50%);
  min-width: 140px;
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  padding: 4px;
  box-shadow: 0 4px 12px rgba(0,0,0,0.15);
  z-index: 100;
}

.ddl-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  font-size: 12px;
  font-weight: 500;
  color: #374151;
  border-radius: 4px;
  cursor: pointer;
  transition: background 0.15s;
  white-space: nowrap;
}

.ddl-item:hover {
  background: #f3f4f6;
}

.ddl-item.ddl-danger {
  color: #dc2626;
}

.ddl-item.ddl-danger:hover {
  background: #fef2f2;
}

/* Dropdown transition */
.dropdown-enter-active,
.dropdown-leave-active {
  transition: opacity 0.15s, transform 0.15s;
}
.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(4px);
}

/* Spin animation for loading icons */
.spin-icon {
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
