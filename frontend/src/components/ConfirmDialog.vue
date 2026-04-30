<template>
  <teleport to="body">
    <transition name="confirm-fade">
      <div v-if="visible" class="confirm-overlay" @click.self="handleCancel">
        <div class="confirm-dialog" :class="typeClass">
          <div class="confirm-header">
            <div class="confirm-icon" v-if="iconName">
              <!-- Warning Icon -->
              <svg v-if="iconName === 'warning'" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" />
                <line x1="12" y1="9" x2="12" y2="13" />
                <line x1="12" y1="17" x2="12.01" y2="17" />
              </svg>
              <!-- Danger Icon -->
              <svg v-else-if="iconName === 'danger'" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10" />
                <line x1="15" y1="9" x2="9" y2="15" />
                <line x1="9" y1="9" x2="15" y2="15" />
              </svg>
              <!-- Info Icon -->
              <svg v-else width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10" />
                <line x1="12" y1="16" x2="12" y2="12" />
                <line x1="12" y1="8" x2="12.01" y2="8" />
              </svg>
            </div>
            <span class="confirm-title">{{ title }}</span>
            <button class="confirm-close-btn" @click="handleCancel" title="Cancel">&times;</button>
          </div>
          <div class="confirm-body">
            <div class="confirm-message" v-if="!messageHtml">{{ message }}</div>
            <div class="confirm-message" v-else v-html="messageHtml"></div>
          </div>
          <div class="confirm-footer" :class="{ 'single-btn': singleButton }">
            <button
              v-if="!singleButton"
              class="confirm-btn confirm-btn-cancel"
              @click="handleCancel"
            >
              {{ cancelText }}
            </button>
            <button
              class="confirm-btn confirm-btn-confirm"
              :class="confirmClass"
              @click="handleConfirm"
              :disabled="confirmDisabled"
            >
              {{ singleButton ? cancelText : confirmText }}
            </button>
          </div>
        </div>
      </div>
    </transition>
  </teleport>
</template>

<script>
export default {
  name: 'ConfirmDialog',
  props: {
    visible: {
      type: Boolean,
      default: false
    },
    title: {
      type: String,
      default: 'Confirm'
    },
    message: {
      type: String,
      default: ''
    },
    messageHtml: {
      type: String,
      default: ''
    },
    confirmText: {
      type: String,
      default: 'Confirm'
    },
    cancelText: {
      type: String,
      default: 'Cancel'
    },
    singleButton: {
      type: Boolean,
      default: false
    },
    type: {
      type: String,
      default: 'default', // 'default' | 'warning' | 'danger'
    },
    confirmDisabled: {
      type: Boolean,
      default: false
    }
  },
  emits: ['confirm', 'cancel', 'update:visible'],
  computed: {
    typeClass() {
      return `confirm-type-${this.type}`
    },
    confirmClass() {
      return `confirm-btn-${this.type}`
    },
    iconName() {
      if (this.type === 'danger') return 'danger'
      if (this.type === 'warning') return 'warning'
      return 'info'
    }
  },
  watch: {
    visible(val) {
      if (val) {
        document.addEventListener('keydown', this.handleKeydown)
      } else {
        document.removeEventListener('keydown', this.handleKeydown)
      }
    }
  },
  beforeUnmount() {
    document.removeEventListener('keydown', this.handleKeydown)
  },
  methods: {
    handleKeydown(e) {
      if (e.key === 'Escape') {
        this.handleCancel()
      } else if (e.key === 'Enter' && !this.confirmDisabled) {
        this.handleConfirm()
      }
    },
    handleConfirm() {
      console.log('[ConfirmDialog] handleConfirm called, emitting confirm event')
      this.$emit('confirm')
      this.$emit('update:visible', false)
    },
    handleCancel() {
      this.$emit('cancel')
      this.$emit('update:visible', false)
    }
  }
}
</script>

<style scoped>
.confirm-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  z-index: 3000;
  display: flex;
  align-items: center;
  justify-content: center;
}

.confirm-dialog {
  background: #1e1e2e;
  border: 1px solid #313244;
  border-radius: 12px;
  width: 420px;
  max-width: 90vw;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
  overflow: hidden;
}

/* Header */
.confirm-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 16px 20px;
  background: #181825;
  border-bottom: 1px solid #313244;
  font-weight: 700;
  font-size: 15px;
  color: #cdd6f4;
}

.confirm-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.confirm-type-warning .confirm-icon {
  color: #f9e2af;
}

.confirm-type-danger .confirm-icon {
  color: #f38ba8;
}

.confirm-type-default .confirm-icon {
  color: #89b4fa;
}

.confirm-close-btn {
  margin-left: auto;
  background: none;
  border: none;
  color: #6c7086;
  font-size: 20px;
  cursor: pointer;
  line-height: 1;
  padding: 0 4px;
  border-radius: 4px;
  transition: color 0.15s, background 0.15s;
}
.confirm-close-btn:hover {
  color: #f38ba8;
  background: rgba(243, 139, 168, 0.1);
}

/* Body */
.confirm-body {
  padding: 20px;
  max-height: 400px;
  overflow-y: auto;
}

.confirm-message {
  font-size: 13px;
  color: #a6adc8;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: 'Menlo', 'Monaco', 'Courier New', monospace;
}

/* Footer */
.confirm-footer {
  display: flex;
  gap: 10px;
  padding: 16px 20px;
  border-top: 1px solid #313244;
  justify-content: flex-end;
}

.confirm-footer.single-btn {
  justify-content: center;
}

.confirm-btn {
  padding: 8px 20px;
  font-size: 13px;
  font-weight: 700;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s, transform 0.1s;
  border: none;
  outline: none;
}

.confirm-btn:active {
  transform: scale(0.97);
}

.confirm-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.confirm-btn-cancel {
  background: #313244;
  color: #a6adc8;
  border: 1px solid #45475a;
}
.confirm-btn-cancel:hover {
  background: #45475a;
}

/* Confirm button variants */
.confirm-btn-default {
  background: #89b4fa;
  color: #1e1e2e;
}
.confirm-btn-default:hover {
  background: #b4d4fc;
}

.confirm-btn-warning {
  background: #f9e2af;
  color: #1e1e2e;
}
.confirm-btn-warning:hover {
  background: #fbecc8;
}

.confirm-btn-danger {
  background: #f38ba8;
  color: #1e1e2e;
}
.confirm-btn-danger:hover {
  background: #f5a3bb;
}

/* Transition */
.confirm-fade-enter-active,
.confirm-fade-leave-active {
  transition: opacity 0.2s;
}
.confirm-fade-enter-from,
.confirm-fade-leave-to {
  opacity: 0;
}
.confirm-fade-enter-active .confirm-dialog,
.confirm-fade-leave-active .confirm-dialog {
  transition: transform 0.2s;
}
.confirm-fade-enter-from .confirm-dialog,
.confirm-fade-leave-to .confirm-dialog {
  transform: scale(0.95);
}
</style>
