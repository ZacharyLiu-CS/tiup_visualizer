<template>
  <div class="host-card" 
       :class="{ 
         'selected': isSelected, 
         'highlighted': isHighlighted 
       }"
       @click="handleClick">
    <div class="host-icon">
      <svg width="48" height="48" viewBox="0 0 48 48" fill="none">
        <rect x="4" y="8" width="40" height="32" rx="2" stroke="currentColor" stroke-width="2"/>
        <circle cx="10" cy="16" r="1.5" fill="currentColor"/>
        <circle cx="14" cy="16" r="1.5" fill="currentColor"/>
        <line x1="20" y1="16" x2="38" y2="16" stroke="currentColor" stroke-width="1"/>
        <line x1="8" y1="22" x2="40" y2="22" stroke="currentColor" stroke-width="1"/>
        <line x1="8" y1="28" x2="40" y2="28" stroke="currentColor" stroke-width="1"/>
        <line x1="8" y1="34" x2="40" y2="34" stroke="currentColor" stroke-width="1"/>
      </svg>
    </div>
    <div class="host-info">
      <div class="host-ip">{{ host }}</div>
      <div class="host-meta">
        <span class="cluster-count">{{ hostInfo.clusters.length }} clusters</span>
        <span class="component-count">{{ hostInfo.components.length }} components</span>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'HostCard',
  props: {
    host: {
      type: String,
      required: true
    },
    hostInfo: {
      type: Object,
      required: true
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
  methods: {
    handleClick() {
      this.$emit('select', this.host)
    }
  }
}
</script>

<style scoped>
.host-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 16px;
  background: white;
  border: 2px solid #e5e7eb;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.3s ease;
  min-width: 140px;
}

.host-card:hover {
  border-color: #3b82f6;
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.2);
  transform: translateY(-2px);
}

.host-card.selected {
  border-color: #3b82f6;
  background: #eff6ff;
}

.host-card.highlighted {
  border-color: #10b981;
  background: #f0fdf4;
}

.host-icon {
  color: #6b7280;
  margin-bottom: 8px;
}

.host-card.selected .host-icon,
.host-card.highlighted .host-icon {
  color: #3b82f6;
}

.host-info {
  text-align: center;
  width: 100%;
}

.host-ip {
  font-weight: 600;
  font-size: 14px;
  color: #1f2937;
  margin-bottom: 4px;
  word-break: break-all;
}

.host-meta {
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: 11px;
  color: #6b7280;
}

.cluster-count,
.component-count {
  padding: 2px 6px;
  background: #f3f4f6;
  border-radius: 4px;
}
</style>
