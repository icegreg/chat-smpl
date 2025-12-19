<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import { useVoiceStore } from '@/stores/voice'
import { useAuthStore } from '@/stores/auth'
import type { ScheduledConference, ConferenceParticipant } from '@/types'

const props = defineProps<{
  chatId: string
}>()

const voiceStore = useVoiceStore()
const authStore = useAuthStore()

// Fetch conferences for this chat on mount
onMounted(() => {
  voiceStore.fetchChatConferences(props.chatId)
})

// Re-fetch when chatId changes
watch(() => props.chatId, (newChatId) => {
  voiceStore.fetchChatConferences(newChatId)
})

// Get conferences for this chat
const conferences = computed(() => voiceStore.getChatConferencesSync(props.chatId))

// Get the next upcoming conference
const nextEvent = computed<ScheduledConference | null>(() => {
  const now = new Date()
  const upcoming = conferences.value
    .filter(c => c.scheduled_at && new Date(c.scheduled_at) > now)
    .sort((a, b) => {
      const aTime = a.scheduled_at ? new Date(a.scheduled_at).getTime() : 0
      const bTime = b.scheduled_at ? new Date(b.scheduled_at).getTime() : 0
      return aTime - bTime
    })
  return upcoming[0] || null
})

// Current user's RSVP status for the next event
const myRSVP = computed(() => {
  if (!nextEvent.value || !authStore.user) return 'pending'
  const participant = nextEvent.value.participants?.find(
    (p: ConferenceParticipant) => p.user_id === authStore.user!.id
  )
  return participant?.rsvp_status || 'pending'
})

// Accepted participants
const acceptedParticipants = computed(() => {
  if (!nextEvent.value) return []
  return (nextEvent.value.participants || []).filter(
    (p: ConferenceParticipant) => p.rsvp_status === 'accepted'
  )
})

// Format scheduled time
function formatTime(dateString: string): string {
  const date = new Date(dateString)
  const now = new Date()
  const diffMs = date.getTime() - now.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMins / 60)
  const diffDays = Math.floor(diffHours / 24)

  if (diffMins < 0) return 'Started'
  if (diffMins < 60) return `in ${diffMins}m`
  if (diffHours < 24) return `in ${diffHours}h`
  if (diffDays === 1) return 'Tomorrow'
  return date.toLocaleDateString([], { month: 'short', day: 'numeric' })
}

// RSVP handlers
async function accept() {
  if (!nextEvent.value) return
  await voiceStore.updateRSVP(nextEvent.value.id, 'accepted')
}

async function decline() {
  if (!nextEvent.value) return
  await voiceStore.updateRSVP(nextEvent.value.id, 'declined')
}

// Join the conference
async function joinEvent() {
  if (!nextEvent.value) return
  await voiceStore.joinConference(nextEvent.value.id)
}

// Get avatar initials
function getInitials(name?: string): string {
  if (!name) return '?'
  return name.split(' ').map(n => n[0]).join('').slice(0, 2).toUpperCase()
}
</script>

<template>
  <div v-if="nextEvent" class="scheduled-event-widget">
    <!-- Event Info -->
    <div class="event-header">
      <svg class="calendar-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
      </svg>
      <div class="event-info">
        <span class="event-name">{{ nextEvent.name }}</span>
        <span class="event-time">{{ formatTime(nextEvent.scheduled_at!) }}</span>
      </div>
      <!-- Join button if accepted -->
      <button
        v-if="myRSVP === 'accepted'"
        class="join-btn"
        @click="joinEvent"
        title="Join event"
      >
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" />
        </svg>
      </button>
    </div>

    <!-- RSVP Buttons (if pending) -->
    <div v-if="myRSVP === 'pending'" class="rsvp-buttons">
      <button class="rsvp-btn accept" @click="accept">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
        </svg>
        Accept
      </button>
      <button class="rsvp-btn decline" @click="decline">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
        </svg>
        Decline
      </button>
    </div>

    <!-- Accepted Avatars (if already responded) -->
    <div v-else class="accepted-section">
      <span v-if="myRSVP === 'declined'" class="declined-badge">
        You declined
      </span>
      <div class="accepted-avatars">
        <div
          v-for="(participant, idx) in acceptedParticipants.slice(0, 5)"
          :key="participant.user_id"
          class="avatar"
          :style="{ zIndex: 5 - idx }"
          :title="participant.display_name || participant.username"
        >
          <img
            v-if="participant.avatar_url"
            :src="participant.avatar_url"
            :alt="participant.display_name || participant.username"
          />
          <span v-else class="initials">
            {{ getInitials(participant.display_name || participant.username) }}
          </span>
        </div>
        <span v-if="acceptedParticipants.length > 5" class="more-count">
          +{{ acceptedParticipants.length - 5 }}
        </span>
      </div>
      <span class="accepted-count">
        {{ nextEvent.accepted_count }} accepted
      </span>
    </div>
  </div>
</template>

<style scoped>
.scheduled-event-widget {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px;
  background: linear-gradient(135deg, #4f46e5 0%, #7c3aed 100%);
  border-radius: 12px;
  color: #fff;
}

.event-header {
  display: flex;
  align-items: center;
  gap: 10px;
}

.calendar-icon {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
}

.event-info {
  flex: 1;
  min-width: 0;
}

.event-name {
  display: block;
  font-weight: 600;
  font-size: 14px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.event-time {
  font-size: 12px;
  opacity: 0.85;
}

.join-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 50%;
  border: none;
  background: rgba(255, 255, 255, 0.2);
  color: #fff;
  cursor: pointer;
  transition: all 0.2s ease;
}

.join-btn:hover {
  background: rgba(255, 255, 255, 0.3);
  transform: scale(1.05);
}

.join-btn .icon {
  width: 18px;
  height: 18px;
}

.rsvp-buttons {
  display: flex;
  gap: 8px;
}

.rsvp-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 8px 12px;
  border-radius: 8px;
  border: none;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.rsvp-btn .icon {
  width: 16px;
  height: 16px;
}

.rsvp-btn.accept {
  background: rgba(34, 197, 94, 0.9);
  color: #fff;
}

.rsvp-btn.accept:hover {
  background: #22c55e;
}

.rsvp-btn.decline {
  background: rgba(255, 255, 255, 0.2);
  color: #fff;
}

.rsvp-btn.decline:hover {
  background: rgba(239, 68, 68, 0.9);
}

.accepted-section {
  display: flex;
  align-items: center;
  gap: 10px;
}

.declined-badge {
  font-size: 12px;
  padding: 4px 8px;
  background: rgba(239, 68, 68, 0.3);
  border-radius: 4px;
}

.accepted-avatars {
  display: flex;
  flex-direction: row-reverse;
  justify-content: flex-end;
}

.avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  border: 2px solid #4f46e5;
  background: #6366f1;
  margin-left: -8px;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
}

.avatar:last-child {
  margin-left: 0;
}

.avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar .initials {
  font-size: 10px;
  font-weight: 600;
}

.more-count {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.2);
  font-size: 10px;
  font-weight: 600;
  margin-left: -8px;
}

.accepted-count {
  font-size: 12px;
  opacity: 0.85;
  margin-left: auto;
}
</style>
