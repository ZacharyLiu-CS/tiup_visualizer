<template>
  <teleport to="body">
    <transition name="slide-right">
      <div v-if="visible" class="cc-overlay" @click.self="$emit('close')">
        <div class="cc-panel">
          <!-- Title bar -->
          <div class="cc-titlebar">
            <div class="cc-title">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="18" height="18">
                <circle cx="12" cy="12" r="10" /><line x1="12" y1="8" x2="12" y2="16" /><line x1="8" y1="12" x2="16" y2="12" />
              </svg>
              创建 TiKV 集群
            </div>
            <button class="ctrl-btn close-btn" @click="$emit('close')" title="Close">&times;</button>
          </div>

          <!-- Tab switch: Create / History -->
          <div class="cc-tabs">
            <button class="cc-tab-btn" :class="{ active: activeTab === 'create' }" @click="activeTab = 'create'">创建集群</button>
            <button class="cc-tab-btn" :class="{ active: activeTab === 'history' }" @click="switchToHistory">历史配置</button>
          </div>

          <!-- ===== CREATE TAB ===== -->
          <div v-if="activeTab === 'create'" class="cc-content">

            <!-- Cluster Name & Username -->
            <div class="form-row">
              <div class="form-group flex-1">
                <label>集群名称 <span class="required">*</span></label>
                <input v-model="form.clusterName" class="form-input" placeholder="例如: my-tikv-cluster"
                  :disabled="deploying" />
              </div>
              <div class="form-group flex-1">
                <label>部署用户名 <span class="required">*</span></label>
                <input v-model="form.username" class="form-input" placeholder="例如: root"
                  :disabled="deploying" />
              </div>
            </div>

            <!-- Version -->
            <div class="form-row">
              <div class="form-group flex-1">
                <label>TiUP 版本 <span class="required">*</span></label>
                <input v-model="form.version" class="form-input" placeholder="v8.1.0"
                  :disabled="deploying" />
              </div>
              <div class="form-group flex-1">
                <label>配置文件来源</label>
                <div class="radio-group">
                  <label class="radio-label">
                    <input type="radio" v-model="form.configSource" value="paste" :disabled="deploying" /> 粘贴内容
                  </label>
                  <label class="radio-label">
                    <input type="radio" v-model="form.configSource" value="upload" :disabled="deploying" /> 上传文件
                  </label>
                  <label class="radio-label">
                    <input type="radio" v-model="form.configSource" value="history" :disabled="deploying" /> 选择历史
                  </label>
                </div>
              </div>
            </div>

            <!-- Paste mode -->
            <div v-if="form.configSource === 'paste'" class="form-group">
              <label>粘贴 YAML 配置</label>
              <textarea v-model="form.configContent" class="form-textarea" placeholder="# 粘贴 tiup cluster topology 配置文件内容&#10;# 例如:&#10;global:&#10;  user: root&#10;  ...&#10;"
                rows="14" spellcheck="false" :disabled="deploying"></textarea>
            </div>

            <!-- Upload mode -->
            <div v-if="form.configSource === 'upload'" class="form-group">
              <label>上传 YAML 配置文件</label>
              <div class="file-upload-area" :class="{ dragover: isDragOver }"
                @dragover.prevent="isDragOver = true"
                @dragleave.prevent="isDragOver = false"
                @drop.prevent="handleFileDrop"
                @click="$refs.fileInput.click()">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="32" height="32">
                  <path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4" /><polyline points="17 8 12 3 7 8" /><line x1="12" y1="3" x2="12" y2="15" />
                </svg>
                <p v-if="!uploadedFileName">点击或拖拽上传 .yaml 文件</p>
                <p v-else class="uploaded-file-name">
                  <span class="file-icon">&#128196;</span> {{ uploadedFileName }}
                  <button class="clear-file-btn" @click.stop="clearUploadedFile" title="移除">&times;</button>
                </p>
                <input ref="fileInput" type="file" accept=".yaml,.yml,.toml" style="display:none"
                  @change="handleFileSelect" :disabled="deploying" />
              </div>
            </div>

            <!-- History select mode -->
            <div v-if="form.configSource === 'history'" class="form-group">
              <label>选择历史配置文件</label>
              <select v-model="selectedHistoryName" class="form-select" :disabled="deploying || historyConfigs.length === 0">
                <option value="">-- 选择历史配置 --</option>
                <option v-for="h in historyConfigs" :key="h.name" :value="h.name">{{ h.name }} ({{ formatSize(h.size) }} · {{ h.createdAt }})</option>
              </select>
              <div v-if="selectedHistoryName && historyConfigPreview" class="history-preview">
                <div class="preview-header">
                  <span>{{ selectedHistoryName }}.yaml</span>
                  <div class="preview-actions">
                    <button class="clear-btn" @click="viewFullConfig" title="查看完整配置">查看完整</button>
                    <button class="clear-btn" @click="downloadConfig(selectedHistoryName)" title="下载">下载</button>
                  </div>
                </div>
                <pre class="preview-content">{{ historyConfigPreview }}</pre>
              </div>
            </div>

            <!-- Command Preview -->
            <div class="form-group">
              <label>部署命令预览</label>
              <pre class="cmd-preview">{{ deployCommandPreview }}</pre>
            </div>

            <!-- Actions -->
            <div class="form-actions">
              <button class="run-btn" @click="handleDeploy" :disabled="deploying || !canDeploy">
                <svg v-if="deploying" class="spin-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                  <path d="M21 12a9 9 0 11-6.219-8.56" />
                </svg>
                <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="14" height="14">
                  <polygon points="5 3 19 12 5 21 5 3" />
                </svg>
                {{ deploying ? '部署中...' : '开始部署' }}
              </button>
              <button v-if="deploying" class="cancel-btn" @click="cancelDeploy">取消</button>
            </div>

            <!-- Deploy Output -->
            <div v-if="deployOutput" class="result-box">
              <div class="result-header">
                <span class="result-count">部署结果</span>
                <button class="copy-btn" @click="copyOutput" title="复制输出">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="13" height="13">
                    <rect x="9" y="9" width="13" height="13" rx="2" ry="2" /><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" />
                  </svg>
                  {{ outputCopied ? '已复制' : '复制' }}
                </button>
              </div>
              <pre class="result-output">{{ deployOutput }}</pre>
            </div>

            <!-- Error -->
            <div v-if="deployError" class="result-error">{{ deployError }}</div>
          </div>

          <!-- ===== HISTORY TAB ===== -->
          <div v-if="activeTab === 'history'" class="cc-content">
            <div class="view-toggle">
              <button class="clear-btn" @click="switchToCreate">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="13" height="13">
                  <polyline points="15 18 9 12 15 6" />
                </svg>
                返回创建
              </button>
              <button class="clear-btn" @click="loadHistory" :disabled="historyLoading">
                <svg :class="{ 'spin-icon': historyLoading }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="13" height="13">
                  <polyline points="23 4 23 10 17 10" /><path d="M20.49 15a9 9 0 11-2.12-9.36L23 10" />
                </svg>
                刷新
              </button>
            </div>

            <div v-if="historyLoading && historyConfigs.length === 0" class="empty-state">
              加载中...
            </div>

            <div v-else-if="historyConfigs.length === 0" class="empty-state">
              暂无历史配置文件。创建集群后会自动保存配置到历史记录。
            </div>

            <div v-for="item in historyConfigs" :key="item.name" class="history-card">
              <div class="history-card-header" @click="toggleHistoryDetail(item.name)">
                <span class="history-name">{{ item.name }}</span>
                <span class="history-meta">{{ formatSize(item.size) }} · {{ item.createdAt }}</span>
                <span class="history-expand">{{ expandedHistoryItem === item.name ? '▾' : '▸' }}</span>
              </div>

              <div v-if="expandedHistoryItem === item.name" class="history-card-detail">
                <pre v-if="viewingConfig === item.name && fullConfigContent" class="config-viewer">{{ fullConfigContent }}</pre>
                <p v-else class="config-placeholder">点击下方按钮查看或下载配置文件</p>
                <div class="history-card-actions">
                  <button class="run-btn" @click="useForCreate(item.name)" style="padding:6px 14px;font-size:12px;">
                    使用此配置创建
                  </button>
                  <button class="help-btn" @click="viewConfig(item.name)">查看</button>
                  <button class="clear-btn" @click="downloadConfig(item.name)">下载</button>
                  <button class="cancel-btn" @click="deleteConfig(item.name)">删除</button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </transition>
  </teleport>
