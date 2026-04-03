<template>
  <teleport to="body">
    <transition name="slide-right">
      <div v-if="visible" class="graph-tools-overlay" @click.self="$emit('close')">
        <div class="graph-tools-panel">
          <!-- Title bar -->
          <div class="panel-titlebar">
            <div class="panel-title">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="18" height="18">
                <circle cx="5" cy="12" r="2" /><circle cx="19" cy="5" r="2" /><circle cx="19" cy="19" r="2" />
                <line x1="7" y1="11" x2="17" y2="6" /><line x1="7" y1="13" x2="17" y2="18" />
              </svg>
              Graph Tools
            </div>
            <button class="ctrl-btn close-btn" @click="$emit('close')" title="Close">&times;</button>
          </div>

          <!-- Tool tabs -->
          <div class="panel-tabs">
            <button
              v-for="tab in tabs"
              :key="tab.id"
              class="tab-btn"
              :class="{ active: activeTab === tab.id }"
              @click="activeTab = tab.id"
            >
              {{ tab.label }}
            </button>
          </div>

          <!-- KV2Graph tab -->
          <div v-if="activeTab === 'kv2graph'" class="tab-content">
            <div class="tool-desc">
              <p>基于 TiKV RawKV 的图元数据读取与解析工具，支持单点查询和前缀扫描。</p>
            </div>

            <!-- PD Source -->
            <div class="form-group">
              <label>PD 来源</label>
              <div class="radio-group">
                <label class="radio-label">
                  <input type="radio" v-model="kv.pdSource" value="cluster" /> 选择集群
                </label>
                <label class="radio-label">
                  <input type="radio" v-model="kv.pdSource" value="custom" /> 自定义 PD 地址
                </label>
              </div>
            </div>

            <!-- Cluster selector -->
            <div class="form-group" v-if="kv.pdSource === 'cluster'">
              <label>集群</label>
              <select v-model="kv.clusterName" class="form-select">
                <option value="">-- 选择集群 --</option>
                <option v-for="c in clusters" :key="c.name" :value="c.name">{{ c.name }}</option>
              </select>
            </div>

            <!-- Custom PD -->
            <div class="form-group" v-else>
              <label>PD 地址</label>
              <div class="pd-input-wrap">
                <input
                  v-model="kv.customPD"
                  class="form-input"
                  placeholder="10.0.0.1:2379,10.0.0.2:2379"
                  @focus="showPDHistory = pdHistory.length > 0"
                  @blur="hidePDHistoryDelayed"
                  @input="showPDHistory = false"
                  @keyup.enter="runKV2Graph"
                />
                <div v-if="showPDHistory" class="pd-history-dropdown">
                  <div
                    v-for="(h, i) in pdHistory"
                    :key="i"
                    class="pd-history-item"
                    @mousedown.prevent="selectPDHistory(h)"
                  >
                    <span class="pd-history-text">{{ h }}</span>
                    <button class="pd-history-del" @mousedown.prevent.stop="removePDHistory(i)" title="删除">×</button>
                  </div>
                </div>
              </div>
            </div>

            <!-- Query mode -->
            <div class="form-group">
              <label>查询模式</label>
              <div class="radio-group">
                <label class="radio-label">
                  <input type="radio" v-model="kv.mode" value="key" /> 单点查询
                </label>
                <label class="radio-label">
                  <input type="radio" v-model="kv.mode" value="scan" /> 前缀扫描
                </label>
              </div>
            </div>

            <!-- Key / Prefix input -->
            <div class="form-group" v-if="kv.mode === 'key'">
              <label>Key</label>
              <input v-model="kv.key" class="form-input" placeholder="graph:1" @keyup.enter="runKV2Graph" />
            </div>
            <div class="form-group" v-else>
              <label>前缀</label>
              <input v-model="kv.prefix" class="form-input" placeholder="g%" @keyup.enter="runKV2Graph" />
            </div>

            <!-- Options row -->
            <div class="form-row">
              <div class="form-group flex-1">
                <label>Column Family</label>
                <select v-model="kv.cf" class="form-select">
                  <option value="default">default</option>
                  <option value="write">write</option>
                  <option value="lock">lock</option>
                </select>
              </div>
              <div class="form-group flex-1">
                <label>解析方式</label>
                <select v-model="kv.parseType" class="form-select">
                  <option value="graph_meta">graph_meta</option>
                  <option value="hex">hex</option>
                </select>
              </div>
              <div class="form-group flex-1" v-if="kv.mode === 'scan'">
                <label>Limit</label>
                <input v-model.number="kv.limit" type="number" class="form-input" placeholder="100" min="1" max="10000" />
              </div>
            </div>

            <!-- Run button -->
            <div class="form-actions">
              <button class="run-btn" @click="runKV2Graph" :disabled="kv.loading || !kvReady">
                <svg v-if="kv.loading" class="spin-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                  <path d="M21 12a9 9 0 11-6.219-8.56" />
                </svg>
                <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="14" height="14">
                  <polygon points="5 3 19 12 5 21 5 3" />
                </svg>
                {{ kv.loading ? '查询中...' : '执行查询' }}
              </button>
              <button class="cancel-btn" v-if="kv.loading" @click="cancelKV2Graph">取消</button>
              <button class="clear-btn" @click="clearKVResult" v-if="(kv.result !== null || kv.error) && !kv.loading">清除结果</button>
            </div>

            <!-- Empty result hint -->
            <div v-if="kv.result !== null && !kv.error && kv.mode === 'scan' && kv.result.length === 0" class="result-empty">
              未找到匹配前缀 <code>{{ kv.prefix }}</code> 的数据
            </div>

            <!-- Error -->
            <div v-if="kv.error" class="result-error">{{ kv.error }}</div>

            <!-- Result -->
            <div v-if="kv.result !== null && !kv.error && !(kv.mode === 'scan' && kv.result.length === 0)" class="result-box">
              <div class="result-header">
                <span class="result-count" v-if="kv.mode === 'scan'">
                  共 {{ kv.resultCount }} 条
                  <span v-if="parseErrorCount > 0" class="parse-error-hint">（{{ parseErrorCount }} 条解析异常）</span>
                </span>
                <button class="copy-btn" @click="copyResult" title="复制 JSON">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="13" height="13">
                    <rect x="9" y="9" width="13" height="13" rx="2" ry="2" /><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" />
                  </svg>
                  {{ copied ? '已复制' : '复制' }}
                </button>
              </div>

              <!-- Scan table view (graph_meta + simple) -->
              <div v-if="kv.mode === 'scan' && kv.parseType === 'graph_meta' && Array.isArray(kv.result)" class="result-table-wrap">
                <table class="result-table">
                  <thead>
                    <tr>
                      <th v-for="col in scanColumns" :key="col">{{ col }}</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="(row, i) in kv.result" :key="i" :class="{ 'row-error': row.parseError }">
                      <td v-for="col in scanColumns" :key="col">{{ row[col] ?? '' }}</td>
                    </tr>
                  </tbody>
                </table>
              </div>

              <!-- JSON view -->
              <pre v-else class="result-json">{{ kv.resultJson }}</pre>
            </div>
          </div>

        </div>
      </div>
    </transition>
  </teleport>
