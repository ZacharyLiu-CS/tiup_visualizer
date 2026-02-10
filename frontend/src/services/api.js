import axios from 'axios'

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
})

export const clusterAPI = {
  getAllClusters: () => api.get('/clusters'),
  getClusterDetail: (clusterName) => api.get(`/clusters/${clusterName}`),
  getAllHosts: () => api.get('/hosts'),
  getHostClusters: (hostIp) => api.get(`/hosts/${hostIp}/clusters`),
}

export default api
