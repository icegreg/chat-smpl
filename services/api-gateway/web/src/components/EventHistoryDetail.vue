<script setup lang="ts">
import { ref } from 'vue'
import type { ConferenceHistory, Message, ModeratorAction } from '@/types'

defineProps<{
  conference: ConferenceHistory
  messages: Message[]
  moderatorActions: ModeratorAction[]
  isModerator: boolean
  loading: boolean
}>()

defineEmits<{
  back: []
}>()

const activeSection = ref<'participants' | 'messages' | 'actions'>('participants')

// Format helpers
function formatTime(dateStr: string): string {
  return new Date(dateStr).toLocaleTimeString(undefined, {
    hour: '2-digit',
    minute: '2-digit'
  })
}

function formatDateTime(dateStr: string): string {
  return new Date(dateStr).toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

function formatDuration(joinedAt: string, leftAt?: string): string {
  const start = new Date(joinedAt).getTime()
  const end = leftAt ? new Date(leftAt).getTime() : Date.now()
  const minutes = Math.floor((end - start) / 60000)
  if (minutes < 1) return '< 1 min'
  if (minutes < 60) return `${minutes} min`
  const hours = Math.floor(minutes / 60)
  const remainingMinutes = minutes % 60
  return remainingMinutes > 0 ? `${hours}h ${remainingMinutes}m` : `${hours}h`
}

function getStatusColor(status: string): string {
  switch (status) {
    case 'kicked':
      return 'text-red-600'
    case 'left':
      return 'text-gray-600'
    default:
      return 'text-green-600'
  }
}

function getActionDescription(action: ModeratorAction): string {
  const targetName = action.target_display_name || action.target_username || 'participant'
  switch (action.action_type) {
    case 'mute':
      return `muted ${targetName}`
    case 'unmute':
      return `unmuted ${targetName}`
    case 'kick':
      return `kicked ${targetName}`
    case 'role_change':
      return `changed role of ${targetName}`
    case 'start_recording':
      return 'started recording'
    case 'stop_recording':
      return 'stopped recording'
    default:
      return action.action_type
  }
}

function getInitial(name?: string): string {
  return (name || 'U').charAt(0).toUpperCase()
}
</script>

<template>
  <div class="h-full flex flex-col">
    <!-- Header -->
    <div class="px-4 py-3 border-b bg-white">
      <div class="flex items-center gap-2 mb-1">
        <button
          @click="$emit('back')"
          class="p-1 -ml-1 text-gray-400 hover:text-gray-600 rounded transition-colors"
          title="Back to list"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
          </svg>
        </button>
        <h4 class="font-semibold text-gray-900 truncate">{{ conference.name }}</h4>
      </div>
      <div class="flex items-center gap-2 text-xs text-gray-500">
        <span>{{ conference.started_at ? formatDateTime(conference.started_at) : formatDateTime(conference.created_at) }}</span>
        <span v-if="conference.ended_at" class="text-gray-400">-</span>
        <span v-if="conference.ended_at">{{ formatTime(conference.ended_at) }}</span>
        <span
          v-else
          class="px-1.5 py-0.5 rounded-full bg-green-100 text-green-700 font-medium"
        >
          Live
        </span>
      </div>
    </div>

    <!-- Section tabs -->
    <div class="flex border-b bg-gray-50">
      <button
        @click="activeSection = 'participants'"
        :class="[
          'flex-1 py-2 text-xs font-medium border-b-2 transition-colors',
          activeSection === 'participants'
            ? 'text-indigo-600 border-indigo-600 bg-white'
            : 'text-gray-500 border-transparent hover:text-gray-700'
        ]"
      >
        Participants
      </button>
      <button
        @click="activeSection = 'messages'"
        :class="[
          'flex-1 py-2 text-xs font-medium border-b-2 transition-colors',
          activeSection === 'messages'
            ? 'text-indigo-600 border-indigo-600 bg-white'
            : 'text-gray-500 border-transparent hover:text-gray-700'
        ]"
      >
        Messages
      </button>
      <button
        v-if="isModerator"
        @click="activeSection = 'actions'"
        :class="[
          'flex-1 py-2 text-xs font-medium border-b-2 transition-colors',
          activeSection === 'actions'
            ? 'text-indigo-600 border-indigo-600 bg-white'
            : 'text-gray-500 border-transparent hover:text-gray-700'
        ]"
      >
        Actions
      </button>
    </div>

    <!-- Loading state -->
    <div v-if="loading" class="flex-1 flex justify-center items-center">
      <svg class="w-6 h-6 animate-spin text-indigo-600" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
      </svg>
    </div>

    <!-- Content -->
    <div v-else class="flex-1 overflow-y-auto">
      <!-- Participants section -->
      <div v-if="activeSection === 'participants'" class="p-4 space-y-4">
        <div v-if="!conference.all_participants?.length" class="text-center py-8 text-gray-500 text-sm">
          No participant data
        </div>
        <div
          v-for="participant in conference.all_participants"
          :key="participant.user_id"
          class="space-y-2"
        >
          <div class="flex items-center gap-2">
            <div class="w-8 h-8 rounded-full bg-indigo-100 flex items-center justify-center flex-shrink-0">
              <span class="text-xs font-medium text-indigo-600">
                {{ getInitial(participant.display_name || participant.username) }}
              </span>
            </div>
            <span class="font-medium text-sm text-gray-900">
              {{ participant.display_name || participant.username || 'Unknown' }}
            </span>
          </div>
          <div class="ml-10 space-y-1">
            <div
              v-for="(session, idx) in participant.sessions"
              :key="idx"
              class="text-xs flex items-center gap-2"
            >
              <span class="text-green-600">{{ formatTime(session.joined_at) }}</span>
              <template v-if="session.left_at">
                <span class="text-gray-400">-</span>
                <span :class="getStatusColor(session.status)">
                  {{ formatTime(session.left_at) }}
                  <span v-if="session.status === 'kicked'" class="ml-1">(kicked)</span>
                </span>
              </template>
              <span class="text-gray-400 ml-auto">
                {{ formatDuration(session.joined_at, session.left_at) }}
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- Messages section -->
      <div v-if="activeSection === 'messages'" class="p-4">
        <div v-if="messages.length === 0" class="text-center py-8 text-gray-500 text-sm">
          No messages during this event
        </div>
        <div v-else class="space-y-3">
          <div
            v-for="msg in messages"
            :key="msg.id"
            class="text-sm"
          >
            <div class="flex items-baseline gap-2">
              <span class="font-medium text-gray-900">
                {{ msg.sender_display_name || msg.sender_username || 'Unknown' }}
              </span>
              <span class="text-xs text-gray-400">{{ formatTime(msg.created_at) }}</span>
            </div>
            <p class="text-gray-700 mt-0.5">{{ msg.content }}</p>
          </div>
        </div>
      </div>

      <!-- Moderator actions section -->
      <div v-if="activeSection === 'actions' && isModerator" class="p-4">
        <div v-if="moderatorActions.length === 0" class="text-center py-8 text-gray-500 text-sm">
          No moderator actions
        </div>
        <div v-else class="space-y-2">
          <div
            v-for="action in moderatorActions"
            :key="action.id"
            class="text-sm border-l-2 border-orange-400 pl-3 py-1"
          >
            <span class="text-xs text-gray-400">{{ formatTime(action.created_at) }}</span>
            <p class="text-gray-700">
              <span class="font-medium text-gray-900">
                {{ action.actor_display_name || action.actor_username || 'Unknown' }}
              </span>
              {{ ' ' + getActionDescription(action) }}
            </p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