</template>

<script>
import { clusterCreateAPI } from '../services/api'

export default {
  name: 'CreateClusterModal',
  props: {
    visible: {
      type: Boolean,
      default: false
    }
  },
  emits: ['close', 'deployed'],
  data() {
    return {
      activeTab: 'create',
      form: {
        clusterName: '',
        username: 'root',
        version: 'v8.1.0',
        configSource: 'paste',
        configContent: '',
      },
      // Upload
      uploadedFile: null,
      uploadedFileName: '',
      isDragOver: false,
      // History
      historyConfigs: [],
      historyLoading: false,
      selectedHistoryName: '',
      historyConfigPreview: '',
      expandedHistoryItem: null,
      viewingConfig: '',
      fullConfigContent: '',
      // Deploy state
      deploying: false,
      deployOutput: '',
      deployError: '',
      _abortController: null,
      outputCopied: false,
    }
  },
  computed: {
    canDeploy() {
      if (!this.form.clusterName.trim() || !this.form.username.trim() || !this.form.version.trim()) return false
      if (this.form.configSource === 'paste' && !this.form.configContent.trim()) return false
      if (this.form.configSource === 'upload' && !this.uploadedFile) return false
      if (this.form.configSource === 'history' && !this.selectedHistoryName) return false
      return true
    },
    deployCommandPreview() {
      const name = this.form.clusterName.trim()
      const ver = this.form.version.trim() || '<version>'
      const user = this.form.username.trim() || '<user>'
      let configRef = ''
      if (this.form.configSource === 'paste') configRef = '(粘贴内容)'
      else if (this.form.configSource === 'upload') configRef = this.uploadedFileName || '(上传文件)'
      else if (this.form.configSource === 'history') configRef = `${this.selectedHistoryName}.yaml`
      else configRef = '<config>'
      return `tiup cluster deploy ${name} ${ver} ${configRef} --user ${user} -y`
    }
  },
  watch: {
    visible(val) {
      if (val) {
        this.loadHistory()
      }
    },
    selectedHistoryName(name) {
      if (name) {
        this.loadHistoryPreview(name)
      } else {
        this.historyConfigPreview = ''
      }
    },
  },
  methods: {
    async loadHistory() {
      this.historyLoading = true
      try {
        const res = await clusterCreateAPI.listHistory()
        this.historyConfigs = res.data?.configs || []
      } catch (e) {
        // silent fail
      } finally {
        this.historyLoading = false
      }
    },
    async loadHistoryPreview(name) {
      try {
        const res = await clusterCreateAPI.getConfig(name)
        const content = res.data || ''
        // Show first ~30 lines as preview
        const lines = content.split('\n')
        this.historyConfigPreview = lines.slice(0, 30).join('\n') + (lines.length > 30 ? '\n... (共 ' + lines.length + ' 行)' : '')
      } catch (e) {
        this.historyConfigPreview = '加载失败'
      }
    },
    switchToHistory() {
      this.activeTab = 'history'
      this.loadHistory()
    },
    switchToCreate() {
      this.activeTab = 'create'
    },
    toggleHistoryDetail(name) {
      this.expandedHistoryItem = this.expandedHistoryItem === name ? null : name
      if (this.expandedHistoryItem === name) {
        this.viewingConfig = ''
        this.fullConfigContent = ''
      }
    },
    async viewConfig(name) {
      try {
        const res = await clusterCreateAPI.getConfig(name)
        this.viewingConfig = name
        this.fullConfigContent = res.data || ''
      } catch (e) {
        alert('加载失败: ' + (e.response?.data?.detail || e.message))
      }
    },
    viewFullConfig() {
      this.viewConfig(this.selectedHistoryName)
    },
    downloadConfig(name) {
      const url = clusterCreateAPI.getConfigUrl(name, 'download')
      const a = document.createElement('a')
      a.href = url
      a.download = `${name}.yaml`
      a.click()
    },
    useForCreate(name) {
      this.activeTab = 'create'
      this.form.configSource = 'history'
      this.$nextTick(() => {
        this.selectedHistoryName = name
      })
    },
    async deleteConfig(name) {
      if (!confirm(`确定删除配置 "${name}" 吗？`)) return
      try {
        await clusterCreateAPI.deleteConfig(name)
        await this.loadHistory()
        if (this.selectedHistoryName === name) {
          this.selectedHistoryName = ''
          this.historyConfigPreview = ''
        }
        if (this.expandedHistoryItem === name) {
          this.expandedHistoryItem = null
        }
      } catch (e) {
        alert('删除失败: ' + (e.response?.data?.detail || e.message))
      }
    },

    // File handling
    handleFileSelect(e) {
      const file = e.target.files[0]
      if (file) {
        this.setUploadedFile(file)
      }
    },
    handleFileDrop(e) {
      this.isDragOver = false
      const file = e.dataTransfer.files[0]
      if (file) {
        this.setUploadedFile(file)
      }
    },
    setUploadedFile(file) {
      this.uploadedFile = file
      this.uploadedFileName = file.name
    },
    clearUploadedFile() {
      this.uploadedFile = null
      this.uploadedFileName = ''
      if (this.$refs.fileInput) this.$refs.fileInput.value = ''
    },

    // Deploy
    async handleDeploy() {
      if (!this.canDeploy || this.deploying) return

      this.deploying = true
      this.deployOutput = ''
      this.deployError = ''

      // Use longer timeout for deploy (up to 30 min)
      try {
        let res
        if (this.form.configSource === 'upload' && this.uploadedFile) {
          // File upload
          const formData = new FormData()
          formData.append('cluster_name', this.form.clusterName.trim())
          formData.append('version', this.form.version.trim())
          formData.append('username', this.form.username.trim())
          formData.append('config_file', this.uploadedFile)
          res = await clusterCreateAPI.deployUpload(formData)
        } else {
          // JSON with pasted or history content
          let configContent = this.form.configContent
          if (this.form.configSource === 'history' && this.selectedHistoryName) {
            const cfgRes = await clusterCreateAPI.getConfig(this.selectedHistoryName)
            configContent = cfgRes.data || ''
          }

          res = await clusterCreateAPI.deploy({
            cluster_name: this.form.clusterName.trim(),
            version: this.form.version.trim(),
            username: this.form.username.trim(),
            config: configContent,
          })
        }

        this.deployOutput = res.data?.output || JSON.stringify(res.data, null, 2)

        // Refresh history after successful deployment
        await this.loadHistory()

        this.$emit('deployed', {
          clusterName: this.form.clusterName.trim(),
          output: this.deployOutput
        })
      } catch (e) {
        const msg = e.response?.data?.detail || e.message || '部署失败'
        // Extract meaningful message from response
        this.deployError = msg
        if (e.response?.data) {
          this.deployOutput = typeof e.response.data === 'string'
            ? e.response.data
            : JSON.stringify(e.response.data, null, 2)
        }
      } finally {
        this.deploying = false
      }
    },
    cancelDeploy() {
      this.deploying = false
    },
    async copyOutput() {
      try {
        await navigator.clipboard.writeText(this.deployOutput)
        this.outputCopied = true
        setTimeout(() => { this.outputCopied = false }, 2000)
      } catch {}
    },
    formatSize(bytes) {
      if (!bytes) return '0 B'
      const units = ['B', 'KB', 'MB']
      let i = 0
      while (bytes >= 1024 && i < units.length - 1) {
        bytes /= 1024
        i++
      }
      return bytes.toFixed(i > 0 ? 1 : 0) + ' ' + units[i]
    },
  }
}
</script>

