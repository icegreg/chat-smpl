import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { ConferenceHistory, ChatFile, ModeratorAction, Message } from '@/types'
import { api } from '@/api/client'

export type HistoryTab = 'events' | 'files'

export const useEventHistoryStore = defineStore('eventHistory', () => {
  // State
  const conferences = ref<ConferenceHistory[]>([])
  const selectedConference = ref<ConferenceHistory | null>(null)
  const conferenceMessages = ref<Message[]>([])
  const moderatorActions = ref<ModeratorAction[]>([])
  const chatFiles = ref<ChatFile[]>([])
  const loading = ref(false)
  const loadingDetail = ref(false)
  const loadingFiles = ref(false)
  const error = ref<string | null>(null)
  const totalConferences = ref(0)
  const totalFiles = ref(0)
  const activeTab = ref<HistoryTab>('events')

  // Computed
  const hasConferences = computed(() => conferences.value.length > 0)
  const hasFiles = computed(() => chatFiles.value.length > 0)

  // Load conference history for a chat
  async function loadConferenceHistory(chatId: string, limit = 20, offset = 0): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const result = await api.getChatConferenceHistory(chatId, limit, offset)
      conferences.value = result.conferences
      totalConferences.value = result.total
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load conference history'
      console.error('Failed to load conference history:', e)
    } finally {
      loading.value = false
    }
  }

  // Load detailed conference info
  async function loadConferenceDetails(conferenceId: string): Promise<void> {
    loadingDetail.value = true
    error.value = null
    try {
      const [history, messages] = await Promise.all([
        api.getConferenceHistory(conferenceId),
        api.getConferenceMessages(conferenceId),
      ])
      selectedConference.value = history
      conferenceMessages.value = messages.messages || []
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load conference details'
      console.error('Failed to load conference details:', e)
    } finally {
      loadingDetail.value = false
    }
  }

  // Load messages sent during a conference
  async function loadConferenceMessages(conferenceId: string): Promise<void> {
    try {
      const result = await api.getConferenceMessages(conferenceId)
      conferenceMessages.value = result.messages
    } catch (e) {
      console.error('Failed to load conference messages:', e)
      conferenceMessages.value = []
    }
  }

  // Load moderator actions (for moderators only)
  async function loadModeratorActions(conferenceId: string): Promise<void> {
    try {
      const result = await api.getModeratorActions(conferenceId)
      moderatorActions.value = result.actions
    } catch (e) {
      console.error('Failed to load moderator actions:', e)
      moderatorActions.value = []
    }
  }

  // Load chat files
  async function loadChatFiles(chatId: string, limit = 50, offset = 0): Promise<void> {
    loadingFiles.value = true
    error.value = null
    try {
      const result = await api.getChatFiles(chatId, limit, offset)
      chatFiles.value = result.files || []
      totalFiles.value = result.total
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load files'
      console.error('Failed to load chat files:', e)
    } finally {
      loadingFiles.value = false
    }
  }

  // Select a conference and load its details
  async function selectConference(conference: ConferenceHistory | null): Promise<void> {
    if (!conference) {
      clearSelection()
      return
    }
    selectedConference.value = conference
    await Promise.all([
      loadConferenceDetails(conference.id),
      loadModeratorActions(conference.id),
    ])
  }

  // Set active tab
  function setActiveTab(tab: HistoryTab): void {
    activeTab.value = tab
  }

  // Clear selection and related data
  function clearSelection(): void {
    selectedConference.value = null
    conferenceMessages.value = []
    moderatorActions.value = []
  }

  // Reset all data
  function reset(): void {
    conferences.value = []
    selectedConference.value = null
    conferenceMessages.value = []
    moderatorActions.value = []
    chatFiles.value = []
    loading.value = false
    error.value = null
    totalConferences.value = 0
    totalFiles.value = 0
  }

  return {
    // State
    conferences,
    selectedConference,
    conferenceMessages,
    moderatorActions,
    chatFiles,
    loading,
    loadingDetail,
    loadingFiles,
    error,
    totalConferences,
    totalFiles,
    activeTab,

    // Computed
    hasConferences,
    hasFiles,

    // Actions
    loadConferenceHistory,
    loadConferenceDetails,
    loadConferenceMessages,
    loadModeratorActions,
    loadChatFiles,
    selectConference,
    setActiveTab,
    clearSelection,
    reset,
  }
})
