<script setup lang="ts">
import { onMounted, onUnmounted, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useNetworkStore } from '@/stores/network'
import { useVoiceStore } from '@/stores/voice'
import { api } from '@/api/client'
import NetworkStatusBar from '@/components/NetworkStatusBar.vue'
import IncomingCall from '@/components/voice/IncomingCall.vue'
import ConferenceView from '@/components/voice/ConferenceView.vue'

const router = useRouter()
const authStore = useAuthStore()
const networkStore = useNetworkStore()
const voiceStore = useVoiceStore()

// Set up global auth failure handler - redirect to login when session expires
api.setOnAuthFailure(() => {
  console.log('[Auth] Session expired, redirecting to login')
  authStore.user = null
  router.push('/login')
})

// Computed properties for voice UI
const showIncomingCall = computed(() => voiceStore.hasIncomingCall && !voiceStore.isInCall)
const showConference = computed(() => voiceStore.currentConference !== null && voiceStore.isInCall)

// DEBUG: Watch showConference changes
watch(showConference, (newVal, oldVal) => {
  console.log(`[App.vue] showConference changed from ${oldVal} to ${newVal}`)
  console.log('[App.vue] currentConference:', voiceStore.currentConference?.id)
  console.log('[App.vue] isInCall:', voiceStore.isInCall)
})

const incomingCallerName = computed(() => {
  if (voiceStore.incomingConferenceData) {
    return voiceStore.incomingConferenceData.name || 'Conference Call'
  }
  if (voiceStore.incomingCallData) {
    return voiceStore.incomingCallData.caller_display_name ||
           voiceStore.incomingCallData.caller_username ||
           'Unknown'
  }
  if (voiceStore.vertoIncomingCall) {
    return voiceStore.vertoIncomingCall.remoteName ||
           voiceStore.vertoIncomingCall.remoteNumber ||
           'Unknown'
  }
  return 'Unknown'
})

const isConferenceCall = computed(() => {
  // Check Centrifugo notification
  if (voiceStore.incomingConferenceData !== null) return true

  // Check Verto incoming call - if remoteNumber looks like a conference
  const vertoCall = voiceStore.vertoIncomingCall
  if (vertoCall?.remoteNumber) {
    const fsName = vertoCall.remoteNumber
    return fsName.startsWith('conf_') ||
           fsName.startsWith('adhoc_') ||
           fsName.startsWith('scheduled_') ||
           fsName.startsWith('private_')
  }

  return false
})

function handleAnswerCall() {
  if (voiceStore.incomingConferenceData) {
    voiceStore.answerConferenceCall()
  } else {
    voiceStore.answerCall()
  }
}

function handleRejectCall() {
  voiceStore.rejectCall()
}

function handleLeaveConference() {
  voiceStore.leaveConference()
}

function handleToggleMute() {
  voiceStore.toggleMute()
}

function handleMuteParticipant(userId: string, mute: boolean) {
  voiceStore.muteParticipant(userId, mute)
}

function handleKickParticipant(userId: string) {
  voiceStore.kickParticipant(userId)
}

function handleEndConference() {
  voiceStore.endConference()
}

onMounted(async () => {
  // Инициализируем отслеживание сети
  networkStore.init()

  await authStore.init()

  // Автоматически подключаемся к Verto для получения входящих вызовов
  if (authStore.user) {
    console.log('[App] User logged in, connecting to Verto for incoming calls...')
    voiceStore.initVerto().catch(err => {
      console.warn('[App] Failed to auto-connect to Verto:', err)
    })
  }
})

// Следим за изменениями пользователя (логин/логаут)
watch(() => authStore.user, (newUser, oldUser) => {
  if (newUser && !oldUser) {
    // Пользователь залогинился - подключаемся к Verto
    console.log('[App] User logged in (watch), connecting to Verto for incoming calls...')
    voiceStore.initVerto().catch(err => {
      console.warn('[App] Failed to connect to Verto after login:', err)
    })
  } else if (!newUser && oldUser) {
    // Пользователь вышел - отключаемся от Verto
    console.log('[App] User logged out, disconnecting from Verto...')
    voiceStore.disconnectVerto()
  }
})

onUnmounted(() => {
  networkStore.cleanup()
  voiceStore.disconnectVerto()
})
</script>

<template>
  <div class="min-h-screen bg-gray-100">
    <!-- Hidden audio element for Verto.js - must exist before calls are made -->
    <audio id="verto-audio" autoplay style="display: none;"></audio>

    <!-- Network status bar (shows when offline/slow/syncing) -->
    <NetworkStatusBar />

    <!-- Main content -->
    <router-view />

    <!-- Global voice components -->
    <!-- Incoming call overlay - Teleport to body for proper z-index stacking -->
    <Teleport to="body">
      <IncomingCall
        v-if="showIncomingCall"
        :caller-name="incomingCallerName"
        :is-conference="isConferenceCall"
        @answer="handleAnswerCall"
        @reject="handleRejectCall"
      />
    </Teleport>

    <!-- Conference view overlay -->
    <Teleport to="body">
      <div v-if="showConference" class="fixed inset-0 z-50">
        <ConferenceView
          :conference="voiceStore.currentConference"
          :participants="voiceStore.sortedParticipants"
          :current-user-id="authStore.user?.id"
          :is-muted="voiceStore.isMuted"
          :call-start-time="voiceStore.currentCallState?.startTime"
          @leave="handleLeaveConference"
          @toggle-mute="handleToggleMute"
          @mute-participant="handleMuteParticipant"
          @kick-participant="handleKickParticipant"
          @end-conference="handleEndConference"
        />
      </div>
    </Teleport>
  </div>
</template>
