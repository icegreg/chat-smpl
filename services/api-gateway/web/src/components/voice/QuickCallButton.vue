<template>
  <button
    class="quick-call-btn"
    :class="{ active: isInCall, loading: loading }"
    :disabled="loading"
    @click="handleClick"
    :title="buttonTitle"
  >
    <span v-if="loading" class="spinner"></span>
    <svg v-else-if="isInCall" class="icon hangup" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M23 16.67v3.33a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 5.11 2h3.33a2 2 0 0 1 2 1.72c.127.96.361 1.903.7 2.81a2 2 0 0 1-.45 2.11L9.09 10.24a16 16 0 0 0 6 6l1.6-1.6a2 2 0 0 1 2.11-.45c.907.339 1.85.573 2.81.7a2 2 0 0 1 1.72 2.03v.75z" transform="rotate(135 12 12)"></path>
    </svg>
    <svg v-else class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M23 16.67v3.33a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 5.11 2h3.33a2 2 0 0 1 2 1.72c.127.96.361 1.903.7 2.81a2 2 0 0 1-.45 2.11L9.09 10.24a16 16 0 0 0 6 6l1.6-1.6a2 2 0 0 1 2.11-.45c.907.339 1.85.573 2.81.7a2 2 0 0 1 1.72 2.03v.75z"></path>
    </svg>
    <span v-if="showLabel" class="btn-label">{{ buttonLabel }}</span>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useVoiceStore } from '@/stores/voice'

const props = defineProps<{
  chatId: string
  showLabel?: boolean
}>()

const voiceStore = useVoiceStore()

const loading = computed(() => voiceStore.loading)
const isInCall = computed(() => voiceStore.isInCall)

const buttonTitle = computed(() => {
  if (loading.value) return 'Connecting...'
  if (isInCall.value) return 'End call'
  return 'Start call'
})

const buttonLabel = computed(() => {
  if (loading.value) return 'Connecting...'
  if (isInCall.value) return 'End'
  return 'Call'
})

async function handleClick() {
  if (loading.value) return

  if (isInCall.value) {
    // Leave current call/conference
    if (voiceStore.currentConference) {
      await voiceStore.leaveConference()
    } else {
      await voiceStore.hangupCall()
    }
  } else {
    // Start new call
    await voiceStore.startChatCall(props.chatId)
  }
}
</script>

<style scoped>
.quick-call-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  border-radius: 8px;
  border: none;
  background: #22c55e;
  color: #fff;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.quick-call-btn:hover:not(:disabled) {
  background: #16a34a;
}

.quick-call-btn:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}

.quick-call-btn.active {
  background: #ef4444;
}

.quick-call-btn.active:hover:not(:disabled) {
  background: #dc2626;
}

.quick-call-btn.loading {
  background: #6b7280;
}

.icon {
  width: 18px;
  height: 18px;
}

.icon.hangup {
  transform: rotate(0);
}

.spinner {
  width: 18px;
  height: 18px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.btn-label {
  white-space: nowrap;
}

/* Compact mode without label */
.quick-call-btn:not(:has(.btn-label)) {
  padding: 8px;
}
</style>
