<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import type { Chat, ScheduledConference } from '@/types'
import CreateChatModal from './CreateChatModal.vue'
import StatusSelector from './StatusSelector.vue'
import { usePresenceStore } from '@/stores/presence'
import { useVoiceStore } from '@/stores/voice'
import { useAuthStore } from '@/stores/auth'

defineProps<{
  chats: Chat[]
  currentChatId?: string
  loading: boolean
}>()

const router = useRouter()
const presenceStore = usePresenceStore()
const voiceStore = useVoiceStore()
const authStore = useAuthStore()

// Fetch scheduled conferences
onMounted(() => {
  presenceStore.setupVisibilityHandler()
  voiceStore.fetchScheduledConferences(true) // upcoming only
})

// Active events where user is participant with accepted RSVP or pending
const activeEvents = computed(() => {
  const userId = authStore.user?.id
  if (!userId) return []

  return voiceStore.scheduledConferences
    .filter(conf => {
      // Check if user is participant
      const participant = conf.participants?.find(p => p.user_id === userId)
      if (!participant) return false
      // Show if accepted or pending
      return participant.rsvp_status === 'accepted' || participant.rsvp_status === 'pending'
    })
    .slice(0, 5) // Limit to 5 events
})

const emit = defineEmits<{
  select: [chatId: string]
}>()

void emit
const showCreateModal = ref(false)

function formatTime(dateString: string): string {
  const date = new Date(dateString)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))

  if (days === 0) {
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  } else if (days === 1) {
    return 'Yesterday'
  } else if (days < 7) {
    return date.toLocaleDateString([], { weekday: 'short' })
  }
  return date.toLocaleDateString([], { month: 'short', day: 'numeric' })
}

function getChatIcon(type: string): string {
  switch (type) {
    case 'direct':
      return 'user'
    case 'group':
      return 'users'
    case 'channel':
      return 'hash'
    default:
      return 'message'
  }
}

async function handleChatCreated(chat: Chat) {
  showCreateModal.value = false
  emit('select', chat.id)
}

// Event helpers
function formatEventTime(dateString: string | undefined): string {
  if (!dateString) return ''
  const date = new Date(dateString)
  const now = new Date()
  const diff = date.getTime() - now.getTime()
  const minutes = Math.floor(diff / (1000 * 60))
  const hours = Math.floor(diff / (1000 * 60 * 60))
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))

  if (diff < 0) {
    return 'Now'
  } else if (minutes < 60) {
    return `in ${minutes}m`
  } else if (hours < 24) {
    return `in ${hours}h`
  } else if (days === 1) {
    return 'Tomorrow'
  } else {
    return date.toLocaleDateString([], { month: 'short', day: 'numeric' })
  }
}

function getEventTypeIcon(eventType: string): string {
  switch (eventType) {
    case 'adhoc':
    case 'adhoc_chat':
      return 'phone'
    case 'scheduled':
    case 'recurring':
      return 'calendar'
    default:
      return 'video'
  }
}

function getMyRsvp(event: ScheduledConference): string {
  const userId = authStore.user?.id
  if (!userId) return 'pending'
  const participant = event.participants?.find(p => p.user_id === userId)
  return participant?.rsvp_status || 'pending'
}

function handleEventClick(_event: ScheduledConference) {
  router.push('/events')
}

async function handleJoinEvent(event: ScheduledConference) {
  await voiceStore.joinConference(event.id)
}
</script>