</template>

<script>
import { mapState } from 'pinia'
import { useClusterStore } from '../stores/cluster'
import { tikvAPI } from '../services/api'

const PD_HISTORY_KEY = 'kv2graph_pd_history'
const PD_HISTORY_MAX = 10
const QUERY_TIMEOUT = 60000 // 60s

export default {
  name: 'GraphToolsPanel',
  props: {
    visible: {
      type: Boolean,
      default: false
    }
  },
  emits: ['close'],
  data() {
    return {
      activeTab: 'kv2graph',
      tabs: [
        { id: 'kv2graph', label: 'KV2Graph' },
      ],
      kv: {
        pdSource: 'cluster',
        clusterName: '',
        customPD: '',
        mode: 'scan',
        key: '',
        prefix: 'g%',
        cf: 'default',
        parseType: 'graph_meta',
        limit: 100,
        loading: false,
        result: null,
        resultCount: 0,
        resultJson: '',
        error: '',
      },
      copied: false,
      showPDHistory: false,
      pdHistory: [],
      _abortController: null,
    }
  },
  computed: {
    ...mapState(useClusterStore, ['clusters']),
    kvReady() {
      if (this.kv.pdSource === 'cluster') return !!this.kv.clusterName
      return !!this.kv.customPD.trim()
    },
    parseErrorCount() {
      if (!Array.isArray(this.kv.result)) return 0
      return this.kv.result.filter(r => r.parseError).length
    },
    scanColumns() {
      if (!Array.isArray(this.kv.result) || this.kv.result.length === 0) return []
      const all = new Set()
      this.kv.result.forEach(row => Object.keys(row).forEach(k => all.add(k)))
      const priority = ['id', 'name', 'key']
      const rest = [...all].filter(k => !priority.includes(k))
      return [...priority.filter(k => all.has(k)), ...rest]
    },
  },
  mounted() {
    this.pdHistory = this.loadPDHistory()
  },
  methods: {
    loadPDHistory() {
      try {
        return JSON.parse(localStorage.getItem(PD_HISTORY_KEY) || '[]')
      } catch { return [] }
    },
    savePDHistory() {
      localStorage.setItem(PD_HISTORY_KEY, JSON.stringify(this.pdHistory))
    },
    addPDHistory(pd) {
      const trimmed = pd.trim()
      if (!trimmed) return
      this.pdHistory = [trimmed, ...this.pdHistory.filter(h => h !== trimmed)].slice(0, PD_HISTORY_MAX)
      this.savePDHistory()
    },
    removePDHistory(i) {
      this.pdHistory.splice(i, 1)
      this.savePDHistory()
    },
    selectPDHistory(h) {
      this.kv.customPD = h
      this.showPDHistory = false
    },
    hidePDHistoryDelayed() {
      setTimeout(() => { this.showPDHistory = false }, 150)
    },
    async runKV2Graph() {
      if (!this.kvReady) return
      // Cancel any previous in-flight request
      if (this._abortController) {
        this._abortController.abort()
      }
      this._abortController = new AbortController()
      const signal = this._abortController.signal

      this.kv.loading = true
      this.kv.error = ''
      this.kv.result = null
      this.kv.resultCount = 0
      this.kv.resultJson = ''

      // Save PD history on use
      if (this.kv.pdSource === 'custom') {
        this.addPDHistory(this.kv.customPD)
      }

      // Timeout via AbortController
      const timer = setTimeout(() => {
        if (this._abortController) this._abortController.abort()
      }, QUERY_TIMEOUT)

      try {
        const opts = { 'parse-type': this.kv.parseType, cf: this.kv.cf }
        const useDirect = this.kv.pdSource === 'custom'
        if (this.kv.mode === 'key') {
          if (!this.kv.key) { this.kv.error = '请输入 Key'; return }
          const res = useDirect
            ? await tikvAPI.directGetKey(this.kv.customPD.trim(), this.kv.key, opts, signal)
            : await tikvAPI.getKey(this.kv.clusterName, this.kv.key, opts, signal)
          this.kv.result = res.data
          this.kv.resultJson = JSON.stringify(res.data, null, 2)
        } else {
          if (!this.kv.prefix) { this.kv.error = '请输入前缀'; return }
          opts.limit = this.kv.limit || 100
          const res = useDirect
            ? await tikvAPI.directScanPrefix(this.kv.customPD.trim(), this.kv.prefix, opts, signal)
            : await tikvAPI.scanPrefix(this.kv.clusterName, this.kv.prefix, opts, signal)
          this.kv.result = res.data.entries || []
          this.kv.resultCount = res.data.total || 0
          this.kv.resultJson = JSON.stringify(this.kv.result, null, 2)
          if (this.kv.result.length === 0) {
            // empty result is handled by template, no error
          }
        }
      } catch (e) {
        if (e.name === 'CanceledError' || e.code === 'ERR_CANCELED') {
          this.kv.error = '查询已取消'
        } else if (e.name === 'AbortError') {
          this.kv.error = `查询超时（>${QUERY_TIMEOUT / 1000}s），请检查 PD 地址是否可达`
        } else {
          const msg = e.response?.data?.detail || e.message || '查询失败'
          // Distinguish connection errors from data errors
          if (e.code === 'ECONNABORTED' || msg.toLowerCase().includes('timeout')) {
            this.kv.error = `连接超时：${msg}`
          } else {
            this.kv.error = msg
          }
        }
      } finally {
        clearTimeout(timer)
        this.kv.loading = false
        this._abortController = null
      }
    },
    cancelKV2Graph() {
      if (this._abortController) {
        this._abortController.abort()
        this._abortController = null
      }
    },
    clearKVResult() {
      this.kv.result = null
      this.kv.error = ''
      this.kv.resultJson = ''
    },
    async copyResult() {
      try {
        await navigator.clipboard.writeText(this.kv.resultJson)
        this.copied = true
        setTimeout(() => { this.copied = false }, 2000)
      } catch {}
    },
  }
}
</script>

