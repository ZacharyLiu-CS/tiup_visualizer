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

          <!-- Region Balancer tab -->
          <div v-if="activeTab === 'balancer'" class="tab-content">

            <!-- Config View -->
            <template v-if="bal.view === 'config'">
              <div class="tool-desc">
                <p>TiKV Region 均衡调度工具，分析 Region 分布并生成最优调度计划，支持批量执行与实时监控。</p>
              </div>

              <!-- PD Source -->
              <div class="form-group">
                <label>PD 来源</label>
                <div class="radio-group">
                  <label class="radio-label">
                    <input type="radio" v-model="bal.pdSource" value="cluster" /> 选择集群
                  </label>
                  <label class="radio-label">
                    <input type="radio" v-model="bal.pdSource" value="custom" /> 自定义 PD 地址
                  </label>
                </div>
              </div>

              <!-- Cluster selector -->
              <div class="form-group" v-if="bal.pdSource === 'cluster'">
                <label>集群</label>
                <select v-model="bal.clusterName" class="form-select">
                  <option value="">-- 选择集群 --</option>
                  <option v-for="c in clusters" :key="c.name" :value="c.name">{{ c.name }}</option>
                </select>
              </div>

              <!-- Custom PD -->
              <div class="form-group" v-else>
                <label>PD 地址</label>
                <div class="pd-input-wrap">
                  <input
                    v-model="bal.customPD"
                    class="form-input"
                    placeholder="10.0.0.1:2379,10.0.0.2:2379"
                    @focus="showBalPDHistory = balPdHistory.length > 0"
                    @blur="hideBalPDHistoryDelayed"
                    @input="showBalPDHistory = false"
                  />
                  <div v-if="showBalPDHistory" class="pd-history-dropdown">
                    <div
                      v-for="(h, i) in balPdHistory"
                      :key="i"
                      class="pd-history-item"
                      @mousedown.prevent="selectBalPDHistory(h)"
                    >
                      <span class="pd-history-text">{{ h }}</span>
                      <button class="pd-history-del" @mousedown.prevent.stop="removeBalPDHistory(i)" title="删除">×</button>
                    </div>
                  </div>
                </div>
              </div>

              <!-- TiUP Version -->
              <div class="form-group">
                <label>TiUP 版本</label>
                <input v-model="bal.tiupVersion" class="form-input" placeholder="v8.1.0" />
              </div>

              <!-- Thresholds row -->
              <div class="form-row">
                <div class="form-group flex-1">
                  <label>Peer 阈值</label>
                  <input v-model.number="bal.peerThreshold" type="number" class="form-input" min="0" max="100" />
                </div>
                <div class="form-group flex-1">
                  <label>Leader 阈值</label>
                  <input v-model.number="bal.leaderThreshold" type="number" class="form-input" min="0" max="100" />
                </div>
              </div>

              <!-- Batch & Concurrency row -->
              <div class="form-row">
                <div class="form-group flex-1">
                  <label>批次大小</label>
                  <input v-model.number="bal.batchSize" type="number" class="form-input" min="1" max="50" />
                </div>
                <div class="form-group flex-1">
                  <label>并发度</label>
                  <input v-model.number="bal.concurrency" type="number" class="form-input" min="1" max="10" />
                </div>
              </div>

              <!-- Actions -->
              <div class="form-actions">
                <button class="run-btn" @click="analyzeCluster" :disabled="bal.analyzing || !balReady">
                  <svg v-if="bal.analyzing" class="spin-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                    <path d="M21 12a9 9 0 11-6.219-8.56" />
                  </svg>
                  <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="14" height="14">
                    <circle cx="11" cy="11" r="8" /><line x1="21" y1="21" x2="16.65" y2="16.65" />
                  </svg>
                  {{ bal.analyzing ? '分析中...' : '分析集群' }}
                </button>
                <button class="clear-btn" @click="switchToQueue">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="13" height="13">
                    <rect x="3" y="3" width="7" height="7" /><rect x="14" y="3" width="7" height="7" /><rect x="3" y="14" width="7" height="7" /><rect x="14" y="14" width="7" height="7" />
                  </svg>
                  查看任务队列
                </button>
              </div>

              <!-- Error -->
              <div v-if="bal.error" class="result-error">{{ bal.error }}</div>

              <!-- Plan preview -->
              <div v-if="bal.plan" class="bal-plan-preview">
                <div class="bal-summary">
                  <span>{{ bal.plan.total_regions }} regions / {{ bal.plan.total_stores }} stores</span>
                  <span class="bal-summary-ops">
                    {{ bal.plan.peer_ops }} peer transfers + {{ bal.plan.leader_ops }} leader transfers = {{ bal.plan.operations.length }} total
                  </span>
                </div>

                <!-- Before distribution -->
                <div class="bal-section">
                  <div class="bal-section-title">当前分布</div>
                  <div class="result-table-wrap">
                    <table class="dist-table">
                      <thead>
                        <tr><th>Store</th><th>Peers</th><th>Peer Δ</th><th>Leaders</th><th>Leader Δ</th></tr>
                      </thead>
                      <tbody>
                        <tr v-for="s in bal.plan.before" :key="s.store_id">
                          <td>{{ s.store_id }}</td>
                          <td>{{ s.peer_count }}</td>
                          <td :class="deltaClass(s.peer_delta)">{{ formatDelta(s.peer_delta) }}</td>
                          <td>{{ s.leader_count }}</td>
                          <td :class="deltaClass(s.leader_delta)">{{ formatDelta(s.leader_delta) }}</td>
                        </tr>
                      </tbody>
                    </table>
                  </div>
                </div>

                <!-- After distribution -->
                <div class="bal-section">
                  <div class="bal-section-title">预期分布</div>
                  <div class="result-table-wrap">
                    <table class="dist-table">
                      <thead>
                        <tr><th>Store</th><th>Peers</th><th>Peer Δ</th><th>Leaders</th><th>Leader Δ</th></tr>
                      </thead>
                      <tbody>
                        <tr v-for="s in bal.plan.after" :key="s.store_id">
                          <td>{{ s.store_id }}</td>
                          <td>{{ s.peer_count }}</td>
                          <td :class="deltaClass(s.peer_delta)">{{ formatDelta(s.peer_delta) }}</td>
                          <td>{{ s.leader_count }}</td>
                          <td :class="deltaClass(s.leader_delta)">{{ formatDelta(s.leader_delta) }}</td>
                        </tr>
                      </tbody>
                    </table>
                  </div>
                </div>

                <!-- Operations list -->
                <div class="bal-section">
                  <div class="bal-section-title">调度操作 ({{ bal.plan.operations.length }})</div>
                  <div class="ops-list">
                    <div v-for="(op, i) in bal.plan.operations" :key="i" class="op-item">
                      <span class="op-index">[{{ i + 1 }}]</span>
                      <span class="op-type" :class="op.type === 'transfer-peer' ? 'op-peer' : 'op-leader'">{{ op.type }}</span>
                      <span>region={{ op.region_id }}</span>
                      <span v-if="op.from_store">store:{{ op.from_store }}→{{ op.to_store }}</span>
                      <span v-else>to:store-{{ op.to_store }}</span>
                    </div>
                  </div>
                </div>

                <!-- Execute button -->
                <div class="form-actions">
                  <button class="run-btn" @click="executePlan" :disabled="!bal.plan || bal.plan.operations.length === 0">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="14" height="14">
                      <polygon points="5 3 19 12 5 21 5 3" />
                    </svg>
                    执行计划
                  </button>
                  <button class="clear-btn" @click="bal.plan = null">清除计划</button>
                </div>
              </div>
            </template>

            <!-- Queue View -->
            <template v-else>
              <div class="view-toggle">
                <button class="clear-btn" @click="switchToConfig">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="13" height="13">
                    <polyline points="15 18 9 12 15 6" />
                  </svg>
                  返回配置
                </button>
                <button class="clear-btn" @click="refreshTasks" :disabled="bal.tasksLoading">
                  <svg :class="{ 'spin-icon': bal.tasksLoading }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="13" height="13">
                    <polyline points="23 4 23 10 17 10" /><path d="M20.49 15a9 9 0 11-2.12-9.36L23 10" />
                  </svg>
                  刷新
                </button>
              </div>

              <!-- Concurrency control -->
              <div class="concurrency-control">
                <label>并发度</label>
                <input v-model.number="bal.concurrency" type="number" class="form-input" min="1" max="10" style="width: 70px" />
                <button class="clear-btn" @click="setConcurrency">设置</button>
              </div>

              <!-- Empty state -->
              <div v-if="bal.tasks.length === 0 && !bal.tasksLoading" class="empty-state">
                暂无调度任务
              </div>

              <!-- Task list -->
              <div v-for="task in bal.tasks" :key="task.id" class="task-card">
                <div class="task-header" @click="toggleTaskDetail(task.id)">
                  <span class="task-status" :class="task.status">
                    <svg v-if="task.status === 'running'" class="spin-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="12" height="12">
                      <path d="M21 12a9 9 0 11-6.219-8.56" />
                    </svg>
                    {{ { pending: '等待', running: '执行中', completed: '完成', cancelled: '已取消', failed: '失败' }[task.status] || task.status }}
                  </span>
                  <span class="task-id">{{ task.id }}</span>
                  <span class="task-meta">{{ task.config.pd_addr }} | PT={{ task.config.peer_threshold }} LT={{ task.config.leader_threshold }}</span>
                  <span class="task-expand">{{ bal.expandedTask === task.id ? '▾' : '▸' }}</span>
                </div>

                <!-- Progress bar -->
                <div v-if="task.status === 'running' && task.total > 0" class="progress-bar">
                  <div class="progress-fill" :style="{ width: (task.progress / task.total * 100) + '%' }"></div>
                  <span class="progress-text">{{ task.progress }}/{{ task.total }}</span>
                </div>

                <div class="task-info">
                  <span class="task-time">{{ formatTime(task.created_at) }}</span>
                  <div class="task-actions">
                    <button v-if="task.status === 'pending' || task.status === 'running'" class="cancel-btn" @click.stop="cancelTask(task.id)">取消</button>
                    <button v-if="task.status === 'completed' || task.status === 'cancelled' || task.status === 'failed'" class="del-btn" @click.stop="deleteTask(task.id)">删除</button>
                  </div>
                </div>

                <!-- Expanded detail -->
                <div v-if="bal.expandedTask === task.id && task.results && task.results.length > 0" class="task-detail">
                  <div class="result-table-wrap">
                    <table class="dist-table">
                      <thead>
                        <tr><th>#</th><th>操作</th><th>Region</th><th>From</th><th>To</th><th>状态</th><th>耗时</th></tr>
                      </thead>
                      <tbody>
                        <tr v-for="(r, i) in task.results" :key="i" :class="{ 'row-error': r.status === 'failed' }">
                          <td>{{ i + 1 }}</td>
                          <td>{{ r.operation.type }}</td>
                          <td>{{ r.operation.region_id }}</td>
                          <td>{{ r.operation.from_store || '-' }}</td>
                          <td>{{ r.operation.to_store }}</td>
                          <td>
                            <span class="task-status" :class="r.status === 'success' ? 'completed' : r.status">{{ r.status }}</span>
                          </td>
                          <td>{{ r.duration || '-' }}</td>
                        </tr>
                      </tbody>
                    </table>
                  </div>
                  <div v-if="task.error" class="result-error" style="margin-top: 8px">{{ task.error }}</div>
                </div>
              </div>
            </template>

          </div>

        </div>
      </div>
    </transition>
  </teleport>
