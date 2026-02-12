<template>
  <div id="app-root">
    <HomeView v-if="isAuthenticated" :username="username" @logout="handleLogout" />
    <LoginView v-else @login-success="onLoginSuccess" />
  </div>
</template>

<script>
import { defineAsyncComponent } from 'vue'
import LoginView from './views/LoginView.vue'
import { useAuthStore } from './stores/auth'

const HomeView = defineAsyncComponent(() => import('./views/HomeView.vue'))

export default {
  name: 'App',
  components: {
    HomeView,
    LoginView,
  },
  data() {
    return {
      isAuthenticated: false,
      username: '',
    }
  },
  created() {
    const authStore = useAuthStore()
    this.isAuthenticated = authStore.isAuthenticated
    this.username = authStore.username || ''

    // Verify token on load
    if (authStore.isAuthenticated) {
      authStore.verifyToken().then((valid) => {
        this.isAuthenticated = valid
        if (valid) {
          this.username = authStore.username
        }
      })
    }

    // Listen for token expiration events
    window.addEventListener('auth:expired', this.handleAuthExpired)
  },
  beforeUnmount() {
    window.removeEventListener('auth:expired', this.handleAuthExpired)
  },
  methods: {
    onLoginSuccess() {
      const authStore = useAuthStore()
      this.isAuthenticated = true
      this.username = authStore.username
    },
    handleLogout() {
      const authStore = useAuthStore()
      authStore.logout()
      this.isAuthenticated = false
      this.username = ''
    },
    handleAuthExpired() {
      const authStore = useAuthStore()
      authStore.logout()
      this.isAuthenticated = false
      this.username = ''
    },
  },
}
</script>