<style scoped>
.graph-tools-overlay {
  position: fixed;
  inset: 0;
  z-index: 1200;
  background: rgba(0, 0, 0, 0.25);
}

.graph-tools-panel {
  position: fixed;
  top: 0;
  right: 0;
  width: 520px;
  max-width: 100vw;
  height: 100vh;
  background: #1e1e2e;
  color: #cdd6f4;
  display: flex;
  flex-direction: column;
  box-shadow: -4px 0 24px rgba(0, 0, 0, 0.4);
  overflow: hidden;
}

/* Slide-right transition */
.slide-right-enter-active,
.slide-right-leave-active {
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1), opacity 0.3s;
}
.slide-right-enter-from,
.slide-right-leave-to {
  transform: translateX(100%);
  opacity: 0;
}
.slide-right-enter-to,
.slide-right-leave-from {
  transform: translateX(0);
  opacity: 1;
}

/* Title bar */
.panel-titlebar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  height: 44px;
  background: #181825;
  border-bottom: 1px solid #313244;
  flex-shrink: 0;
}

.panel-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 700;
  font-size: 14px;
  color: #cdd6f4;
}

.ctrl-btn {
  background: none;
  border: none;
  cursor: pointer;
  color: #6c7086;
  font-size: 20px;
  line-height: 1;
  padding: 2px 6px;
  border-radius: 4px;
  transition: color 0.15s, background 0.15s;
}
.ctrl-btn:hover {
  color: #f38ba8;
  background: rgba(243, 139, 168, 0.1);
}

