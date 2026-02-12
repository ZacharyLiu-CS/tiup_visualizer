<template>
  <div class="cluster-card"
       :class="{ 
         'selected': isSelected, 
         'highlighted': isHighlighted 
       }"
       @click="handleClick">
    <span 
      v-if="index" 
      class="cluster-index-badge"
    >{{ index }}</span>
    <div class="cluster-icon">
      <svg width="48" height="48" viewBox="0 0 48 48" fill="none">
        <circle cx="24" cy="12" r="6" stroke="currentColor" stroke-width="2"/>
        <circle cx="12" cy="32" r="6" stroke="currentColor" stroke-width="2"/>
        <circle cx="36" cy="32" r="6" stroke="currentColor" stroke-width="2"/>
        <line x1="24" y1="18" x2="16" y2="26" stroke="currentColor" stroke-width="2"/>
        <line x1="24" y1="18" x2="32" y2="26" stroke="currentColor" stroke-width="2"/>
      </svg>
    </div>
    <div class="cluster-info">
      <div class="cluster-name">{{ cluster.name }}</div>
      <div class="cluster-version">{{ cluster.version }}</div>
      <div class="cluster-user">User: {{ cluster.user }}</div>
    </div>
    <div class="status-bar" :class="statusClass"></div>
  </div>
</template>

<script>
export default {
  name: 'ClusterCard',
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
  emits: ['select'],
  computed: {
    statusClass() {
      const status = this.cluster.status
      if (status === 'healthy') return 'status-healthy'
      if (status === 'partial') return 'status-partial'
      if (status === 'unhealthy') return 'status-unknown'
      return 'status-unhealthy'
    }
  },
  methods: {
    handleClick() {
      this.$emit('select', this.cluster.name)
    }
  }
}
</script>

<style scoped>
.cluster-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 16px;
  padding-bottom: 0;
  background: white;
  border: 2px solid #e5e7eb;
  border-radius: 8px;
  cursor: pointer;
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
  margin-bottom: 8px;
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

.status-bar {
  width: calc(100% + 4px);
  height: 6px;
  margin-top: 12px;
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
</style>
