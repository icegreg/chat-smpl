<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useHealthStore } from '@/stores/health'

const healthStore = useHealthStore()
const showDetails = ref(false)

onMounted(() => {
  healthStore.startPolling(10000) // Poll every 10s
})

onUnmounted(() => {
  healthStore.stopPolling()
})

const voiceStatusLabel = computed(() => {
  switch (healthStore.health.voiceStatus) {
    case 'OK':
      return 'Работает'
    case 'ERROR':
      return 'Ошибка'
    case 'DISABLED':
      return 'Отключён'
    default:
      return '-'
  }
})

function formatTime(date: Date | null) {
  if (!date) return '-'
  return date.toLocaleTimeString('ru-RU', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

function formatMs(ms: number | null) {
  if (ms === null) return '-'
  return `${ms}ms`
}

function toggleDetails() {
  showDetails.value = !showDetails.value
}
</script>

<template>
  <div class="system-status">
    <!-- Compact indicator -->
    <button
      class="status-dot"
      :style="{ backgroundColor: healthStore.statusColor }"
      @click="toggleDetails"
      :title="healthStore.statusLabel"
    >
      <span
        v-if="healthStore.isDegraded || healthStore.isDown"
        class="pulse"
        :style="{ backgroundColor: healthStore.statusColor + '66' }"
      ></span>
    </button>

    <!-- Expanded details panel -->
    <Transition name="slide-fade">
      <div v-if="showDetails" class="details-panel">
        <div class="details-header">
          <span
            class="status-badge"
            :style="{ backgroundColor: healthStore.statusColor + '33', color: healthStore.statusColor }"
          >
            {{ healthStore.health.status.toUpperCase() }}
          </span>
          <button class="close-btn" @click="showDetails = false">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div class="metrics">
          <!-- Chat metrics -->
          <div class="section-label">Чат</div>
          <div class="metric">
            <span class="label">Полный цикл</span>
            <span class="value" :class="{ slow: (healthStore.health.totalRoundtripMs ?? 0) > 1000 }">
              {{ formatMs(healthStore.health.totalRoundtripMs) }}
            </span>
          </div>
          <div class="metric">
            <span class="label">gRPC</span>
            <span class="value">{{ formatMs(healthStore.health.apiToChatServiceMs) }}</span>
          </div>
          <div class="metric">
            <span class="label">Centrifugo</span>
            <span class="value" :class="{ error: !healthStore.health.centrifugoConnected }">
              {{ healthStore.health.centrifugoConnected ? 'Подключён' : 'Отключён' }}
            </span>
          </div>

          <!-- Voice metrics -->
          <template v-if="healthStore.health.voiceCheckEnabled">
            <div class="section-label">Звонки</div>
            <div class="metric">
              <span class="label">Статус</span>
              <span class="value" :class="{ error: healthStore.health.voiceStatus === 'ERROR', warn: healthStore.health.voiceStatus === 'DISABLED' }">
                {{ voiceStatusLabel }}
              </span>
            </div>
            <div class="metric" v-if="healthStore.health.voiceStatus === 'OK'">
              <span class="label">Создание конф.</span>
              <span class="value">{{ formatMs(healthStore.health.createConferenceMs) }}</span>
            </div>
            <div class="metric" v-if="healthStore.health.voiceStatus === 'OK'">
              <span class="label">Добавление (2 уч.)</span>
              <span class="value">{{ formatMs(healthStore.health.addParticipantsMs) }}</span>
            </div>
            <div class="metric" v-if="healthStore.health.voiceStatus === 'OK'">
              <span class="label">Итого</span>
              <span class="value" :class="{ slow: (healthStore.health.voiceTotalMs ?? 0) > 500 }">
                {{ formatMs(healthStore.health.voiceTotalMs) }}
              </span>
            </div>
          </template>

          <div class="metric">
            <span class="label">Последняя проверка</span>
            <span class="value">{{ formatTime(healthStore.health.lastCheckTime) }}</span>
          </div>
        </div>

        <div v-if="healthStore.health.consecutiveFailures > 0" class="failures">
          Ошибок подряд: {{ healthStore.health.consecutiveFailures }}
        </div>

        <div v-if="healthStore.health.errorMessage" class="error-msg">
          {{ healthStore.health.errorMessage }}
        </div>

        <div v-if="healthStore.health.voiceErrorMessage" class="error-msg">
          Звонки: {{ healthStore.health.voiceErrorMessage }}
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.system-status {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 8px;
}

.status-dot {
  position: relative;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  border: none;
  cursor: pointer;
  transition: transform 0.2s;
}

.status-dot:hover {
  transform: scale(1.2);
}

.pulse {
  position: absolute;
  inset: -4px;
  border-radius: 50%;
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%,
  100% {
    transform: scale(1);
    opacity: 1;
  }
  50% {
    transform: scale(1.8);
    opacity: 0;
  }
}

.details-panel {
  position: absolute;
  left: 60px;
  bottom: 0;
  width: 260px;
  background: #1e293b;
  border: 1px solid #334155;
  border-radius: 8px;
  padding: 12px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.4);
  z-index: 100;
}

.details-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.status-badge {
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.5px;
}

.close-btn {
  background: transparent;
  border: none;
  color: #94a3b8;
  cursor: pointer;
  padding: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.close-btn:hover {
  color: #f1f5f9;
}

.metrics {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.metric {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
}

.label {
  color: #94a3b8;
}

.value {
  color: #f1f5f9;
  font-weight: 500;
  font-family: monospace;
}

.value.slow {
  color: #eab308;
}

.value.error {
  color: #ef4444;
}

.value.warn {
  color: #eab308;
}

.section-label {
  color: #64748b;
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-top: 8px;
  margin-bottom: -4px;
}

.section-label:first-child {
  margin-top: 0;
}

.failures {
  margin-top: 8px;
  padding: 6px 8px;
  background: rgba(234, 179, 8, 0.1);
  border-radius: 4px;
  font-size: 11px;
  color: #eab308;
}

.error-msg {
  margin-top: 8px;
  padding: 8px;
  background: rgba(239, 68, 68, 0.1);
  border-radius: 4px;
  font-size: 11px;
  color: #fca5a5;
  word-break: break-word;
}

/* Transition */
.slide-fade-enter-active {
  transition: all 0.2s ease-out;
}

.slide-fade-leave-active {
  transition: all 0.15s ease-in;
}

.slide-fade-enter-from,
.slide-fade-leave-to {
  transform: translateX(-10px);
  opacity: 0;
}
</style>