/* Tabs */
.panel-tabs {
  display: flex;
  gap: 0;
  background: #181825;
  border-bottom: 1px solid #313244;
  flex-shrink: 0;
  padding: 0 12px;
}

.tab-btn {
  background: none;
  border: none;
  border-bottom: 2px solid transparent;
  color: #6c7086;
  font-size: 13px;
  font-weight: 600;
  padding: 10px 16px;
  cursor: pointer;
  transition: color 0.15s, border-color 0.15s;
  margin-bottom: -1px;
}
.tab-btn.active {
  color: #89b4fa;
  border-bottom-color: #89b4fa;
}
.tab-btn:hover:not(.active) {
  color: #cdd6f4;
}

/* Tab content */
.tab-content {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.tool-desc p {
  margin: 0;
  font-size: 12px;
  color: #6c7086;
  line-height: 1.5;
}

/* Form */
.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-row {
  display: flex;
  gap: 12px;
  align-items: flex-end;
}

.flex-1 {
  flex: 1;
  min-width: 0;
}

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
  padding: 7px 10px;
  outline: none;
  transition: border-color 0.15s;
  width: 100%;
  box-sizing: border-box;
}
.form-input:focus,
.form-select:focus {
  border-color: #89b4fa;
}
.form-select option {
  background: #313244;
}

.radio-group {
  display: flex;
  gap: 20px;
}
.radio-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #cdd6f4;
  cursor: pointer;
  font-weight: 400;
}
.radio-label input {
  accent-color: #89b4fa;
}

/* Actions */
.form-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.run-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: #89b4fa;
  color: #1e1e2e;
  border: none;
  border-radius: 6px;
  padding: 8px 18px;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
  transition: background 0.15s, opacity 0.15s;
}
.run-btn:hover:not(:disabled) {
  background: #b4d0ff;
}
.run-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.clear-btn {
  background: #313244;
  color: #a6adc8;
  border: 1px solid #45475a;
  border-radius: 6px;
  padding: 8px 14px;
  font-size: 13px;
  cursor: pointer;
  transition: background 0.15s;
}
.clear-btn:hover {
  background: #45475a;
}

