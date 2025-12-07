import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { PresenceInfo, UserStatus } from '@/types'
import { api } from '@/api/client'

export const usePresenceStore = defineStore('presence', () => {
  const myPresence = ref<PresenceInfo | null>(null)
  const userPresences = ref<Map<string, PresenceInfo>>(new Map())
  const connectionId = ref<string | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  // Status colors mapping
  const statusColors: Record<UserStatus | 'offline', string> = {
    available: '#22c55e', // green
    busy: '#ef4444', // red
    away: '#eab308', // yellow
    dnd: '#a855f7', // purple
    offline: '#9ca3af', // gray
  }

  // Status labels
  const statusLabels: Record<UserStatus | 'offline', string> = {
    available: 'Available',
    busy: 'Busy',
    away: 'Away',
    dnd: 'Do Not Disturb',
    offline: 'Offline',
  }

  const myStatus = computed(() => myPresence.value?.status || 'available')
  const isOnline = computed(() => myPresence.value?.is_online || false)

  function generateConnectionId(): string {
    return `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
  }

  function getStatusColor(presence: PresenceInfo | null | undefined): string {
    if (!presence || !presence.is_online) {
      return statusColors.offline
    }
    return statusColors[presence.status] || statusColors.offline
  }

  function getStatusLabel(presence: PresenceInfo | null | undefined): string {
    if (!presence || !presence.is_online) {
      return statusLabels.offline
    }
    return statusLabels[presence.status] || statusLabels.offline
  }

  function getUserPresence(userId: string): PresenceInfo | null {
    return userPresences.value.get(userId) || null
  }

  async function setStatus(status: UserStatus) {
    loading.value = true
    error.value = null
    try {
      myPresence.value = await api.setPresenceStatus(status)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to set status'
      console.error('Failed to set status:', e)
    } finally {
      loading.value = false
    }
  }

  async function fetchMyPresence() {
    try {
      myPresence.value = await api.getMyPresence()
    } catch (e) {
      console.error('Failed to fetch presence:', e)
    }
  }

  async function fetchUsersPresence(userIds: string[]) {
    if (userIds.length === 0) return

    try {
      const { presences } = await api.getUsersPresence(userIds)
      for (const presence of presences) {
        userPresences.value.set(presence.user_id, presence)
      }
    } catch (e) {
      console.error('Failed to fetch users presence:', e)
    }
  }

  async function registerConnection() {
    if (connectionId.value) return // Already registered

    connectionId.value = generateConnectionId()
    try {
      myPresence.value = await api.registerConnection(connectionId.value)
      console.log('Presence connection registered:', connectionId.value)
    } catch (e) {
      console.error('Failed to register connection:', e)
      connectionId.value = null
    }
  }

  async function unregisterConnection() {
    if (!connectionId.value) return

    try {
      await api.unregisterConnection(connectionId.value)
      console.log('Presence connection unregistered:', connectionId.value)
    } catch (e) {
      console.error('Failed to unregister connection:', e)
    } finally {
      connectionId.value = null
    }
  }

  // Handle browser tab visibility change
  function setupVisibilityHandler() {
    document.addEventListener('visibilitychange', async () => {
      if (document.hidden) {
        // Tab became hidden - set away status (optional)
        // Could automatically set to 'away' here
      } else {
        // Tab became visible - refresh presence
        await fetchMyPresence()
      }
    })

    // Handle page unload
    window.addEventListener('beforeunload', () => {
      if (connectionId.value) {
        // Use sendBeacon for reliable unload tracking
        const data = JSON.stringify({ connection_id: connectionId.value })
        navigator.sendBeacon('/api/presence/disconnect', data)
      }
    })
  }

  function cleanup() {
    unregisterConnection()
    myPresence.value = null
    userPresences.value.clear()
    connectionId.value = null
  }

  return {
    myPresence,
    userPresences,
    loading,
    error,
    myStatus,
    isOnline,
    statusColors,
    statusLabels,
    getStatusColor,
    getStatusLabel,
    getUserPresence,
    setStatus,
    fetchMyPresence,
    fetchUsersPresence,
    registerConnection,
    unregisterConnection,
    setupVisibilityHandler,
    cleanup,
  }
})
