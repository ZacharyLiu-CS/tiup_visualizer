import axios from 'axios'

// Resolve API base URL relative to the app's deploy path.
// In sub-path deployment (e.g. /tiup-visualizer/), Vite sets import.meta.env.BASE_URL
// to that path, so API calls become /tiup-visualizer/api/v1/...
const base = import.meta.env.BASE_URL.replace(/\/+$/, '')

const api = axios.create({
  baseURL: `${base}/api/v1`,
  timeout: 30000,
})

// Request interceptor: attach JWT token to all requests
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('auth_token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

// Response interceptor: handle 401 by redirecting to login
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response && error.response.status === 401) {
      localStorage.removeItem('auth_token')
      localStorage.removeItem('auth_username')
      // Dispatch a custom event so App.vue can react
      window.dispatchEvent(new CustomEvent('auth:expired'))
    }
    return Promise.reject(error)
  }
)

export const authAPI = {
  login: (username, password) => api.post('/auth/login', { username, password }),
  verify: (token) => api.get('/auth/verify', { params: { token } }),
}

export const clusterAPI = {
  getOverview: () => api.get('/overview'),
  getAllClusters: () => api.get('/clusters'),
  getClusterDetail: (clusterName) => api.get(`/clusters/${clusterName}`),
  getAllHosts: () => api.get('/hosts'),
  getHostClusters: (hostIp) => api.get(`/hosts/${hostIp}/clusters`),
  getLogFileUrl: (clusterName, componentId, filename, action = 'view', tail = 0) => {
    const token = localStorage.getItem('auth_token')
    let url = `${base}/api/v1/logs/${encodeURIComponent(clusterName)}/${encodeURIComponent(componentId)}/${encodeURIComponent(filename)}?action=${action}&token=${encodeURIComponent(token || '')}`
    if (tail > 0) {
      url += `&tail=${tail}`
    }
    return url
  },
}

export const serverLogAPI = {
  listLogs: () => api.get('/server-logs'),
  getLogUrl: (filename, action = 'view') => {
    const token = localStorage.getItem('auth_token')
    return `${base}/api/v1/server-logs/${encodeURIComponent(filename)}?action=${action}&token=${encodeURIComponent(token || '')}`
  },
}

export default api