/* Spin animation */
.spin-icon {
  animation: spin 0.8s linear infinite;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Error */
.result-error {
  background: rgba(243, 139, 168, 0.1);
  border: 1px solid rgba(243, 139, 168, 0.3);
  border-radius: 6px;
  color: #f38ba8;
  padding: 10px 12px;
  font-size: 13px;
  word-break: break-all;
}

/* Empty result */
.result-empty {
  background: rgba(166, 173, 200, 0.08);
  border: 1px solid #45475a;
  border-radius: 6px;
  color: #6c7086;
  padding: 10px 12px;
  font-size: 13px;
}
.result-empty code {
  color: #89b4fa;
  font-family: monospace;
}

/* Cancel button */
.cancel-btn {
  background: rgba(243, 139, 168, 0.15);
  color: #f38ba8;
  border: 1px solid rgba(243, 139, 168, 0.4);
  border-radius: 6px;
  padding: 8px 14px;
  font-size: 13px;
  cursor: pointer;
  transition: background 0.15s;
}
.cancel-btn:hover {
  background: rgba(243, 139, 168, 0.28);
}

/* Parse error hint */
.parse-error-hint {
  color: #fab387;
  margin-left: 4px;
}

/* Row error highlight */
.result-table tr.row-error td {
  color: #fab387;
  background: rgba(250, 179, 135, 0.06);
}

/* PD history dropdown */
.pd-input-wrap {
  position: relative;
}

.pd-history-dropdown {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  right: 0;
  background: #1e1e2e;
  border: 1px solid #45475a;
  border-radius: 6px;
  z-index: 100;
  box-shadow: 0 4px 16px rgba(0,0,0,0.4);
  max-height: 200px;
  overflow-y: auto;
}

.pd-history-item {
  display: flex;
  align-items: center;
  padding: 7px 10px;
  cursor: pointer;
  transition: background 0.12s;
  gap: 8px;
}
.pd-history-item:hover {
  background: #313244;
}

.pd-history-text {
  flex: 1;
  font-size: 12px;
  color: #cdd6f4;
  font-family: monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.pd-history-del {
  background: none;
  border: none;
  color: #6c7086;
  font-size: 15px;
  cursor: pointer;
  padding: 0 2px;
  line-height: 1;
  flex-shrink: 0;
  border-radius: 3px;
  transition: color 0.12s;
}
.pd-history-del:hover {
  color: #f38ba8;
}

/* Result box */
.result-box {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-height: 0;
}

.result-header {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
}

.result-count {
  font-size: 12px;
  color: #a6e3a1;
  margin-right: auto;
}

.copy-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: #313244;
  color: #a6adc8;
  border: 1px solid #45475a;
  border-radius: 5px;
  padding: 4px 10px;
  font-size: 12px;
  cursor: pointer;
  transition: background 0.15s;
}
.copy-btn:hover {
  background: #45475a;
}

/* Table */
.result-table-wrap {
  overflow-x: auto;
  border-radius: 6px;
  border: 1px solid #313244;
}

.result-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}
.result-table th {
  background: #181825;
  color: #89b4fa;
  font-weight: 600;
  padding: 8px 12px;
  text-align: left;
  white-space: nowrap;
  border-bottom: 1px solid #313244;
}
.result-table td {
  padding: 6px 12px;
  border-bottom: 1px solid #2a2a3d;
  color: #cdd6f4;
  word-break: break-all;
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.result-table tr:last-child td {
  border-bottom: none;
}
.result-table tr:hover td {
  background: #2a2a3d;
}

/* JSON pre */
.result-json {
  background: #181825;
  border: 1px solid #313244;
  border-radius: 6px;
  padding: 12px;
  font-size: 12px;
  color: #a6e3a1;
  overflow: auto;
  max-height: 400px;
  white-space: pre;
  margin: 0;
  font-family: 'JetBrains Mono', 'Consolas', monospace;
}
</style>
