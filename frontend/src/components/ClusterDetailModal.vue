<template>
  <div class="cluster-detail-modal" v-if="clusterDetail" @click.self="close">
    <div class="modal-content">
      <div class="modal-header">
        <h2>{{ clusterDetail.cluster_name }}</h2>
        <button class="close-btn" @click="close">×</button>
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
                  <td class="path-cell">{{ component.data_dir }}</td>
                  <td class="path-cell">{{ component.deploy_dir }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'ClusterDetailModal',
  props: {
    clusterDetail: {
      type: Object,
      default: null
    }
  },
  emits: ['close'],
  methods: {
    close() {
      this.$emit('close')
    },
    getStatusClass(status) {
      if (status.includes('Up')) return 'status-up'
      if (status.includes('Down')) return 'status-down'
      return 'status-unknown'
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

.table-container {
  overflow-x: auto;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
}

.components-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
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
}

.host-cell {
  font-family: monospace;
  font-weight: 500;
  color: #1f2937;
}

.path-cell {
  font-family: monospace;
  font-size: 11px;
  color: #6b7280;
  max-width: 250px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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
</style>
