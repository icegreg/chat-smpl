<script setup lang="ts">
import { computed } from 'vue'
import type { Participant } from '@/types'
import { usePresenceStore } from '@/stores/presence'
import StatusIndicator from './StatusIndicator.vue'

const props = defineProps<{
  participant: Participant
}>()

const emit = defineEmits<{
  close: []
}>()

const presenceStore = usePresenceStore()

const displayName = computed(() => {
  if (props.participant.display_name) {
    return props.participant.display_name
  }
  if (props.participant.user?.display_name) {
    return props.participant.user.display_name
  }
  if (props.participant.username) {
    return props.participant.username
  }
  if (props.participant.user?.username) {
    return props.participant.user.username
  }
  return 'Unknown'
})

const username = computed(() => {
  return props.participant.username || props.participant.user?.username || null
})

const avatarUrl = computed(() => {
  if (props.participant.avatar_url) {
    return props.participant.avatar_url
  }
  if (props.participant.user?.avatar_url) {
    return props.participant.user.avatar_url
  }
  return null
})

const randomCatUrl = computed(() => {
  const seed = props.participant.user_id.replace(/-/g, '').substring(0, 8)
  return `https://cataas.com/cat?width=128&height=128&${seed}`
})

const presence = computed(() => {
  return presenceStore.getUserPresence(props.participant.user_id)
})

const isOnline = computed(() => {
  return presence.value?.is_online ?? false
})

const orgInfo = computed(() => {
  return props.participant.org_info
})

const hasOrgData = computed(() => {
  return orgInfo.value?.has_org_data ?? false
})

// Get local time for user based on their timezone
const localTime = computed(() => {
  const timezone = orgInfo.value?.timezone || 'Europe/Moscow'
  try {
    return new Date().toLocaleTimeString('ru-RU', {
      timeZone: timezone,
      hour: '2-digit',
      minute: '2-digit'
    })
  } catch {
    // Fallback if timezone is invalid
    return new Date().toLocaleTimeString('ru-RU', {
      timeZone: 'Europe/Moscow',
      hour: '2-digit',
      minute: '2-digit'
    })
  }
})

const timezoneLabel = computed(() => {
  const tz = orgInfo.value?.timezone || 'Europe/Moscow'
  // Simple mapping for common Russian timezones
  const tzMap: Record<string, string> = {
    'Europe/Moscow': 'MSK (UTC+3)',
    'Europe/Kaliningrad': 'UTC+2',
    'Europe/Samara': 'UTC+4',
    'Asia/Yekaterinburg': 'UTC+5',
    'Asia/Omsk': 'UTC+6',
    'Asia/Krasnoyarsk': 'UTC+7',
    'Asia/Irkutsk': 'UTC+8',
    'Asia/Yakutsk': 'UTC+9',
    'Asia/Vladivostok': 'UTC+10',
    'Asia/Magadan': 'UTC+11',
    'Asia/Kamchatka': 'UTC+12'
  }
  return tzMap[tz] || tz
})

// Close on Escape key
function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    emit('close')
  }
}
</script>

<template>
  <Teleport to="body">
    <div
      class="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
      @click.self="$emit('close')"
      @keydown="handleKeydown"
      tabindex="0"
    >
      <div class="bg-white rounded-lg shadow-xl w-80 max-w-[90vw] overflow-hidden">
        <!-- Header with close button -->
        <div class="relative bg-gradient-to-br from-blue-500 to-indigo-600 pt-8 pb-16">
          <button
            @click="$emit('close')"
            class="absolute top-2 right-2 p-1 text-white/80 hover:text-white rounded"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <!-- Avatar (positioned to overlap header) -->
        <div class="relative flex justify-center -mt-12">
          <div class="relative">
            <img
              v-if="avatarUrl"
              :src="avatarUrl"
              :alt="displayName"
              class="w-24 h-24 rounded-full object-cover border-4 border-white shadow-lg"
            />
            <img
              v-else
              :src="randomCatUrl"
              :alt="displayName"
              class="w-24 h-24 rounded-full object-cover border-4 border-white shadow-lg"
            />
            <StatusIndicator
              :user-id="participant.user_id"
              size="md"
              class="absolute bottom-1 right-1 border-2 border-white rounded-full"
            />
          </div>
        </div>

        <!-- User info -->
        <div class="px-6 pt-4 pb-6 text-center">
          <!-- Name and username -->
          <h3 class="text-xl font-semibold text-gray-900">{{ displayName }}</h3>
          <p v-if="username" class="text-sm text-gray-500 mt-0.5">@{{ username }}</p>

          <!-- Online status -->
          <p class="text-sm mt-2" :class="isOnline ? 'text-green-600' : 'text-gray-400'">
            {{ isOnline ? 'Online' : 'Offline' }}
          </p>

          <!-- Organization info -->
          <div v-if="hasOrgData" class="mt-4 text-left border-t pt-4">
            <!-- Position -->
            <div v-if="orgInfo?.position_name" class="flex items-start gap-2 mb-2">
              <svg class="w-5 h-5 text-gray-400 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 13.255A23.931 23.931 0 0112 15c-3.183 0-6.22-.62-9-1.745M16 6V4a2 2 0 00-2-2h-4a2 2 0 00-2 2v2m4 6h.01M5 20h14a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
              </svg>
              <span class="text-sm text-gray-700">{{ orgInfo.position_name }}</span>
            </div>

            <!-- Department -->
            <div v-if="orgInfo?.department_name" class="flex items-start gap-2 mb-2">
              <svg class="w-5 h-5 text-gray-400 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
              </svg>
              <span class="text-sm text-gray-700">{{ orgInfo.department_name }}</span>
            </div>

            <!-- Company -->
            <div v-if="orgInfo?.company_name" class="flex items-start gap-2 mb-2">
              <svg class="w-5 h-5 text-gray-400 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
              </svg>
              <span class="text-sm text-gray-700">{{ orgInfo.company_name }}</span>
            </div>

            <!-- Local time -->
            <div class="flex items-start gap-2">
              <svg class="w-5 h-5 text-gray-400 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <span class="text-sm text-gray-700">
                {{ localTime }} <span class="text-gray-400">({{ timezoneLabel }})</span>
              </span>
            </div>
          </div>

          <!-- No org data message for guests -->
          <div v-else class="mt-4 text-sm text-gray-400 italic">
            No organization data
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>
