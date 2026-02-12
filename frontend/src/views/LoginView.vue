<template>
  <div class="login-container">
    <div class="login-card">
      <div class="login-header">
        <div class="login-logo">
          <svg viewBox="0 0 48 48" width="48" height="48" fill="none">
            <rect x="4" y="4" width="40" height="40" rx="8" fill="#4f46e5" />
            <path d="M14 24h20M24 14v20M18 18l12 12M30 18l-12 12" stroke="#fff" stroke-width="2.5" stroke-linecap="round" />
          </svg>
        </div>
        <h1 class="login-title">TiUP Visualizer</h1>
        <p class="login-subtitle">Cluster Management Dashboard</p>
      </div>

      <form class="login-form" @submit.prevent="handleLogin">
        <div class="form-group">
          <label class="form-label" for="username">Username</label>
          <div class="input-wrapper">
            <svg class="input-icon" viewBox="0 0 20 20" fill="currentColor" width="18" height="18">
              <path fill-rule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clip-rule="evenodd"/>
            </svg>
            <input
              id="username"
              v-model="username"
              type="text"
              class="form-input"
              placeholder="Enter username"
              autocomplete="username"
              :disabled="loading"
              @keydown.enter="$refs.passwordInput?.focus()"
            />
          </div>
        </div>

        <div class="form-group">
          <label class="form-label" for="password">Password</label>
          <div class="input-wrapper">
            <svg class="input-icon" viewBox="0 0 20 20" fill="currentColor" width="18" height="18">
              <path fill-rule="evenodd" d="M5 9V7a5 5 0 0110 0v2a2 2 0 012 2v5a2 2 0 01-2 2H5a2 2 0 01-2-2v-5a2 2 0 012-2zm8-2v2H7V7a3 3 0 016 0z" clip-rule="evenodd"/>
            </svg>
            <input
              id="password"
              ref="passwordInput"
              v-model="password"
              type="password"
              class="form-input"
              placeholder="Enter password"
              autocomplete="current-password"
              :disabled="loading"
            />
          </div>
        </div>

        <div v-if="error" class="error-message">
          <svg viewBox="0 0 20 20" fill="currentColor" width="16" height="16">
            <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clip-rule="evenodd"/>
          </svg>
          {{ error }}
        </div>

        <button type="submit" class="login-btn" :disabled="loading || !username || !password">
          <span v-if="loading" class="spinner"></span>
          <span v-else>Sign In</span>
        </button>
      </form>
    </div>
  </div>
</template>

<script>
import { useAuthStore } from '../stores/auth'

export default {
  name: 'LoginView',
  emits: ['login-success'],
  data() {
    return {
      username: '',
      password: '',
      error: '',
      loading: false,
    }
  },
  methods: {
    async handleLogin() {
      if (!this.username || !this.password) return
      this.error = ''
      this.loading = true
      try {
        const authStore = useAuthStore()
        await authStore.login(this.username, this.password)
        this.$emit('login-success')
      } catch (err) {
        if (err.response && err.response.status === 401) {
          this.error = 'Invalid username or password'
        } else if (err.response) {
          this.error = err.response.data?.detail || 'Login failed'
        } else {
          this.error = 'Network error, please try again'
        }
      } finally {
        this.loading = false
      }
    },
  },
}
</script>

<style scoped>
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #0f172a 0%, #1e293b 50%, #0f172a 100%);
  padding: 20px;
}

.login-card {
  width: 100%;
  max-width: 420px;
  background: #1e293b;
  border-radius: 16px;
  padding: 40px 36px;
  box-shadow: 0 25px 60px rgba(0, 0, 0, 0.4), 0 0 0 1px rgba(255, 255, 255, 0.05);
}

.login-header {
  text-align: center;
  margin-bottom: 32px;
}

.login-logo {
  margin-bottom: 16px;
}

.login-title {
  font-size: 24px;
  font-weight: 700;
  color: #f1f5f9;
  margin-bottom: 6px;
}

.login-subtitle {
  font-size: 14px;
  color: #64748b;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-label {
  font-size: 13px;
  font-weight: 500;
  color: #94a3b8;
}

.input-wrapper {
  position: relative;
  display: flex;
  align-items: center;
}

.input-icon {
  position: absolute;
  left: 12px;
  color: #475569;
  pointer-events: none;
}

.form-input {
  width: 100%;
  padding: 12px 12px 12px 40px;
  background: #0f172a;
  border: 1px solid #334155;
  border-radius: 10px;
  color: #f1f5f9;
  font-size: 14px;
  outline: none;
  transition: border-color 0.2s, box-shadow 0.2s;
}

.form-input::placeholder {
  color: #475569;
}

.form-input:focus {
  border-color: #4f46e5;
  box-shadow: 0 0 0 3px rgba(79, 70, 229, 0.15);
}

.form-input:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.error-message {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.2);
  border-radius: 8px;
  color: #fca5a5;
  font-size: 13px;
}

.login-btn {
  width: 100%;
  padding: 12px;
  background: #4f46e5;
  color: #fff;
  border: none;
  border-radius: 10px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s, transform 0.1s;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin-top: 4px;
}

.login-btn:hover:not(:disabled) {
  background: #4338ca;
}

.login-btn:active:not(:disabled) {
  transform: scale(0.98);
}

.login-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.spinner {
  width: 20px;
  height: 20px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