<template>
  <aside class="w-80 bg-white border-r flex flex-col">
    <!-- Header -->
    <div class="p-4 border-b">
      <div class="flex items-center justify-between mb-3">
        <h2 class="font-semibold text-gray-700">Chats</h2>
        <button
          @click="showCreateModal = true"
          class="p-2 text-gray-500 hover:text-indigo-600 hover:bg-gray-100 rounded-lg"
          title="Create new chat"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
          </svg>
        </button>
      </div>
      <StatusSelector />
    </div>

    <!-- Chat list -->
    <div class="flex-1 overflow-y-auto">
      <!-- Active Events Section -->
      <div v-if="activeEvents.length > 0" class="border-b">
        <div class="px-4 py-2 bg-gradient-to-r from-indigo-50 to-purple-50">
          <div class="flex items-center justify-between">
            <h3 class="text-xs font-semibold text-indigo-700 uppercase tracking-wide">
              Upcoming Events
            </h3>
            <router-link
              to="/events"
              class="text-xs text-indigo-600 hover:text-indigo-800"
            >
              View all
            </router-link>
          </div>
        </div>
        <div class="divide-y divide-gray-100">
          <div
            v-for="event in activeEvents"
            :key="event.id"
            @click="handleEventClick(event)"
            class="px-4 py-3 hover:bg-indigo-50 cursor-pointer transition-colors"
          >
            <div class="flex items-start gap-3">
              <!-- Event icon -->
              <div class="w-10 h-10 rounded-full bg-indigo-100 flex items-center justify-center flex-shrink-0">
                <svg v-if="getEventTypeIcon(event.event_type) === 'calendar'" class="w-5 h-5 text-indigo-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
                <svg v-else-if="getEventTypeIcon(event.event_type) === 'phone'" class="w-5 h-5 text-indigo-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" />
                </svg>
                <svg v-else class="w-5 h-5 text-indigo-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" />
                </svg>
              </div>

              <!-- Event info -->
              <div class="flex-1 min-w-0">
                <div class="flex items-center justify-between">
                  <span class="font-medium text-gray-900 truncate text-sm">
                    {{ event.name }}
                  </span>
                  <span class="text-xs font-medium text-indigo-600">
                    {{ formatEventTime(event.scheduled_at) }}
                  </span>
                </div>
                <div class="flex items-center justify-between mt-1">
                  <div class="flex items-center gap-1 text-xs text-gray-500">
                    <span>{{ event.accepted_count || 0 }} joined</span>
                    <span v-if="getMyRsvp(event) === 'pending'" class="text-yellow-600">• Pending</span>
                  </div>
                  <button
                    v-if="event.status === 'active'"
                    @click.stop="handleJoinEvent(event)"
                    class="px-2 py-1 text-xs font-medium text-white bg-green-500 hover:bg-green-600 rounded"
                  >
                    Join
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Chats Section Header (only if events are shown) -->
      <div v-if="activeEvents.length > 0" class="px-4 py-2 bg-gray-50 border-b">
        <h3 class="text-xs font-semibold text-gray-500 uppercase tracking-wide">Chats</h3>
      </div>

      <div v-if="loading && chats.length === 0" class="p-4 text-center text-gray-500">
        Loading...
      </div>

      <div v-else-if="chats.length === 0" class="p-4 text-center text-gray-500">
        No chats yet. Create one to start messaging!
      </div>

      <div v-else class="divide-y">
        <button
          v-for="chat in chats"
          :key="chat.id"
          @click="$emit('select', chat.id)"
          class="w-full p-4 flex items-start gap-3 hover:bg-gray-50 transition-colors text-left"
          :class="{ 'bg-indigo-50': currentChatId === chat.id }"
        >
          <!-- Avatar/Icon -->
          <div
            class="w-10 h-10 rounded-full flex items-center justify-center flex-shrink-0"
            :class="{
              'bg-indigo-100 text-indigo-600': chat.type === 'group',
              'bg-green-100 text-green-600': chat.type === 'direct',
              'bg-purple-100 text-purple-600': chat.type === 'channel',
            }"
          >
            <svg v-if="getChatIcon(chat.type) === 'user'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
            </svg>
            <svg v-else-if="getChatIcon(chat.type) === 'users'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
            </svg>
            <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 20l4-16m2 16l4-16M6 9h14M4 15h14" />
            </svg>
          </div>

          <!-- Content -->
          <div class="flex-1 min-w-0">
            <div class="flex items-center justify-between">
              <span class="font-medium text-gray-900 truncate">
                {{ chat.name }}
                <span v-if="chat.is_favorite" class="text-yellow-500 ml-1">★</span>
              </span>
              <span v-if="chat.last_message" class="text-xs text-gray-500">
                {{ formatTime(chat.last_message.created_at) }}
              </span>
            </div>
            <div class="flex items-center justify-between mt-1">
              <p v-if="chat.last_message" class="text-sm text-gray-500 truncate">
                {{ chat.last_message.content }}
              </p>
              <p v-else class="text-sm text-gray-400 italic">No messages yet</p>
              <span
                v-if="chat.unread_count && chat.unread_count > 0"
                class="ml-2 px-2 py-0.5 text-xs font-medium bg-indigo-600 text-white rounded-full"
              >
                {{ chat.unread_count }}
              </span>
            </div>
          </div>
        </button>
      </div>
    </div>

    <!-- Create chat modal -->
    <CreateChatModal
      v-if="showCreateModal"
      @close="showCreateModal = false"
      @created="handleChatCreated"
    />
  </aside>
</template>
