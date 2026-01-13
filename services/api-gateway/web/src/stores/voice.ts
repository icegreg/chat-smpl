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
  const incomingConferenceData = ref<ConferenceEvent | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  // Scheduled events state
  const scheduledConferences = ref<ScheduledConference[]>([])
  const chatConferences = ref<Map<string, ScheduledConference[]>>(new Map())
  const pendingReminders = ref<ConferenceReminderEvent[]>([])

  // Active conferences by chat (for UI indicators)
  const activeConferencesByChat = ref<Map<string, Conference>>(new Map())

  // Computed
  const isConnected = computed(() => verto.isConnected.value)
  const isInCall = computed(() => verto.isInCall.value)
  const currentCallState = computed(() => verto.currentCall.value)
  const hasIncomingCall = computed(() =>
    verto.hasIncomingCall.value ||
    incomingCallData.value !== null ||
    incomingConferenceData.value !== null
  )
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

  // Helper functions for active conferences by chat
  function hasActiveConference(chatId: string): boolean {
    return activeConferencesByChat.value.has(chatId)
  }

  function getActiveConference(chatId: string): Conference | null {
    return activeConferencesByChat.value.get(chatId) || null
  }

  // Load active conferences for UI indicators
  async function loadActiveConferences(): Promise<void> {
    try {
      const response = await api.getActiveConferences()
      activeConferencesByChat.value.clear()
      for (const conf of response.conferences) {
        if (conf.chat_id) {
          activeConferencesByChat.value.set(conf.chat_id, conf)
        }
      }
    } catch (e) {
      console.error('[VoiceStore] Failed to load active conferences:', e)
    }
  }

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

  // Cleanup function for browser close/refresh
  // Uses sendBeacon for reliable delivery even during page unload
  function cleanupOnUnload(): void {
    if (!currentConference.value) return

    const confId = currentConference.value.id
    // Get token directly from localStorage since store might not be accessible during unload
    const token = localStorage.getItem('access_token')

    if (!token) return

    // Use sendBeacon for reliable delivery during page unload
    // This is more reliable than fetch during beforeunload
    const url = `/api/voice/conferences/${confId}/leave`
    const blob = new Blob([JSON.stringify({})], { type: 'application/json' })

    // sendBeacon doesn't support custom headers, so we use a special endpoint
    // that accepts token as query param for unload scenarios
    const beaconUrl = `${url}?_token=${encodeURIComponent(token)}`

    try {
      navigator.sendBeacon(beaconUrl, blob)
      console.log('[VoiceStore] Sent leave beacon for conference:', confId)
    } catch (e) {
      console.error('[VoiceStore] Failed to send leave beacon:', e)
    }
  }

  // Setup browser unload handlers
  function setupUnloadHandlers(): void {
    if (typeof window === 'undefined') return

    // beforeunload - fires when user closes/refreshes page
    window.addEventListener('beforeunload', cleanupOnUnload)

    // pagehide - fires on mobile when switching apps
    window.addEventListener('pagehide', cleanupOnUnload)

    console.log('[VoiceStore] Unload handlers registered')
  }

  // Remove unload handlers (called on store cleanup)
  function _removeUnloadHandlers(): void {
    if (typeof window === 'undefined') return

    window.removeEventListener('beforeunload', cleanupOnUnload)
    window.removeEventListener('pagehide', cleanupOnUnload)
  }
  void _removeUnloadHandlers // Mark as intentionally unused but available for future use

  // Register Verto disconnect callback
  verto.onDisconnect(() => {
    console.log('[VoiceStore] Verto disconnected, cleaning up conference state')
    // If we were in a conference, try to leave via API
    if (currentConference.value) {
      const confId = currentConference.value.id
      // Fire and forget - best effort cleanup
      api.leaveConference(confId).catch(e => {
        console.warn('[VoiceStore] Failed to leave conference on disconnect:', e)
      })
      currentConference.value = null
      participants.value = []
    }
    activeCall.value = null
  })

  // Auto-setup unload handlers
  setupUnloadHandlers()

  // Conference operations
  async function createConference(name: string, chatId?: string): Promise<Conference | null> {
    try {
      loading.value = true
      error.value = null

      const conference = await api.createConference({ name, chat_id: chatId })
      currentConference.value = conference

      // Update active conferences map for UI indicators (for the creator)
      // Use a new Map to trigger Vue reactivity
      if (chatId && conference) {
        const newMap = new Map(activeConferencesByChat.value)
        newMap.set(chatId, conference)
        activeConferencesByChat.value = newMap
      }

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

      // Join via API with display name for system messages
      const authStore = useAuthStore()
      const displayName = authStore.user?.display_name || authStore.user?.username || ''
      await api.joinConference(conferenceId, { muted, display_name: displayName })

      // Get conference details
      currentConference.value = await api.getConference(conferenceId)

      // Update active conferences map for UI indicators
      // Use a new Map to trigger Vue reactivity
      if (currentConference.value?.chat_id) {
        const newMap = new Map(activeConferencesByChat.value)
        newMap.set(currentConference.value.chat_id, currentConference.value)
        activeConferencesByChat.value = newMap
      }

      // Load participants
      const result = await api.getConferenceParticipants(conferenceId)
      participants.value = result.participants

      // Make Verto call to conference using FreeSWITCH conference name
      // freeswitch_name already has the correct prefix (adhoc_chat_, scheduled_, etc.)
      const confDestination = currentConference.value.freeswitch_name
      console.log('[VoiceStore] Making Verto call to:', confDestination)
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
      const vertoIncoming = verto.incomingCall.value
      const fsName = vertoIncoming?.remoteNumber

      // Check if this is a conference call by looking at the remoteNumber (fsName)
      const isConferenceCall = fsName && (
        fsName.startsWith('conf_') ||
        fsName.startsWith('adhoc_') ||
        fsName.startsWith('scheduled_') ||
        fsName.startsWith('private_')
      )

      console.log('[VoiceStore] answerCall: Answering Verto call, fsName:', fsName, 'isConference:', isConferenceCall)

      const answered = verto.answerCall()
      if (!answered) {
        console.error('[VoiceStore] answerCall: verto.answerCall() returned false')
        return false
      }

      console.log('[VoiceStore] answerCall: Verto answered successfully')

      // If this is a conference call, load conference info FIRST
      // Only clear incoming states after successful load
      if (isConferenceCall && fsName) {
        try {
          console.log('[VoiceStore] Incoming conference call, loading conference:', fsName)
          const conference = await api.getConferenceByFSName(fsName)
          currentConference.value = conference as Conference

          // Load participants
          const result = await api.getConferenceParticipants(conference.id)
          participants.value = result.participants

          console.log('[VoiceStore] Conference loaded:', conference.name, 'participants:', participants.value.length)
        } catch (e) {
          console.error('[VoiceStore] Failed to load conference info:', e)
          error.value = 'Failed to load conference information'
          // Don't hangup - user is already connected via Verto
          // Just set a minimal conference state so UI doesn't break
          if (fsName) {
            currentConference.value = {
              id: 'unknown',
              name: 'Conference',
              freeswitch_name: fsName,
              status: 'active',
              created_at: new Date().toISOString(),
            } as Conference
          }
        }
      }

      // Clear all incoming call states to hide the popup
      // The handleDialogState callback will also clear these, but we do it here
      // to ensure the popup hides right away
      incomingCallData.value = null
      incomingConferenceData.value = null

      return true
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

    // Dismiss incoming conference if any
    if (incomingConferenceData.value) {
      incomingConferenceData.value = null
    }

    return verto.rejectCall()
  }

  // Answer incoming conference call (join the conference)
  async function answerConferenceCall(): Promise<boolean> {
    // Prevent double-answer
    if (loading.value) {
      console.warn('[VoiceStore] Already answering call, ignoring duplicate request')
      return false
    }

    // If there's an actual Verto incoming call from FreeSWITCH,
    // we should ANSWER it, not make a new call
    if (verto.incomingCall.value) {
      console.log('[VoiceStore] answerConferenceCall: Answering existing Verto incoming call')
      // Clear the Centrifugo notification
      incomingConferenceData.value = null
      // Use answerCall which properly handles Verto incoming calls
      return await answerCall()
    }

    // No Verto incoming call - this means we got Centrifugo notification
    // but FreeSWITCH hasn't called us yet. Wait for the real SIP call.
    if (!incomingConferenceData.value) {
      console.error('[VoiceStore] No incoming conference to answer')
      return false
    }

    console.log('[VoiceStore] answerConferenceCall: No Verto call yet, waiting for SIP call...')
    loading.value = true
    const conferenceId = incomingConferenceData.value.id

    try {
      // Wait up to 5 seconds for FreeSWITCH to send the real SIP call
      const waitForVertoCall = new Promise<boolean>((resolve) => {
        let attempts = 0
        const maxAttempts = 50 // 5 seconds (50 * 100ms)

        const checkInterval = setInterval(() => {
          attempts++

          // Check if Verto incoming call arrived
          if (verto.incomingCall.value) {
            console.log('[VoiceStore] Verto incoming call arrived after', attempts * 100, 'ms! Answering...')
            clearInterval(checkInterval)
            // Don't clear incomingConferenceData here - answerCall will do it
            resolve(true)
            return
          }

          // Timeout - FreeSWITCH didn't send SIP call, join via outbound call
          if (attempts >= maxAttempts) {
            console.warn('[VoiceStore] Timeout waiting for Verto call after 5s, joining via outbound call')
            clearInterval(checkInterval)
            resolve(false)
          }
        }, 100) // Check every 100ms
      })

      const hasVertoCall = await waitForVertoCall

      if (hasVertoCall) {
        // Answer the real SIP call
        incomingConferenceData.value = null
        return await answerCall()
      } else {
        // No SIP call arrived, join via outbound call (makeCall)
        incomingConferenceData.value = null
        return await joinConference(conferenceId)
      }
    } finally {
      loading.value = false
    }
  }

  // Dismiss incoming conference call without joining
  function dismissIncomingConference(): void {
    incomingConferenceData.value = null
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
  async function startChatCall(chatId: string, chatName?: string): Promise<boolean> {
    console.log('[VoiceStore] ========== startChatCall called ==========', { chatId, chatName })
    try {
      loading.value = true
      error.value = null

      // Get credentials and create conference in one call
      console.log('[VoiceStore] startChatCall: Calling API to create conference...')
      const result = await api.startChatCall(chatId, chatName)
      console.log('[VoiceStore] startChatCall: API returned conference:', result.conference.id)
      credentials.value = result.credentials

      // Connect to Verto FIRST before setting conference
      const connected = await verto.connect(result.credentials)
      if (!connected) {
        error.value = 'Failed to connect to voice server'
        console.error('[VoiceStore] Verto connection failed, clearing conference data')
        // Don't set currentConference if Verto failed
        return false
      }

      // Join conference via Verto using FreeSWITCH conference name
      const confDestination = result.conference.freeswitch_name
      console.log('[VoiceStore] startChatCall - Making Verto call to:', confDestination)
      const callStarted = verto.makeCall(confDestination)

      if (!callStarted) {
        error.value = 'Failed to start Verto call'
        console.error('[VoiceStore] Verto makeCall failed')
        return false
      }

      // NOW set conference data after Verto call started successfully
      currentConference.value = result.conference

      // Update active conferences map for UI indicators (for the creator)
      // Use a new Map to trigger Vue reactivity
      if (chatId && result.conference) {
        const newMap = new Map(activeConferencesByChat.value)
        newMap.set(chatId, result.conference)
        activeConferencesByChat.value = newMap
      }

      // Add current user to participants immediately
      const authStore = useAuthStore()
      const userId = authStore.user?.id || result.credentials.user_id
      if (userId) {
        participants.value = [{
          id: '', // Will be updated when server confirms
          conference_id: result.conference.id,
          user_id: userId,
          status: 'joined',
          is_muted: false,
          is_speaking: false,
          username: authStore.user?.username,
          display_name: authStore.user?.display_name || 'You',
        }]
        console.log('[VoiceStore] Added current user to participants:', participants.value)
      }

      return true
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to start call'
      console.error('[VoiceStore] Start chat call error:', e)
      // Clear any partial state
      currentConference.value = null
      participants.value = []
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

  function handleConferenceCreated(data: ConferenceEvent): void {
    const authStore = useAuthStore()
    const userId = authStore.user?.id

    // Update active conferences map for UI indicators
    // Use a new Map to trigger Vue reactivity
    if (data.chat_id) {
      const newMap = new Map(activeConferencesByChat.value)
      newMap.set(data.chat_id, data as Conference)
      activeConferencesByChat.value = newMap
    }

    // Don't show popup if:
    // 1. We're the creator of the conference
    // 2. We're already in a call/conference
    // 3. We're in the process of answering/joining (loading state)
    if (data.created_by === userId) {
      console.log('[VoiceStore] Ignoring conference.created - we are the creator')
      return
    }

    if (currentConference.value || verto.isInCall.value || loading.value) {
      console.log('[VoiceStore] Ignoring conference.created - already in call or joining, isInCall:', verto.isInCall.value, 'loading:', loading.value)
      return
    }

    // Don't show popup if we already have this conference pending
    if (incomingConferenceData.value?.id === data.id) {
      console.log('[VoiceStore] Ignoring duplicate conference.created event for:', data.id)
      return
    }

    console.log('[VoiceStore] Incoming conference call:', data)
    incomingConferenceData.value = data

    // Auto-dismiss after 30 seconds if no Verto call arrives and user doesn't answer
    // This prevents stale popups from hanging around
    setTimeout(() => {
      if (incomingConferenceData.value?.id === data.id) {
        console.log('[VoiceStore] Auto-dismissing stale conference notification after 30s:', data.id)
        incomingConferenceData.value = null
      }
    }, 30000)
  }

  function handleConferenceEnded(data: ConferenceEvent): void {
    // Remove from active conferences map
    // Use a new Map to trigger Vue reactivity
    if (data.chat_id) {
      const newMap = new Map(activeConferencesByChat.value)
      newMap.delete(data.chat_id)
      activeConferencesByChat.value = newMap
    }

    // Clear incoming conference popup if it matches
    if (incomingConferenceData.value?.id === data.id) {
      incomingConferenceData.value = null
    }

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
    incomingConferenceData,
    loading,
    error,

    // Scheduled events state
    scheduledConferences,
    chatConferences,
    pendingReminders,

    // Active conferences by chat (for UI indicators)
    activeConferencesByChat,

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
    answerConferenceCall,
    rejectCall,
    dismissIncomingConference,
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

    // Active conferences helpers
    hasActiveConference,
    getActiveConference,
    loadActiveConferences,
  }
})

// Expose store for E2E testing
if (typeof window !== 'undefined') {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  (window as any).__voiceStore = {
    get currentConference() {
      const store = useVoiceStore()
      return store.currentConference
    },
    get participants() {
      const store = useVoiceStore()
      return store.participants
    },
    get isInCall() {
      const store = useVoiceStore()
      return store.isInCall
    },
    get isConnected() {
      const store = useVoiceStore()
      return store.isConnected
    },
    get hasIncomingCall() {
      const store = useVoiceStore()
      return store.hasIncomingCall
    },
    get incomingConferenceData() {
      const store = useVoiceStore()
      return store.incomingConferenceData
    },
    get incomingCallData() {
      const store = useVoiceStore()
      return store.incomingCallData
    },
    get vertoIncomingCall() {
      const store = useVoiceStore()
      return store.vertoIncomingCall
    },
    get loading() {
      const store = useVoiceStore()
      return store.loading
    },
    get error() {
      const store = useVoiceStore()
      return store.error
    },
    get credentials() {
      const store = useVoiceStore()
      return store.credentials
    },
    // Methods for E2E testing
    async initVerto() {
      const store = useVoiceStore()
      return store.initVerto()
    },
    async startChatCall(chatId: string, chatName?: string) {
      const store = useVoiceStore()
      return store.startChatCall(chatId, chatName)
    },
    async answerConferenceCall() {
      const store = useVoiceStore()
      return store.answerConferenceCall()
    },
    async joinConference(conferenceId: string) {
      const store = useVoiceStore()
      return store.joinConference(conferenceId)
    },
    async leaveConference() {
      const store = useVoiceStore()
      return store.leaveConference()
    },
  }
}
