<script setup lang="ts">
import { onMounted, watch } from 'vue'
import type { Participant } from '@/types'
import { usePresenceStore } from '@/stores/presence'
import StatusIndicator from './StatusIndicator.vue'

const props = defineProps<{
  participants: Participant[]
}>()

const presenceStore = usePresenceStore()

// Fetch presence for all participants when they change
watch(
  () => props.participants,
  (participants) => {
    if (participants.length > 0) {
      const userIds = participants.map(p => p.user_id)
      presenceStore.fetchUsersPresence(userIds)
    }
  },
  { immediate: true }
)

onMounted(() => {
  if (props.participants.length > 0) {
    const userIds = props.participants.map(p => p.user_id)
    presenceStore.fetchUsersPresence(userIds)
  }
})

defineEmits<{
  close: []
}>()

function getDisplayName(participant: Participant): string {
  // Priority: participant.display_name > user.display_name > participant.username > user.username
  if (participant.display_name) {
    return participant.display_name
  }
  if (participant.user?.display_name) {
    return participant.user.display_name
  }
  if (participant.username) {
    return participant.username
  }
  if (participant.user?.username) {
    return participant.user.username
  }
  return 'Unknown'
}

function getUsername(participant: Participant): string | null {
  return participant.username || participant.user?.username || null
}

function getAvatarUrl(participant: Participant): string | null {
  // Priority: participant.avatar_url > user.avatar_url
  if (participant.avatar_url) {
    return participant.avatar_url
  }
  if (participant.user?.avatar_url) {
    return participant.user.avatar_url
  }
  return null
}

function getRandomCatUrl(participantId: string): string {
  // Use participant ID as seed for consistent cat per user
  const seed = participantId.replace(/-/g, '').substring(0, 8)
  return `https://cataas.com/cat?width=64&height=64&${seed}`
}

// Normalize role - handle both string names and numeric enum values from protobuf
function normalizeRole(role: string | number): string {
  // If it's a number (protobuf enum value), convert to string name
  if (typeof role === 'number' || !isNaN(Number(role))) {
    const numRole = Number(role)
    switch (numRole) {
      case 0: return 'unspecified'
      case 1: return 'admin'
      case 2: return 'member'
      case 3: return 'readonly'
      default: return 'member'
    }
  }
  // Already a string
  return role.toLowerCase()
}

function getRoleBadgeClass(role: string | number): string {
  const normalizedRole = normalizeRole(role)
  switch (normalizedRole) {
    case 'owner':
      return 'bg-purple-100 text-purple-800'
    case 'admin':
      return 'bg-blue-100 text-blue-800'
    case 'member':
      return 'bg-gray-100 text-gray-800'
    case 'guest':
    case 'readonly':
      return 'bg-yellow-100 text-yellow-800'
    default:
      return 'bg-gray-100 text-gray-800'
  }
}

function getRoleLabel(role: string | number): string {
  const normalizedRole = normalizeRole(role)
  switch (normalizedRole) {
    case 'owner':
      return 'Owner'
    case 'admin':
      return 'Admin'
    case 'member':
      return 'Member'
    case 'guest':
      return 'Guest'
    case 'readonly':
      return 'Read-only'
    default:
      return 'Member'
  }
}
</script>

<template>
  <div class="w-64 border-l bg-white flex flex-col h-full">
    <div class="px-4 py-3 border-b flex items-center justify-between">
      <h4 class="font-semibold text-gray-900">Participants</h4>
      <button
        @click="$emit('close')"
        class="p-1 text-gray-400 hover:text-gray-600 rounded"
      >
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
    <div class="flex-1 overflow-y-auto">
      <ul class="divide-y divide-gray-100">
        <li
          v-for="participant in participants"
          :key="participant.user_id"
          class="px-4 py-3 hover:bg-gray-50"
        >
          <div class="flex items-center gap-3">
            <div class="relative">
              <img
                v-if="getAvatarUrl(participant)"
                :src="getAvatarUrl(participant)!"
                :alt="getDisplayName(participant)"
                class="w-8 h-8 rounded-full object-cover"
              />
              <img
                v-else
                :src="getRandomCatUrl(participant.user_id)"
                :alt="getDisplayName(participant)"
                class="w-8 h-8 rounded-full object-cover"
              />
              <StatusIndicator
                :user-id="participant.user_id"
                size="sm"
                class="absolute -bottom-0.5 -right-0.5 border-2 border-white rounded-full"
              />
            </div>
            <div class="flex-1 min-w-0">
              <p class="text-sm font-medium text-gray-900 truncate">
                {{ getDisplayName(participant) }}
              </p>
              <p v-if="getUsername(participant)" class="text-xs text-gray-500 truncate">
                @{{ getUsername(participant) }}
              </p>
            </div>
            <span
              :class="getRoleBadgeClass(participant.role)"
              class="text-xs px-2 py-0.5 rounded-full font-medium"
            >
              {{ getRoleLabel(participant.role) }}
            </span>
          </div>
        </li>
      </ul>
      <div v-if="participants.length === 0" class="px-4 py-8 text-center text-gray-500 text-sm">
        No participants
      </div>
    </div>
  </div>
</template>
