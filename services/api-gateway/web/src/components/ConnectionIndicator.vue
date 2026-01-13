<script setup lang="ts">
import { computed } from 'vue'
import { useNetworkStore } from '@/stores/network'
import { useVoiceStore } from '@/stores/voice'

const networkStore = useNetworkStore()
const voiceStore = useVoiceStore()

const centrifugoConnected = computed(() => networkStore.isWebSocketConnected)
const vertoConnected = computed(() => voiceStore.isConnected)

const allConnected = computed(() => centrifugoConnected.value && vertoConnected.value)
const noneConnected = computed(() => !centrifugoConnected.value && !vertoConnected.value)

const statusText = computed(() => {
  if (allConnected.value) return 'Все каналы подключены'
  if (noneConnected.value) return 'Нет подключения'
  const parts = []
  if (!centrifugoConnected.value) parts.push('Centrifugo')
  if (!vertoConnected.value) parts.push('Verto')
  return `Отключено: ${parts.join(', ')}`
})
</script>

<template>
  <div class="connection-indicator" :title="statusText">
    <!-- Centrifugo eye -->
    <div
      class="eye"
      :class="{ connected: centrifugoConnected, disconnected: !centrifugoConnected }"
      title="Centrifugo (чат)"
    >
      <div class="pupil"></div>
    </div>

    <!-- Verto eye -->
    <div
      class="eye"
      :class="{ connected: vertoConnected, disconnected: !vertoConnected }"
      title="Verto (звонки)"
    >
      <div class="pupil"></div>
    </div>
  </div>
</template>

<style scoped>
.connection-indicator {
  display: flex;
  gap: 4px;
  padding: 4px;
  cursor: help;
}

.eye {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.3s ease;
  position: relative;
}

.eye.connected {
  background: radial-gradient(circle at 30% 30%, #4ade80, #22c55e);
  box-shadow: 0 0 6px rgba(34, 197, 94, 0.6);
}

.eye.disconnected {
  background: radial-gradient(circle at 30% 30%, #f87171, #ef4444);
  box-shadow: 0 0 6px rgba(239, 68, 68, 0.6);
  animation: pulse-red 2s infinite;
}

.pupil {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.4);
  position: relative;
}

.pupil::after {
  content: '';
  position: absolute;
  top: 1px;
  left: 1px;
  width: 2px;
  height: 2px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.8);
}

@keyframes pulse-red {
  0%, 100% {
    box-shadow: 0 0 6px rgba(239, 68, 68, 0.6);
  }
  50% {
    box-shadow: 0 0 12px rgba(239, 68, 68, 0.9);
  }
}
</style>