</template>

<script>
import { mapState } from 'pinia'
import { useClusterStore } from '../stores/cluster'
import { tikvAPI, balancerAPI } from '../services/api'

const PD_HISTORY_KEY = 'kv2graph_pd_history'
const BAL_PD_HISTORY_KEY = 'balancer_pd_history'
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
        { id: 'balancer', label: 'Region Balancer' },
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
      bal: {
        pdSource: 'cluster',
        clusterName: '',
        customPD: '',
        tiupVersion: 'v8.1.0',
        peerThreshold: 3,
        leaderThreshold: 2,
        batchSize: 5,
        concurrency: 1,
        analyzing: false,
        plan: null,
        error: '',
        view: 'config',
        tasks: [],
        tasksLoading: false,
        expandedTask: null,
        eventSource: null,
      },
      _balRefreshTimer: null,
      copied: false,
      showPDHistory: false,
      pdHistory: [],
      showBalPDHistory: false,
      balPdHistory: [],
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
    balReady() {
      if (this.bal.pdSource === 'cluster') return !!this.bal.clusterName
      return !!this.bal.customPD.trim()
    },
  },
  watch: {
    activeTab(newVal, oldVal) {
      if (newVal === 'balancer' && this.bal.view === 'queue') {
        this.refreshTasks()
        this.connectSSE()
        this.startAutoRefresh()
      }
      if (oldVal === 'balancer') {
        this.disconnectSSE()
        this.stopAutoRefresh()
      }
    },
  },
  mounted() {
    this.pdHistory = this.loadPDHistory()
    this.balPdHistory = this.loadBalPDHistory()
  },
  beforeUnmount() {
    this.disconnectSSE()
    this.stopAutoRefresh()
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
    // --- Balancer PD History ---
    loadBalPDHistory() {
      try {
        return JSON.parse(localStorage.getItem(BAL_PD_HISTORY_KEY) || '[]')
      } catch { return [] }
    },
    saveBalPDHistory() {
      localStorage.setItem(BAL_PD_HISTORY_KEY, JSON.stringify(this.balPdHistory))
    },
    addBalPDHistory(pd) {
      const trimmed = pd.trim()
      if (!trimmed) return
      this.balPdHistory = [trimmed, ...this.balPdHistory.filter(h => h !== trimmed)].slice(0, PD_HISTORY_MAX)
      this.saveBalPDHistory()
    },
    removeBalPDHistory(i) {
      this.balPdHistory.splice(i, 1)
      this.saveBalPDHistory()
    },
    selectBalPDHistory(h) {
      this.bal.customPD = h
      this.showBalPDHistory = false
    },
    hideBalPDHistoryDelayed() {
      setTimeout(() => { this.showBalPDHistory = false }, 150)
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
    // --- Region Balancer methods ---
    async analyzeCluster() {
      if (!this.balReady) return
      this.bal.analyzing = true
      this.bal.error = ''
      this.bal.plan = null
      // Save PD history on use
      if (this.bal.pdSource === 'custom') {
        this.addBalPDHistory(this.bal.customPD)
      }
      try {
        const params = {
          tiup_version: this.bal.tiupVersion,
          peer_threshold: this.bal.peerThreshold,
          leader_threshold: this.bal.leaderThreshold,
        }
        if (this.bal.pdSource === 'cluster') {
          params.cluster_name = this.bal.clusterName
        } else {
          params.pd_addr = this.bal.customPD.trim()
        }
        const res = await balancerAPI.analyze(params)
        this.bal.plan = res.data
      } catch (e) {
        this.bal.error = e.response?.data?.detail || e.message || '分析失败'
      } finally {
        this.bal.analyzing = false
      }
    },
    async executePlan() {
      if (!this.bal.plan) return
      this.bal.error = ''
      try {
        const params = {
          tiup_version: this.bal.tiupVersion,
          peer_threshold: this.bal.peerThreshold,
          leader_threshold: this.bal.leaderThreshold,
          batch_size: this.bal.batchSize,
          concurrency: this.bal.concurrency,
        }
        if (this.bal.pdSource === 'cluster') {
          params.cluster_name = this.bal.clusterName
        } else {
          params.pd_addr = this.bal.customPD.trim()
        }
        await balancerAPI.createTask(params)
        this.switchToQueue()
      } catch (e) {
        this.bal.error = e.response?.data?.detail || e.message || '创建任务失败'
      }
    },
    async refreshTasks() {
      this.bal.tasksLoading = true
      try {
        const res = await balancerAPI.listTasks()
        this.bal.tasks = res.data || []
      } catch (e) {
        // silently ignore refresh errors
      } finally {
        this.bal.tasksLoading = false
      }
    },
    async cancelTask(id) {
      try {
        await balancerAPI.cancelTask(id)
        await this.refreshTasks()
      } catch (e) {
        this.bal.error = e.response?.data?.detail || e.message || '取消失败'
      }
    },
    async deleteTask(id) {
      try {
        await balancerAPI.deleteTask(id)
        if (this.bal.expandedTask === id) this.bal.expandedTask = null
        await this.refreshTasks()
      } catch (e) {
        this.bal.error = e.response?.data?.detail || e.message || '删除失败'
      }
    },
    async setConcurrency() {
      try {
        await balancerAPI.setConcurrency(this.bal.concurrency)
      } catch (e) {
        this.bal.error = e.response?.data?.detail || e.message || '设置并发度失败'
      }
    },
    toggleTaskDetail(id) {
      this.bal.expandedTask = this.bal.expandedTask === id ? null : id
    },
    connectSSE() {
      this.disconnectSSE()
      try {
        const url = balancerAPI.eventsUrl()
        this.bal.eventSource = new EventSource(url)
        this.bal.eventSource.onmessage = (e) => {
          try {
            const event = JSON.parse(e.data)
            if (event.type === 'task_update' || event.type === 'task_created' || event.type === 'task_deleted') {
              this.refreshTasks()
            }
          } catch {}
        }
        this.bal.eventSource.onerror = () => {
          this.disconnectSSE()
          // auto-reconnect after 5s
          setTimeout(() => {
            if (this.activeTab === 'balancer' && this.bal.view === 'queue') {
              this.connectSSE()
            }
          }, 5000)
        }
      } catch {}
    },
    disconnectSSE() {
      if (this.bal.eventSource) {
        this.bal.eventSource.close()
        this.bal.eventSource = null
      }
    },
    startAutoRefresh() {
      this.stopAutoRefresh()
      this._balRefreshTimer = setInterval(() => {
        if (this.activeTab === 'balancer' && this.bal.view === 'queue') {
          this.refreshTasks()
        }
      }, 3000)
    },
    stopAutoRefresh() {
      if (this._balRefreshTimer) {
        clearInterval(this._balRefreshTimer)
        this._balRefreshTimer = null
      }
    },
    switchToQueue() {
      this.bal.view = 'queue'
      this.refreshTasks()
      this.connectSSE()
      this.startAutoRefresh()
    },
    switchToConfig() {
      this.bal.view = 'config'
      this.disconnectSSE()
      this.stopAutoRefresh()
    },
    formatDelta(v) {
      if (v == null) return ''
      return (v >= 0 ? '+' : '') + v.toFixed(2)
    },
    deltaClass(v) {
      if (v == null) return ''
      if (v > 3) return 'delta-high'
      if (v < -3) return 'delta-low'
      return 'delta-ok'
    },
    formatTime(t) {
      if (!t) return ''
      const d = new Date(t)
      return d.toLocaleString('zh-CN', { hour12: false })
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
  width: 50vw;
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

/* --- Region Balancer styles --- */

.bal-plan-preview {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.bal-summary {
  background: #181825;
  border: 1px solid #313244;
  border-radius: 6px;
  padding: 10px 14px;
  font-size: 13px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px;
}

.bal-summary-ops {
  color: #89b4fa;
  font-weight: 600;
}

.bal-section {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.bal-section-title {
  font-size: 12px;
  font-weight: 700;
  color: #a6adc8;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.dist-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}
.dist-table th {
  background: #181825;
  color: #89b4fa;
  font-weight: 600;
  padding: 6px 10px;
  text-align: right;
  white-space: nowrap;
  border-bottom: 1px solid #313244;
}
.dist-table th:first-child {
  text-align: left;
}
.dist-table td {
  padding: 4px 10px;
  border-bottom: 1px solid #2a2a3d;
  color: #cdd6f4;
  text-align: right;
  font-family: 'JetBrains Mono', 'Consolas', monospace;
  font-size: 11px;
}
.dist-table td:first-child {
  text-align: left;
}
.dist-table tr:last-child td {
  border-bottom: none;
}
.dist-table tr:hover td {
  background: #2a2a3d;
}

.delta-high { color: #f38ba8; }
.delta-low { color: #89b4fa; }
.delta-ok { color: #a6e3a1; }

.ops-list {
  max-height: 300px;
  overflow-y: auto;
  background: #181825;
  border: 1px solid #313244;
  border-radius: 6px;
  padding: 8px;
  font-size: 11px;
  font-family: 'JetBrains Mono', 'Consolas', monospace;
}

.op-item {
  display: flex;
  gap: 8px;
  padding: 2px 0;
  color: #cdd6f4;
}

.op-index {
  color: #6c7086;
  min-width: 36px;
}

.op-type {
  font-weight: 600;
  min-width: 110px;
}
.op-peer { color: #89b4fa; }
.op-leader { color: #a6e3a1; }

/* Queue view */
.view-toggle {
  display: flex;
  gap: 8px;
  align-items: center;
}

.concurrency-control {
  display: flex;
  gap: 8px;
  align-items: center;
  font-size: 12px;
}
.concurrency-control label {
  font-weight: 600;
  color: #a6adc8;
}

.empty-state {
  text-align: center;
  color: #6c7086;
  font-size: 13px;
  padding: 40px 0;
}

.task-card {
  background: #181825;
  border: 1px solid #313244;
  border-radius: 8px;
  overflow: hidden;
}

.task-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  cursor: pointer;
  transition: background 0.12s;
}
.task-header:hover {
  background: #1e1e30;
}

.task-status {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-weight: 700;
  padding: 2px 8px;
  border-radius: 10px;
  white-space: nowrap;
}
.task-status.pending {
  background: rgba(108, 112, 134, 0.2);
  color: #6c7086;
}
.task-status.running {
  background: rgba(137, 180, 250, 0.15);
  color: #89b4fa;
}
.task-status.completed {
  background: rgba(166, 227, 161, 0.15);
  color: #a6e3a1;
}
.task-status.cancelled {
  background: rgba(250, 179, 135, 0.15);
  color: #fab387;
}
.task-status.failed {
  background: rgba(243, 139, 168, 0.15);
  color: #f38ba8;
}

.task-id {
  font-family: 'JetBrains Mono', 'Consolas', monospace;
  font-size: 11px;
  color: #6c7086;
}

.task-meta {
  font-size: 11px;
  color: #a6adc8;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-expand {
  color: #6c7086;
  font-size: 12px;
  flex-shrink: 0;
}

.progress-bar {
  height: 20px;
  background: #313244;
  position: relative;
  margin: 0 14px;
  border-radius: 4px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #89b4fa, #74c7ec);
  border-radius: 4px;
  transition: width 0.3s ease;
}

.progress-text {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 600;
  color: #cdd6f4;
  text-shadow: 0 1px 2px rgba(0,0,0,0.5);
}

.task-info {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 14px 10px;
}

.task-time {
  font-size: 11px;
  color: #6c7086;
}

.task-actions {
  display: flex;
  gap: 6px;
}

.del-btn {
  background: none;
  border: none;
  color: #6c7086;
  font-size: 11px;
  cursor: pointer;
  padding: 2px 8px;
  border-radius: 4px;
  transition: color 0.15s, background 0.15s;
}
.del-btn:hover {
  color: #f38ba8;
  background: rgba(243, 139, 168, 0.1);
}

.task-detail {
  border-top: 1px solid #313244;
  padding: 10px 14px;
  max-height: 300px;
  overflow-y: auto;
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
