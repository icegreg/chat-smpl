import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type {
  Conference,
  VoiceParticipant,
  Call,
  VertoCredentials,
  VoiceEvent,
  ConferenceEvent,
  ParticipantEvent,
  CallEvent,
  ScheduledConference,
  ScheduleConferenceRequest,
  RSVPStatus,
  ConferenceRole,
  ScheduledConferenceEvent,
  RSVPUpdatedEvent,
  ParticipantRoleChangedEvent,
  ConferenceReminderEvent,
} from '@/types'
import { api } from '@/api/client'
import { useVerto } from '@/composables/useVerto'
import { useAuthStore } from './auth'

export const useVoiceStore = defineStore('voice', () => {
  // Verto composable
  const verto = useVerto()

  // State
  const credentials = ref<VertoCredentials | null>(null)
  const currentConference = ref<Conference | null>(null)
  const participants = ref<VoiceParticipant[]>([])
  const activeCall = ref<Call | null>(null)
  const incomingCallData = ref<CallEvent | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  // Scheduled events state
  const scheduledConferences = ref<ScheduledConference[]>([])
  const chatConferences = ref<Map<string, ScheduledConference[]>>(new Map())
  const pendingReminders = ref<ConferenceReminderEvent[]>([])

  // Computed
  const isConnected = computed(() => verto.isConnected.value)
  const isInCall = computed(() => verto.isInCall.value)
  const currentCallState = computed(() => verto.currentCall.value)
  const hasIncomingCall = computed(() => verto.hasIncomingCall.value || incomingCallData.value !== null)
  const isMuted = computed(() => verto.currentCall.value?.isMuted ?? false)

  const sortedParticipants = computed(() => {
    return [...participants.value].sort((a, b) => {
      // Speaking users first
      if (a.is_speaking && !b.is_speaking) return -1
      if (!a.is_speaking && b.is_speaking) return 1
      // Then by join time
      const aTime = a.joined_at ? new Date(a.joined_at).getTime() : 0
      const bTime = b.joined_at ? new Date(b.joined_at).getTime() : 0
      return aTime - bTime
    })
  })

  // Scheduled events computed
  const upcomingConferences = computed(() => {
    const now = new Date()
    return scheduledConferences.value
      .filter(c => c.scheduled_at && new Date(c.scheduled_at) > now)
      .sort((a, b) => {
        const aTime = a.scheduled_at ? new Date(a.scheduled_at).getTime() : 0
        const bTime = b.scheduled_at ? new Date(b.scheduled_at).getTime() : 0
        return aTime - bTime
      })
  })

  const hasUpcomingReminders = computed(() => pendingReminders.value.length > 0)

  // Initialize Verto connection
  async function initVerto(): Promise<boolean> {
    if (verto.isConnected.value) return true

    try {
      loading.value = true
      error.value = null

      // Get credentials from API
      credentials.value = await api.getVertoCredentials()

      // Connect to Verto
      const success = await verto.connect(credentials.value)
      if (!success) {
        error.value = 'Failed to connect to voice server'
        return false
      }

      return true
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to initialize voice'
      console.error('[VoiceStore] Init error:', e)
      return false
    } finally {
      loading.value = false
    }
  }

  // Disconnect from Verto
  function disconnectVerto(): void {
    verto.disconnect()
    credentials.value = null
    currentConference.value = null
    participants.value = []
    activeCall.value = null
  }

  // Conference operations
  async function createConference(name: string, chatId?: string): Promise<Conference | null> {
    try {
      loading.value = true
      error.value = null

      const conference = await api.createConference({ name, chat_id: chatId })
      currentConference.value = conference

      return conference
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to create conference'
      console.error('[VoiceStore] Create conference error:', e)
      return null
    } finally {
      loading.value = false
    }
  }

  async function joinConference(conferenceId: string, muted = false): Promise<boolean> {
    try {
      loading.value = true
      error.value = null

      // Ensure Verto is connected
      if (!verto.isConnected.value) {
        const connected = await initVerto()
        if (!connected) return false
      }

      // Join via API
      await api.joinConference(conferenceId, { muted })

      // Get conference details
      currentConference.value = await api.getConference(conferenceId)

      // Load participants
      const result = await api.getConferenceParticipants(conferenceId)
      participants.value = result.participants

      // Make Verto call to conference
      const confDestination = `conference-${conferenceId}`
      verto.makeCall(confDestination)

      return true
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to join conference'
      console.error('[VoiceStore] Join conference error:', e)
      return false
    } finally {
      loading.value = false
    }
  }

  async function leaveConference(): Promise<void> {
    if (!currentConference.value) return

    try {
      // Hangup Verto call
      verto.hangup()

      // Leave via API
      await api.leaveConference(currentConference.value.id)

      currentConference.value = null
      participants.value = []
    } catch (e) {
      console.error('[VoiceStore] Leave conference error:', e)
    }
  }

  async function muteParticipant(userId: string, mute: boolean): Promise<void> {
    if (!currentConference.value) return

    try {
      await api.muteParticipant(currentConference.value.id, userId, mute)
    } catch (e) {
      console.error('[VoiceStore] Mute participant error:', e)
    }
  }

  async function kickParticipant(userId: string): Promise<void> {
    if (!currentConference.value) return

    try {
      await api.kickParticipant(currentConference.value.id, userId)
    } catch (e) {
      console.error('[VoiceStore] Kick participant error:', e)
    }
  }

  async function endConference(): Promise<void> {
    if (!currentConference.value) return

    try {
      verto.hangup()
      await api.endConference(currentConference.value.id)
      currentConference.value = null
      participants.value = []
    } catch (e) {
      console.error('[VoiceStore] End conference error:', e)
    }
  }

  // Call operations
  async function initiateCall(calleeId: string, chatId?: string): Promise<boolean> {
    try {
      loading.value = true
      error.value = null

      // Ensure Verto is connected
      if (!verto.isConnected.value) {
        const connected = await initVerto()
        if (!connected) return false
      }

      // Initiate call via API
      activeCall.value = await api.initiateCall({ callee_id: calleeId, chat_id: chatId })

      // Make Verto call
      const callDestination = `user-${calleeId}`
      verto.makeCall(callDestination)

      return true
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to initiate call'
      console.error('[VoiceStore] Initiate call error:', e)
      return false
    } finally {
      loading.value = false
    }
  }

  async function answerCall(): Promise<boolean> {
    if (!incomingCallData.value) {
      // Answer Verto incoming call
      return verto.answerCall()
    }

    try {
      loading.value = true

      // Ensure Verto is connected
      if (!verto.isConnected.value) {
        const connected = await initVerto()
        if (!connected) return false
      }

      // Answer via API
      activeCall.value = await api.answerCall(incomingCallData.value.id)
      incomingCallData.value = null

      // Answer Verto call
      verto.answerCall()

      return true
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to answer call'
      console.error('[VoiceStore] Answer call error:', e)
      return false
    } finally {
      loading.value = false
    }
  }

  function rejectCall(): boolean {
    if (incomingCallData.value) {
      // Reject via API (fire and forget)
      api.hangupCall(incomingCallData.value.id).catch(console.error)
      incomingCallData.value = null
    }

    return verto.rejectCall()
  }

  async function hangupCall(): Promise<void> {
    if (activeCall.value) {
      try {
        await api.hangupCall(activeCall.value.id)
      } catch (e) {
        console.error('[VoiceStore] Hangup API error:', e)
      }
      activeCall.value = null
    }

    verto.hangup()
  }

  function toggleMute(): boolean {
    const authStore = useAuthStore()
    const userId = authStore.user?.id

    // Toggle local mute
    const newMuteState = verto.toggleMute()

    // Also update on server if in conference
    if (currentConference.value && userId) {
      api.muteParticipant(currentConference.value.id, userId, newMuteState).catch(console.error)
    }

    return newMuteState
  }

  // Quick call from chat
  async function startChatCall(chatId: string): Promise<boolean> {
    try {
      loading.value = true
      error.value = null

      // Get credentials and create conference in one call
      const result = await api.startChatCall(chatId)
      credentials.value = result.credentials
      currentConference.value = result.conference

      // Connect to Verto
      const connected = await verto.connect(result.credentials)
      if (!connected) {
        error.value = 'Failed to connect to voice server'
        return false
      }

      // Join conference via Verto
      const confDestination = `conference-${result.conference.id}`
      verto.makeCall(confDestination)

      return true
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to start call'
      console.error('[VoiceStore] Start chat call error:', e)
      return false
    } finally {
      loading.value = false
    }
  }

  // ==========================================
  // Scheduled Events Operations
  // ==========================================

  // Schedule a new conference
  async function scheduleConference(
    data: ScheduleConferenceRequest
  ): Promise<ScheduledConference | null> {
    try {
      loading.value = true
      error.value = null

      const conference = await api.scheduleConference(data)
      scheduledConferences.value.push(conference)

      // Also add to chat conferences if linked to a chat
      if (conference.chat_id) {
        const existing = chatConferences.value.get(conference.chat_id) || []
        chatConferences.value.set(conference.chat_id, [...existing, conference])
      }

      return conference
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to schedule conference'
      console.error('[VoiceStore] Schedule conference error:', e)
      return null
    } finally {
      loading.value = false
    }
  }

  // Create ad-hoc conference from chat
  async function createAdHocFromChat(
    chatId: string,
    participantIds?: string[]
  ): Promise<ScheduledConference | null> {
    try {
      loading.value = true
      error.value = null

      const conference = await api.createAdHocFromChat({
        chat_id: chatId,
        participant_user_ids: participantIds,
      })

      // Add to chat conferences
      const existing = chatConferences.value.get(chatId) || []
      chatConferences.value.set(chatId, [...existing, conference])

      return conference
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to create ad-hoc conference'
      console.error('[VoiceStore] Create ad-hoc from chat error:', e)
      return null
    } finally {
      loading.value = false
    }
  }

  // Create quick ad-hoc conference (without chat)
  async function createQuickAdHoc(): Promise<ScheduledConference | null> {
    try {
      loading.value = true
      error.value = null

      const conference = await api.createQuickAdHoc()
      return conference
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to create quick ad-hoc'
      console.error('[VoiceStore] Create quick ad-hoc error:', e)
      return null
    } finally {
      loading.value = false
    }
  }

  // Update RSVP status
  async function updateRSVP(conferenceId: string, status: RSVPStatus): Promise<boolean> {
    try {
      loading.value = true
      error.value = null

      await api.updateRSVP(conferenceId, status)

      // Update local state
      const conf = scheduledConferences.value.find(c => c.id === conferenceId)
      if (conf) {
        const authStore = useAuthStore()
        const participant = conf.participants?.find(p => p.user_id === authStore.user?.id)
        if (participant) {
          participant.rsvp_status = status
        }
        // Update counts
        if (status === 'accepted') {
          conf.accepted_count++
        } else if (status === 'declined') {
          conf.declined_count++
        }
      }

      return true
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to update RSVP'
      console.error('[VoiceStore] Update RSVP error:', e)
      return false
    } finally {
      loading.value = false
    }
  }

  // Update participant role
  async function updateParticipantRole(
    conferenceId: string,
    userId: string,
    newRole: ConferenceRole
  ): Promise<boolean> {
    try {
      loading.value = true
      error.value = null

      await api.updateParticipantRole(conferenceId, userId, newRole)

      // Update local state
      const conf = scheduledConferences.value.find(c => c.id === conferenceId)
      if (conf) {
        const participant = conf.participants?.find(p => p.user_id === userId)
        if (participant) {
          participant.role = newRole
        }
      }

      return true
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to update participant role'
      console.error('[VoiceStore] Update participant role error:', e)
      return false
    } finally {
      loading.value = false
    }
  }

  // Add participants to conference
  async function addConferenceParticipants(
    conferenceId: string,
    userIds: string[],
    defaultRole?: ConferenceRole
  ): Promise<boolean> {
    try {
      loading.value = true
      error.value = null

      await api.addConferenceParticipants(conferenceId, userIds, defaultRole)

      // Refresh conference to get updated participants
      const updated = await api.getScheduledConference(conferenceId)
      const idx = scheduledConferences.value.findIndex(c => c.id === conferenceId)
      if (idx !== -1) {
        scheduledConferences.value[idx] = updated
      }

      return true
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to add participants'
      console.error('[VoiceStore] Add participants error:', e)
      return false
    } finally {
      loading.value = false
    }
  }

  // Remove participant from conference
  async function removeConferenceParticipant(
    conferenceId: string,
    userId: string
  ): Promise<boolean> {
    try {
      loading.value = true
      error.value = null

      await api.removeConferenceParticipant(conferenceId, userId)

      // Update local state
      const conf = scheduledConferences.value.find(c => c.id === conferenceId)
      if (conf && conf.participants) {
        conf.participants = conf.participants.filter(p => p.user_id !== userId)
      }

      return true
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to remove participant'
      console.error('[VoiceStore] Remove participant error:', e)
      return false
    } finally {
      loading.value = false
    }
  }

  // Fetch scheduled conferences for current user
  async function fetchScheduledConferences(upcomingOnly = true): Promise<void> {
    try {
      loading.value = true
      error.value = null

      const result = await api.listScheduledConferences(upcomingOnly)
      scheduledConferences.value = result.conferences
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch scheduled conferences'
      console.error('[VoiceStore] Fetch scheduled conferences error:', e)
    } finally {
      loading.value = false
    }
  }

  // Fetch conferences for a specific chat
  async function fetchChatConferences(chatId: string, upcomingOnly = true): Promise<void> {
    try {
      const result = await api.getChatConferences(chatId, upcomingOnly)
      chatConferences.value.set(chatId, result.conferences)
    } catch (e) {
      console.error('[VoiceStore] Fetch chat conferences error:', e)
    }
  }

  // Get conferences for a chat (from cache or fetch)
  function getChatConferencesSync(chatId: string): ScheduledConference[] {
    return chatConferences.value.get(chatId) || []
  }

  // Cancel a conference
  async function cancelConference(conferenceId: string, cancelSeries = false): Promise<boolean> {
    try {
      loading.value = true
      error.value = null

      await api.cancelConference(conferenceId, cancelSeries)

      // Remove from local state
      scheduledConferences.value = scheduledConferences.value.filter(c => c.id !== conferenceId)

      // Remove from chat conferences
      chatConferences.value.forEach((conferences, chatId) => {
        const filtered = conferences.filter(c => c.id !== conferenceId)
        if (filtered.length !== conferences.length) {
          chatConferences.value.set(chatId, filtered)
        }
      })

      return true
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to cancel conference'
      console.error('[VoiceStore] Cancel conference error:', e)
      return false
    } finally {
      loading.value = false
    }
  }

  // Clear a reminder from pending list
  function dismissReminder(conferenceId: string): void {
    pendingReminders.value = pendingReminders.value.filter(
      r => r.conference_id !== conferenceId
    )
  }

  // Handle voice events from WebSocket
  function handleVoiceEvent(event: VoiceEvent): void {
    switch (event.type) {
      case 'conference.created':
        handleConferenceCreated(event.data as ConferenceEvent)
        break
      case 'conference.ended':
        handleConferenceEnded(event.data as ConferenceEvent)
        break
      case 'participant.joined':
        handleParticipantJoined(event.data as ParticipantEvent)
        break
      case 'participant.left':
        handleParticipantLeft(event.data as ParticipantEvent)
        break
      case 'participant.muted':
        handleParticipantMuted(event.data as ParticipantEvent)
        break
      case 'participant.speaking':
        handleParticipantSpeaking(event.data as ParticipantEvent)
        break
      case 'call.initiated':
        handleCallInitiated(event.data as CallEvent)
        break
      case 'call.answered':
        handleCallAnswered(event.data as CallEvent)
        break
      case 'call.ended':
        handleCallEnded(event.data as CallEvent)
        break
      // Scheduled events
      case 'conference.scheduled':
        handleConferenceScheduled(event.data as ScheduledConferenceEvent)
        break
      case 'conference.rsvp_updated':
        handleRSVPUpdated(event.data as RSVPUpdatedEvent)
        break
      case 'participant.role_changed':
        handleParticipantRoleChanged(event.data as ParticipantRoleChangedEvent)
        break
      case 'conference.reminder':
        handleConferenceReminder(event.data as ConferenceReminderEvent)
        break
    }
  }

  function handleConferenceCreated(_data: ConferenceEvent): void {
    // If this is for a chat we're in, show notification
    // TODO: Show notification about new conference
  }

  function handleConferenceEnded(data: ConferenceEvent): void {
    if (currentConference.value?.id === data.id) {
      currentConference.value = null
      participants.value = []
      verto.hangup()
    }
  }

  function handleParticipantJoined(data: ParticipantEvent): void {
    if (currentConference.value?.id !== data.conference_id) return

    // Add or update participant
    const idx = participants.value.findIndex(p => p.user_id === data.user_id)
    if (idx === -1) {
      participants.value.push({
        id: data.id,
        conference_id: data.conference_id,
        user_id: data.user_id,
        status: data.status,
        is_muted: data.is_muted,
        is_speaking: data.is_speaking,
        username: data.username,
        display_name: data.display_name,
      })
    } else {
      participants.value[idx] = {
        ...participants.value[idx],
        status: data.status,
        is_muted: data.is_muted,
        is_speaking: data.is_speaking,
      }
    }
  }

  function handleParticipantLeft(data: ParticipantEvent): void {
    if (currentConference.value?.id !== data.conference_id) return
    participants.value = participants.value.filter(p => p.user_id !== data.user_id)
  }

  function handleParticipantMuted(data: ParticipantEvent): void {
    if (currentConference.value?.id !== data.conference_id) return
    const participant = participants.value.find(p => p.user_id === data.user_id)
    if (participant) {
      participant.is_muted = data.is_muted
    }
  }

  function handleParticipantSpeaking(data: ParticipantEvent): void {
    if (currentConference.value?.id !== data.conference_id) return
    const participant = participants.value.find(p => p.user_id === data.user_id)
    if (participant) {
      participant.is_speaking = data.is_speaking
    }
  }

  function handleCallInitiated(data: CallEvent): void {
    const authStore = useAuthStore()
    // Check if this call is for us
    if (data.callee_id === authStore.user?.id) {
      incomingCallData.value = data
    }
  }

  function handleCallAnswered(data: CallEvent): void {
    if (activeCall.value?.id === data.id) {
      activeCall.value = {
        ...activeCall.value,
        status: data.status,
      }
    }
    if (incomingCallData.value?.id === data.id) {
      incomingCallData.value = null
    }
  }

  function handleCallEnded(data: CallEvent): void {
    if (activeCall.value?.id === data.id) {
      activeCall.value = null
    }
    if (incomingCallData.value?.id === data.id) {
      incomingCallData.value = null
    }
  }

  // Scheduled events handlers
  function handleConferenceScheduled(data: ScheduledConferenceEvent): void {
    // Refresh scheduled conferences to include the new one
    fetchScheduledConferences()
    // If linked to a chat, also refresh chat conferences
    if (data.chat_id) {
      fetchChatConferences(data.chat_id)
    }
  }

  function handleRSVPUpdated(data: RSVPUpdatedEvent): void {
    const conf = scheduledConferences.value.find(c => c.id === data.conference_id)
    if (conf && conf.participants) {
      const participant = conf.participants.find(p => p.user_id === data.user_id)
      if (participant) {
        // Update counts based on old and new status
        const oldStatus = participant.rsvp_status
        if (oldStatus === 'accepted') conf.accepted_count--
        if (oldStatus === 'declined') conf.declined_count--
        if (data.rsvp_status === 'accepted') conf.accepted_count++
        if (data.rsvp_status === 'declined') conf.declined_count++
        participant.rsvp_status = data.rsvp_status
      }
    }
  }

  function handleParticipantRoleChanged(data: ParticipantRoleChangedEvent): void {
    const conf = scheduledConferences.value.find(c => c.id === data.conference_id)
    if (conf && conf.participants) {
      const participant = conf.participants.find(p => p.user_id === data.user_id)
      if (participant) {
        participant.role = data.new_role
      }
    }
  }

  function handleConferenceReminder(data: ConferenceReminderEvent): void {
    // Add to pending reminders for UI notification
    pendingReminders.value.push(data)
  }

  return {
    // State
    credentials,
    currentConference,
    participants,
    activeCall,
    incomingCallData,
    loading,
    error,

    // Scheduled events state
    scheduledConferences,
    chatConferences,
    pendingReminders,

    // Computed
    isConnected,
    isInCall,
    currentCallState,
    hasIncomingCall,
    isMuted,
    sortedParticipants,

    // Scheduled events computed
    upcomingConferences,
    hasUpcomingReminders,

    // Verto state (expose for UI)
    vertoIncomingCall: verto.incomingCall,

    // Methods
    initVerto,
    disconnectVerto,
    createConference,
    joinConference,
    leaveConference,
    muteParticipant,
    kickParticipant,
    endConference,
    initiateCall,
    answerCall,
    rejectCall,
    hangupCall,
    toggleMute,
    startChatCall,
    handleVoiceEvent,

    // Scheduled events methods
    scheduleConference,
    createAdHocFromChat,
    createQuickAdHoc,
    updateRSVP,
    updateParticipantRole,
    addConferenceParticipants,
    removeConferenceParticipant,
    fetchScheduledConferences,
    fetchChatConferences,
    getChatConferencesSync,
    cancelConference,
    dismissReminder,
  }
})
