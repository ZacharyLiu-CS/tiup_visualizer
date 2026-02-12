import { defineStore } from 'pinia'
import { authAPI } from '../services/api'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem('auth_token') || null,
    username: localStorage.getItem('auth_username') || null,
    isAuthenticated: !!localStorage.getItem('auth_token'),
  }),

  actions: {
    async login(username, password) {
      const response = await authAPI.login(username, password)
      const { access_token } = response.data
      this.token = access_token
      this.username = username
      this.isAuthenticated = true
      localStorage.setItem('auth_token', access_token)
      localStorage.setItem('auth_username', username)
      return response.data
    },

    logout() {
      this.token = null
      this.username = null
      this.isAuthenticated = false
      localStorage.removeItem('auth_token')
      localStorage.removeItem('auth_username')
    },

    async verifyToken() {
      if (!this.token) {
        this.isAuthenticated = false
        return false
      }
      try {
        await authAPI.verify(this.token)
        this.isAuthenticated = true
        return true
      } catch {
        this.logout()
        return false
      }
    },
  },
})
