import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export type NetworkStatus = 'online' | 'offline' | 'slow'

export interface PendingMessage {
  id: string
  chatId: string
  content: string
  fileLinkIds?: string[]
  createdAt: Date
  retryCount: number
}

const PENDING_MESSAGES_KEY = 'pending_messages'
const MAX_RETRY_COUNT = 3

export const useNetworkStore = defineStore('network', () => {
  // Состояние сети
  const isOnline = ref(navigator.onLine)
  const connectionQuality = ref<NetworkStatus>('online')
  const lastOnlineAt = ref<Date | null>(null)
  const lastOfflineAt = ref<Date | null>(null)

  // WebSocket состояние
  const isWebSocketConnected = ref(false)
  const webSocketReconnectAttempts = ref(0)

  // Очередь pending сообщений
  const pendingMessages = ref<PendingMessage[]>([])
  const isSendingPending = ref(false)

  // Computed
  const hasConnection = computed(() => isOnline.value && connectionQuality.value !== 'offline')
  const hasPendingMessages = computed(() => pendingMessages.value.length > 0)
  const pendingCount = computed(() => pendingMessages.value.length)

  // Статус для UI
  const statusMessage = computed(() => {
    if (!isOnline.value) return 'Нет соединения с интернетом'
    if (!isWebSocketConnected.value) return 'Переподключение...'
    if (connectionQuality.value === 'slow') return 'Медленное соединение'
    if (hasPendingMessages.value) return `Отправка сообщений (${pendingCount.value})...`
    return ''
  })

  const statusType = computed<'error' | 'warning' | 'info' | 'none'>(() => {
    if (!isOnline.value) return 'error'
    if (!isWebSocketConnected.value) return 'warning'
    if (connectionQuality.value === 'slow') return 'warning'
    if (hasPendingMessages.value) return 'info'
    return 'none'
  })

  // Инициализация: загрузить pending messages из localStorage
  function init() {
    loadPendingMessages()

    // Слушаем события браузера
    window.addEventListener('online', handleOnline)
    window.addEventListener('offline', handleOffline)

    // Проверяем качество соединения если API доступно
    if ('connection' in navigator) {
      const connection = (navigator as any).connection
      connection?.addEventListener('change', updateConnectionQuality)
      updateConnectionQuality()
    }

    console.log('[NetworkStore] Initialized, online:', isOnline.value)
  }

  function cleanup() {
    window.removeEventListener('online', handleOnline)
    window.removeEventListener('offline', handleOffline)

    if ('connection' in navigator) {
      const connection = (navigator as any).connection
      connection?.removeEventListener('change', updateConnectionQuality)
    }
  }

  function handleOnline() {
    console.log('[NetworkStore] Browser went online')
    isOnline.value = true
    lastOnlineAt.value = new Date()

    // Пробуем отправить pending messages
    processPendingMessages()
  }

  function handleOffline() {
    console.log('[NetworkStore] Browser went offline')
    isOnline.value = false
    lastOfflineAt.value = new Date()
    connectionQuality.value = 'offline'
  }

  function updateConnectionQuality() {
    // Skip slow connection detection for localhost - it's always fast
    const hostname = window.location.hostname
    if (hostname === 'localhost' || hostname === '127.0.0.1' || hostname.startsWith('192.168.') || hostname.startsWith('10.')) {
      connectionQuality.value = 'online'
      return
    }

    if (!('connection' in navigator)) return

    const connection = (navigator as any).connection
    if (!connection) return

    const effectiveType = connection.effectiveType // '4g', '3g', '2g', 'slow-2g'
    const downlink = connection.downlink // Mbps

    // Only mark as slow for really bad connections
    if (effectiveType === 'slow-2g' || effectiveType === '2g' || downlink < 0.25) {
      connectionQuality.value = 'slow'
    } else {
      connectionQuality.value = 'online'
    }

    console.log('[NetworkStore] Connection quality:', connectionQuality.value,
      'effectiveType:', effectiveType, 'downlink:', downlink)
  }

  // WebSocket состояние
  function setWebSocketConnected(connected: boolean) {
    const wasConnected = isWebSocketConnected.value
    isWebSocketConnected.value = connected

    if (connected) {
      webSocketReconnectAttempts.value = 0
      // При восстановлении WS пробуем отправить pending
      if (!wasConnected) {
        processPendingMessages()
      }
    } else {
      webSocketReconnectAttempts.value++
    }
  }

  // Pending messages management
  function loadPendingMessages() {
    try {
      const stored = localStorage.getItem(PENDING_MESSAGES_KEY)
      if (stored) {
        const parsed = JSON.parse(stored) as PendingMessage[]
        pendingMessages.value = parsed.map(m => ({
          ...m,
          createdAt: new Date(m.createdAt)
        }))
        console.log('[NetworkStore] Loaded pending messages:', pendingMessages.value.length)
      }
    } catch (e) {
      console.warn('[NetworkStore] Failed to load pending messages:', e)
    }
  }

  function savePendingMessages() {
    try {
      localStorage.setItem(PENDING_MESSAGES_KEY, JSON.stringify(pendingMessages.value))
    } catch (e) {
      console.warn('[NetworkStore] Failed to save pending messages:', e)
    }
  }

  function addPendingMessage(chatId: string, content: string, fileLinkIds?: string[]): string {
    const id = `pending_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`

    const pending: PendingMessage = {
      id,
      chatId,
      content,
      fileLinkIds,
      createdAt: new Date(),
      retryCount: 0
    }

    pendingMessages.value.push(pending)
    savePendingMessages()

    console.log('[NetworkStore] Added pending message:', id)
    return id
  }

  function removePendingMessage(id: string) {
    const index = pendingMessages.value.findIndex(m => m.id === id)
    if (index !== -1) {
      pendingMessages.value.splice(index, 1)
      savePendingMessages()
      console.log('[NetworkStore] Removed pending message:', id)
    }
  }

  function getPendingMessagesForChat(chatId: string): PendingMessage[] {
    return pendingMessages.value.filter(m => m.chatId === chatId)
  }

  // Обработка очереди pending messages
  async function processPendingMessages() {
    if (!isOnline.value || isSendingPending.value || pendingMessages.value.length === 0) {
      return
    }

    isSendingPending.value = true
    console.log('[NetworkStore] Processing pending messages:', pendingMessages.value.length)

    // Копируем массив чтобы избежать мутации во время итерации
    const toProcess = [...pendingMessages.value]

    for (const pending of toProcess) {
      if (!isOnline.value) {
        console.log('[NetworkStore] Went offline during processing, stopping')
        break
      }

      try {
        // Импортируем api динамически чтобы избежать циклических зависимостей
        const { api } = await import('@/api/client')

        await api.sendMessage(pending.chatId, {
          content: pending.content,
          file_link_ids: pending.fileLinkIds
        })

        removePendingMessage(pending.id)
        console.log('[NetworkStore] Successfully sent pending message:', pending.id)

      } catch (error) {
        console.error('[NetworkStore] Failed to send pending message:', pending.id, error)

        // Увеличиваем счётчик retry
        const index = pendingMessages.value.findIndex(m => m.id === pending.id)
        if (index !== -1) {
          pendingMessages.value[index].retryCount++

          // Если превысили лимит - удаляем
          if (pendingMessages.value[index].retryCount >= MAX_RETRY_COUNT) {
            console.warn('[NetworkStore] Max retry count reached, removing message:', pending.id)
            removePendingMessage(pending.id)
          } else {
            savePendingMessages()
          }
        }
      }

      // Небольшая пауза между отправками
      await new Promise(resolve => setTimeout(resolve, 500))
    }

    isSendingPending.value = false
  }

  return {
    // State
    isOnline,
    connectionQuality,
    lastOnlineAt,
    lastOfflineAt,
    isWebSocketConnected,
    webSocketReconnectAttempts,
    pendingMessages,
    isSendingPending,

    // Computed
    hasConnection,
    hasPendingMessages,
    pendingCount,
    statusMessage,
    statusType,

    // Actions
    init,
    cleanup,
    setWebSocketConnected,
    addPendingMessage,
    removePendingMessage,
    getPendingMessagesForChat,
    processPendingMessages,
  }
})
