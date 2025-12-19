<script setup lang="ts">
import { onMounted, onUnmounted, computed } from 'vue'
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

const incomingCallerName = computed(() => {
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

function handleAnswerCall() {
  voiceStore.answerCall()
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
})

onUnmounted(() => {
  networkStore.cleanup()
  voiceStore.disconnectVerto()
})
</script>

<template>
  <div class="min-h-screen bg-gray-100">
    <!-- Network status bar (shows when offline/slow/syncing) -->
    <NetworkStatusBar />

    <!-- Main content -->
    <router-view />

    <!-- Global voice components -->
    <!-- Incoming call overlay -->
    <IncomingCall
      v-if="showIncomingCall"
      :caller-name="incomingCallerName"
      @answer="handleAnswerCall"
      @reject="handleRejectCall"
    />

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
