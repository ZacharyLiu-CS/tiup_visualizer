import { defineStore } from 'pinia'
import { clusterAPI } from '../services/api'

export const useClusterStore = defineStore('cluster', {
  state: () => ({
    clusters: [],
    hosts: {},
    selectedHost: null,
    selectedCluster: null,
    clusterDetail: null,
    loading: false,
    error: null,
  }),

  actions: {
    async fetchClusters() {
      this.loading = true
      this.error = null
      try {
        const response = await clusterAPI.getAllClusters()
        this.clusters = response.data
      } catch (error) {
        this.error = error.message
        console.error('Failed to fetch clusters:', error)
      } finally {
        this.loading = false
      }
    },

    async fetchHosts() {
      this.loading = true
      this.error = null
      try {
        const response = await clusterAPI.getAllHosts()
        this.hosts = response.data
      } catch (error) {
        this.error = error.message
        console.error('Failed to fetch hosts:', error)
      } finally {
        this.loading = false
      }
    },

    async fetchClusterDetail(clusterName) {
      this.loading = true
      this.error = null
      try {
        const response = await clusterAPI.getClusterDetail(clusterName)
        this.clusterDetail = response.data
      } catch (error) {
        this.error = error.message
        console.error('Failed to fetch cluster detail:', error)
      } finally {
        this.loading = false
      }
    },

    selectHost(hostIp) {
      this.selectedHost = hostIp
      this.selectedCluster = null
      this.clusterDetail = null
    },

    async selectCluster(clusterName) {
      this.selectedCluster = clusterName
      this.selectedHost = null
      await this.fetchClusterDetail(clusterName)
    },

    clearSelection() {
      this.selectedHost = null
      this.selectedCluster = null
      this.clusterDetail = null
    },

    getHostsForCluster(clusterName) {
      const hostsForCluster = []
      Object.entries(this.hosts).forEach(([hostIp, hostInfo]) => {
        if (hostInfo.clusters.includes(clusterName)) {
          hostsForCluster.push(hostIp)
        }
      })
      return hostsForCluster
    },
  },
})
