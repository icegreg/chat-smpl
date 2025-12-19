<template>
  <div class="incoming-call-overlay">
    <div class="incoming-call-modal">
      <!-- Caller info -->
      <div class="caller-info">
        <div class="caller-avatar">
          <img v-if="callerAvatar" :src="callerAvatar" :alt="callerName" />
          <span v-else class="avatar-placeholder">{{ callerInitial }}</span>
        </div>
        <div class="caller-name">{{ callerName }}</div>
        <div class="call-type">{{ callTypeText }}</div>
      </div>

      <!-- Ringing animation -->
      <div class="ringing-animation">
        <div class="ring ring-1"></div>
        <div class="ring ring-2"></div>
        <div class="ring ring-3"></div>
      </div>

      <!-- Action buttons -->
      <div class="call-actions">
        <button class="action-btn reject" @click="$emit('reject')" title="Reject">
          <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M23 16.67v3.33a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 5.11 2h3.33a2 2 0 0 1 2 1.72c.127.96.361 1.903.7 2.81a2 2 0 0 1-.45 2.11L9.09 10.24a16 16 0 0 0 6 6l1.6-1.6a2 2 0 0 1 2.11-.45c.907.339 1.85.573 2.81.7a2 2 0 0 1 1.72 2.03v.75z" transform="rotate(135 12 12)"></path>
          </svg>
        </button>

        <button class="action-btn answer" @click="$emit('answer')" title="Answer">
          <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M23 16.67v3.33a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 5.11 2h3.33a2 2 0 0 1 2 1.72c.127.96.361 1.903.7 2.81a2 2 0 0 1-.45 2.11L9.09 10.24a16 16 0 0 0 6 6l1.6-1.6a2 2 0 0 1 2.11-.45c.907.339 1.85.573 2.81.7a2 2 0 0 1 1.72 2.03v.75z"></path>
          </svg>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  callerName: string
  callerAvatar?: string
  isConference?: boolean
}>()

defineEmits<{
  (e: 'answer'): void
  (e: 'reject'): void
}>()

const callerInitial = computed(() => {
  return props.callerName?.charAt(0).toUpperCase() || '?'
})

const callTypeText = computed(() => {
  return props.isConference ? 'Conference call' : 'Incoming call'
})
</script>

<style scoped>
.incoming-call-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.85);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.incoming-call-modal {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 32px;
  padding: 48px;
}

.caller-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
}

.caller-avatar {
  width: 120px;
  height: 120px;
  border-radius: 50%;
  overflow: hidden;
  background: #3b82f6;
  display: flex;
  align-items: center;
  justify-content: center;
}

.caller-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar-placeholder {
  font-size: 48px;
  font-weight: 600;
  color: #fff;
}

.caller-name {
  font-size: 28px;
  font-weight: 600;
  color: #fff;
}

.call-type {
  font-size: 16px;
  color: rgba(255, 255, 255, 0.7);
}

.ringing-animation {
  position: relative;
  width: 80px;
  height: 80px;
}

.ring {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  border-radius: 50%;
  border: 2px solid rgba(59, 130, 246, 0.6);
  animation: ring 2s infinite ease-out;
}

.ring-1 {
  width: 40px;
  height: 40px;
}

.ring-2 {
  width: 60px;
  height: 60px;
  animation-delay: 0.3s;
}

.ring-3 {
  width: 80px;
  height: 80px;
  animation-delay: 0.6s;
}

@keyframes ring {
  0% {
    transform: translate(-50%, -50%) scale(1);
    opacity: 1;
  }
  100% {
    transform: translate(-50%, -50%) scale(1.5);
    opacity: 0;
  }
}

.call-actions {
  display: flex;
  gap: 48px;
}

.action-btn {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  border: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.action-btn.reject {
  background: #ef4444;
  color: #fff;
}

.action-btn.reject:hover {
  background: #dc2626;
  transform: scale(1.05);
}

.action-btn.answer {
  background: #22c55e;
  color: #fff;
  animation: pulse 1.5s infinite;
}

.action-btn.answer:hover {
  background: #16a34a;
  transform: scale(1.05);
}

@keyframes pulse {
  0%, 100% {
    box-shadow: 0 0 0 0 rgba(34, 197, 94, 0.7);
  }
  50% {
    box-shadow: 0 0 0 15px rgba(34, 197, 94, 0);
  }
}

.icon {
  width: 32px;
  height: 32px;
}
</style>
