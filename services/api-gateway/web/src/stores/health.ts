import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export type SystemHealthStatus = 'ok' | 'degraded' | 'down' | 'unknown'

export interface SystemHealth {
  status: SystemHealthStatus
  lastCheckTime: Date | null
  totalRoundtripMs: number | null
  apiToChatServiceMs: number | null
  consecutiveFailures: number
  errorMessage: string | null
  centrifugoConnected: boolean
  // Voice metrics
  voiceCheckEnabled: boolean
  voiceStatus: string | null // 'OK' | 'ERROR' | 'DISABLED'
  createConferenceMs: number | null
  addParticipantsMs: number | null
  endConferenceMs: number | null
  voiceTotalMs: number | null
  voiceErrorMessage: string | null
}

interface ApiHealthResponse {
  status: string
  last_check_time: string
  total_roundtrip_ms: number
  api_to_chat_service_ms?: number
  consecutive_failures: number
  error_message?: string
  failed_stage?: string
  centrifugo_connected: boolean
  // Voice metrics
  voice_check_enabled: boolean
  voice_status?: string
  create_conference_ms?: number
  add_participants_ms?: number
  end_conference_ms?: number
  voice_total_ms?: number
  voice_error_message?: string
}

export const useHealthStore = defineStore('health', () => {
  const health = ref<SystemHealth>({
    status: 'unknown',
    lastCheckTime: null,
    totalRoundtripMs: null,
    apiToChatServiceMs: null,
    consecutiveFailures: 0,
    errorMessage: null,
    centrifugoConnected: false,
    voiceCheckEnabled: false,
    voiceStatus: null,
    createConferenceMs: null,
    addParticipantsMs: null,
    endConferenceMs: null,
    voiceTotalMs: null,
    voiceErrorMessage: null,
  })

  const isHealthy = computed(() => health.value.status === 'ok')
  const isDegraded = computed(() => health.value.status === 'degraded')
  const isDown = computed(() => health.value.status === 'down')

  const statusColor = computed(() => {
    switch (health.value.status) {
      case 'ok':
        return '#22c55e' // green-500
      case 'degraded':
        return '#eab308' // yellow-500
      case 'down':
        return '#ef4444' // red-500
      default:
        return '#6b7280' // gray-500
    }
  })

  const statusLabel = computed(() => {
    switch (health.value.status) {
      case 'ok':
        return 'Система работает'
      case 'degraded':
        return 'Замедление'
      case 'down':
        return 'Проблемы'
      default:
        return 'Неизвестно'
    }
  })

  let pollingInterval: ReturnType<typeof setInterval> | null = null

  async function fetchHealth() {
    try {
      const response = await fetch('/api/health/system')

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`)
      }

      const data: ApiHealthResponse = await response.json()

      health.value = {
        status: data.status.toLowerCase() as SystemHealthStatus,
        lastCheckTime: new Date(data.last_check_time),
        totalRoundtripMs: data.total_roundtrip_ms,
        apiToChatServiceMs: data.api_to_chat_service_ms ?? null,
        consecutiveFailures: data.consecutive_failures,
        errorMessage: data.error_message || null,
        centrifugoConnected: data.centrifugo_connected,
        voiceCheckEnabled: data.voice_check_enabled,
        voiceStatus: data.voice_status || null,
        createConferenceMs: data.create_conference_ms ?? null,
        addParticipantsMs: data.add_participants_ms ?? null,
        endConferenceMs: data.end_conference_ms ?? null,
        voiceTotalMs: data.voice_total_ms ?? null,
        voiceErrorMessage: data.voice_error_message || null,
      }
    } catch (error) {
      console.error('[HealthStore] Failed to fetch health:', error)
      health.value.status = 'unknown'
      health.value.errorMessage = 'Не удалось получить статус системы'
    }
  }

  function startPolling(intervalMs = 10000) {
    // Fetch immediately
    fetchHealth()

    // Then poll at interval
    if (pollingInterval) {
      clearInterval(pollingInterval)
    }
    pollingInterval = setInterval(fetchHealth, intervalMs)
  }

  function stopPolling() {
    if (pollingInterval) {
      clearInterval(pollingInterval)
      pollingInterval = null
    }
  }

  return {
    health,
    isHealthy,
    isDegraded,
    isDown,
    statusColor,
    statusLabel,
    fetchHealth,
    startPolling,
    stopPolling,
  }
})
