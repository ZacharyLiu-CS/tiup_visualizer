import axios from 'axios'

// Resolve API base URL relative to the app's deploy path.
// In sub-path deployment (e.g. /tiup-visualizer/), Vite sets import.meta.env.BASE_URL
// to that path, so API calls become /tiup-visualizer/api/v1/...
const base = import.meta.env.BASE_URL.replace(/\/+$/, '')

const api = axios.create({
  baseURL: `${base}/api/v1`,
  timeout: 30000,
})

export const clusterAPI = {
  getAllClusters: () => api.get('/clusters'),
  getClusterDetail: (clusterName) => api.get(`/clusters/${clusterName}`),
  getAllHosts: () => api.get('/hosts'),
  getHostClusters: (hostIp) => api.get(`/hosts/${hostIp}/clusters`),
  getLogFileUrl: (clusterName, componentId, filename, action = 'view') => {
    return `${base}/api/v1/logs/${encodeURIComponent(clusterName)}/${encodeURIComponent(componentId)}/${encodeURIComponent(filename)}?action=${action}`
  },
}

export default api