<style scoped>
.cc-overlay {
  position: fixed;
  inset: 0;
  z-index: 1300;
  background: rgba(0, 0, 0, 0.3);
}

.cc-panel {
  position: fixed;
  top: 0;
  right: 0;
  width: 58vw;
  max-width: 100vw;
  height: 100vh;
  background: #1e1e2e;
  color: #cdd6f4;
  display: flex;
  flex-direction: column;
  box-shadow: -6px 0 32px rgba(0, 0, 0, 0.5);
  overflow: hidden;
}

.slide-right-enter-active,
.slide-right-leave-active {
  transition: transform 0.35s cubic-bezier(0.4, 0, 0.2, 1), opacity 0.35s;
}
.slide-right-enter-from,
.slide-right-leave-to {
  transform: translateX(100%);
  opacity: 0;
}

/* Title */
.cc-titlebar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  height: 48px;
  background: #181825;
  border-bottom: 1px solid #313244;
  flex-shrink: 0;
}

.cc-title {
  display: flex;
  align-items: center;
  gap: 10px;
  font-weight: 700;
  font-size: 16px;
  color: #cdd6f4;
}

.ctrl-btn {
  background: none;
  border: none;
  cursor: pointer;
  color: #6c7086;
  font-size: 22px;
  line-height: 1;
  padding: 2px 8px;
  border-radius: 4px;
  transition: color 0.15s, background 0.15s;
}
.ctrl-btn:hover { color: #f38ba8; background: rgba(243,139,168,0.1); }

/* Tabs */
.cc-tabs {
  display: flex;
  gap: 0;
  background: #181825;
  border-bottom: 1px solid #313244;
  flex-shrink: 0;
  padding: 0 16px;
}

.cc-tab-btn {
  background: none;
  border: none;
  border-bottom: 2px solid transparent;
  color: #6c7086;
  font-size: 14px;
  font-weight: 600;
  padding: 11px 20px;
  cursor: pointer;
  transition: color 0.15s, border-color 0.15s;
  margin-bottom: -1px;
}
.cc-tab-btn.active { color: #89b4fa; border-bottom-color: #89b4fa; }
.cc-tab-btn:hover:not(.active) { color: #cdd6f4; }

/* Content area */
.cc-content {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.required { color: #f38ba8; }

/* Form styles reuse from GraphToolsPanel */
.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.form-row {
  display: flex;
  gap: 14px;
  align-items: flex-end;
}
.flex-1 { flex: 1; min-width: 0; }
label {
  font-size: 12px;
  font-weight: 600;
  color: #a6adc8;
}
.form-input,
.form-select {
  background: #313244;
  border: 1px solid #45475a;
  border-radius: 6px;
  color: #cdd6f4;
  font-size: 13px;
  padding: 8px 10px;
  outline: none;
  transition: border-color 0.15s;
  width: 100%;
  box-sizing: border-box;
}
.form-input:focus, .form-select:focus { border-color: #89b4fa; }
.form-select option { background: #313244; }
.form-input:disabled, .form-select:disabled { opacity: 0.55; cursor: not-allowed; }

.radio-group { display: flex; gap: 18px; align-items: center; margin-top: 4px; }
.radio-label {
  display: flex; align-items: center; gap: 6px;
  font-size: 13px; color: #cdd6f4; cursor: pointer; font-weight: 400;
}
.radio-label input { accent-color: #89b4fa; }

.form-textarea {
  background: #313244;
  border: 1px solid #45475a;
  border-radius: 6px;
  color: #cdd6f4;
  font-size: 12px;
  padding: 10px 12px;
  outline: none;
  transition: border-color 0.15s;
  width: 100%;
  box-sizing: border-box;
  resize: vertical;
  font-family: 'JetBrains Mono', 'Consolas', monospace;
  line-height: 1.5;
  min-height: 200px;
}
.form-textarea:focus { border-color: #89b4fa; }
.form-textarea:disabled { opacity: 0.55; cursor: not-allowed; }

/* File upload area */
.file-upload-area {
  border: 2px dashed #45475a;
  border-radius: 8px;
  padding: 28px 20px;
  text-align: center;
  cursor: pointer;
  transition: all 0.2s;
  background: rgba(49,50,68,0.3);
  color: #6c7086;
}
.file-upload-area:hover, .file-upload-area.dragover {
  border-color: #89b4fa;
  background: rgba(137,180,250,0.06);
  color: #89b4fa;
}
.file-upload-area svg { margin-bottom: 8px; opacity: 0.6; }
.uploaded-file-name {
  display: inline-flex; align-items: center; gap: 6px;
  font-size: 13px; color: #a6e3a1; font-weight: 500;
}
.clear-file-btn {
  background: none; border: none; color: #f38ba8; font-size: 16px;
  cursor: pointer; padding: 0 4px; line-height: 1;
}
.clear-file-btn:hover { color: #f38ba8; }

/* Command preview */
.cmd-preview {
  background: #181825;
  border: 1px solid #313244;
  border-radius: 6px;
  padding: 10px 14px;
  font-size: 12px;
  color: #89b4fa;
  margin: 0;
  font-family: 'JetBrains Mono', 'Consolas', monospace;
  white-space: pre-wrap; word-break: break-all;
}

/* History preview inside create tab */
.history-preview {
  background: #11111b; border: 1px solid #313244; border-radius: 6px; overflow: hidden;
}
.preview-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 6px 10px; background: #181825; font-size: 12px; font-weight: 600; color: #a6adc8;
}
.preview-actions { display: flex; gap: 6px; }
.preview-content {
  margin: 0; padding: 8px 10px; font-family: 'JetBrains Mono','Consolas',monospace;
  font-size: 11px; color: #a6adc8; white-space: pre-wrap; max-height: 180px; overflow-y: auto;
}

/* Actions */
.form-actions { display: flex; gap: 8px; align-items: center; }
.run-btn {
  display: inline-flex; align-items: center; gap: 6px;
  background: #89b4fa; color: #1e1e2e; border: none;
  border-radius: 6px; padding: 9px 20px; font-size: 13px; font-weight: 700;
  cursor: pointer; transition: background 0.15s, opacity 0.15s;
}
.run-btn:hover:not(:disabled) { background: #b4d0ff; }
.run-btn:disabled { opacity: 0.5; cursor: not-allowed; }

.cancel-btn {
  background: rgba(243,139,168,0.15); color: #f38ba8;
  border: 1px solid rgba(243,139,168,0.4); border-radius: 6px;
  padding: 8px 14px; font-size: 13px; cursor: pointer; transition: background 0.15s;
}
.cancel-btn:hover { background: rgba(243,139,168,0.28); }

.clear-btn {
  background: #313244; color: #a6adc8;
  border: 1px solid #45475a; border-radius: 6px; padding: 8px 14px;
  font-size: 13px; cursor: pointer; transition: background 0.15s;
}
.clear-btn:hover { background: #45475a; }
.help-btn {
  background: #313244; color: #89b4fa;
  border: 1px solid #45475a; border-radius: 6px; padding: 8px 14px;
  font-size: 13px; cursor: pointer; font-weight: 600; transition: background 0.15s;
}
.help-btn:hover { background: #45475a; }

.spin-icon { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

/* Result box */
.result-box { display: flex; flex-direction: column; gap: 8px; min-height: 0; }
.result-header { display: flex; align-items: center; justify-content: flex-end; gap: 12px; }
.result-count { font-size: 12px; color: #a6e3a1; margin-right: auto; }
.copy-btn {
  display: inline-flex; align-items: center; gap: 4px;
  background: #313244; color: #a6adc8; border: 1px solid #45475a;
  border-radius: 5px; padding: 4px 10px; font-size: 12px; cursor: pointer; transition: background 0.15s;
}
.copy-btn:hover { background: #45475a; }

.result-error {
  background: rgba(243,139,168,0.1); border: 1px solid rgba(243,139,168,0.3);
  border-radius: 6px; color: #f38ba8; padding: 10px 12px; font-size: 13px; word-break: break-all;
}
.result-output {
  background: #11111b; border: 1px solid #313244; border-radius: 6px;
  padding: 12px; font-size: 12px; color: #cdd6f4; overflow: auto;
  max-height: 400px; white-space: pre-wrap; margin: 0;
  font-family: 'JetBrains Mono', 'Consolas', monospace;
}

/* View toggle */
.view-toggle { display: flex; gap: 8px; align-items: center; }
.empty-state {
  text-align: center; color: #6c7086; font-size: 13px; padding: 40px 0;
}

/* History cards */
.history-card {
  background: #181825; border: 1px solid #313244; border-radius: 8px; overflow: hidden;
}
.history-card-header {
  display: flex; align-items: center; gap: 10px;
  padding: 12px 16px; cursor: pointer; transition: background 0.12s;
}
.history-card-header:hover { background: #1e1e30; }
.history-name { font-weight: 600; font-size: 14px; color: #cdd6f4; }
.history-meta { font-size: 12px; color: #6c7086; flex: 1; }
.history-expand { color: #6c7086; font-size: 12px; flex-shrink: 0; }

.history-card-detail {
  border-top: 1px solid #313244; padding: 12px 16px;
  display: flex; flex-direction: column; gap: 10px;
}
.config-viewer {
  background: #11111b; border: 1px solid #313244; border-radius: 6px;
  padding: 12px; font-size: 11px; font-family: 'JetBrains Mono','Consolas',monospace;
  color: #a6e3a1; white-space: pre-wrap; max-height: 350px; overflow: auto; margin: 0;
}
.config-placeholder { color: #6c7086; font-size: 12px; margin: 0; }
.history-card-actions { display: flex; gap: 8px; flex-wrap: wrap; }
</style>
