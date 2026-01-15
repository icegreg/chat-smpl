<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import type { Participant, User } from '@/types'
import { usePresenceStore } from '@/stores/presence'
import { useChatStore } from '@/stores/chat'
import { api } from '@/api/client'
import StatusIndicator from './StatusIndicator.vue'
import ProfileCard from './ProfileCard.vue'

const props = defineProps<{
  participants: Participant[]
  currentUser: User
  chatId: string
}>()

const chatStore = useChatStore()

// Add participant state
const showAddForm = ref(false)
const newParticipantRole = ref<'member' | 'admin' | 'readonly'>('member')
const addLoading = ref(false)
const addError = ref('')

// User search state
const searchQuery = ref('')
const searchResults = ref<User[]>([])
const searchLoading = ref(false)
const selectedUser = ref<User | null>(null)
const searchDropdownRef = ref<HTMLElement | null>(null)
let searchTimeout: ReturnType<typeof setTimeout> | null = null

// Pagination state for search results
const searchPage = ref(1)
const searchHasMore = ref(false)
const searchLoadingMore = ref(false)
const SEARCH_PAGE_SIZE = 15

// Debounced search function
function handleSearchInput() {
  if (searchTimeout) {
    clearTimeout(searchTimeout)
  }

  const query = searchQuery.value.trim()
  if (query.length < 2) {
    searchResults.value = []
    searchPage.value = 1
    searchHasMore.value = false
    return
  }

  searchLoading.value = true
  searchPage.value = 1
  searchTimeout = setTimeout(async () => {
    try {
      const response = await api.searchUsers(query, 1, SEARCH_PAGE_SIZE)
      // Filter out users who are already participants
      const participantIds = new Set(props.participants.map(p => p.user_id))
      searchResults.value = response.users.filter(u => !participantIds.has(u.id))
      searchHasMore.value = response.pagination.page < response.pagination.total_pages
    } catch (e) {
      console.error('Search failed:', e)
      searchResults.value = []
      searchHasMore.value = false
    } finally {
      searchLoading.value = false
    }
  }, 300)
}

// Load more results when scrolling
async function loadMoreResults() {
  if (searchLoadingMore.value || !searchHasMore.value || searchQuery.value.trim().length < 2) {
    return
  }

  searchLoadingMore.value = true
  const nextPage = searchPage.value + 1

  try {
    const response = await api.searchUsers(searchQuery.value.trim(), nextPage, SEARCH_PAGE_SIZE)
    const participantIds = new Set(props.participants.map(p => p.user_id))
    const newUsers = response.users.filter(u => !participantIds.has(u.id))
    searchResults.value = [...searchResults.value, ...newUsers]
    searchPage.value = nextPage
    searchHasMore.value = response.pagination.page < response.pagination.total_pages
  } catch (e) {
    console.error('Load more failed:', e)
  } finally {
    searchLoadingMore.value = false
  }
}

// Handle scroll in dropdown
function handleDropdownScroll(event: Event) {
  const target = event.target as HTMLElement
  const scrollBottom = target.scrollHeight - target.scrollTop - target.clientHeight
  // Load more when within 50px of bottom
  if (scrollBottom < 50) {
    loadMoreResults()
  }
}

function selectUser(user: User) {
  selectedUser.value = user
  searchQuery.value = ''
  searchResults.value = []
  searchPage.value = 1
  searchHasMore.value = false
}

function clearSelectedUser() {
  selectedUser.value = null
}

// Remove participant state
const removeLoading = ref<string | null>(null)

// Normalize role - handle both string names and numeric enum values from protobuf
// Moved before canManageParticipants to use it there
function normalizeRoleValue(role: string | number): string {
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
  return String(role).toLowerCase()
}

// Check if current user can manage participants
// User can manage if:
// 1. They have system-wide owner/moderator role, OR
// 2. They are the chat admin (their participant role is 'admin')
const canManageParticipants = computed(() => {
  // Check system-wide role
  const systemRole = props.currentUser.role
  if (systemRole === 'owner' || systemRole === 'moderator') {
    return true
  }

  // Check chat participant role (normalize to handle protobuf enum values)
  const currentParticipant = props.participants.find(p => p.user_id === props.currentUser.id)
  if (currentParticipant) {
    const normalizedRole = normalizeRoleValue(currentParticipant.role)
    if (normalizedRole === 'admin') {
      return true
    }
  }

  return false
})

async function handleAddParticipant() {
  if (!selectedUser.value) {
    addError.value = 'Выберите пользователя'
    return
  }

  addLoading.value = true
  addError.value = ''

  try {
    await chatStore.addParticipant(props.chatId, selectedUser.value.id, newParticipantRole.value)
    selectedUser.value = null
    searchQuery.value = ''
    showAddForm.value = false
  } catch (e) {
    addError.value = chatStore.error || 'Не удалось добавить участника'
  } finally {
    addLoading.value = false
  }
}

async function handleRemoveParticipant(userId: string, event: Event) {
  event.stopPropagation() // Prevent opening profile card

  if (!confirm('Удалить участника из чата?')) return

  removeLoading.value = userId

  try {
    await chatStore.removeParticipant(props.chatId, userId)
  } catch (e) {
    console.error('Failed to remove participant:', e)
  } finally {
    removeLoading.value = null
  }
}

// Selected participant for profile card modal
const selectedParticipant = ref<Participant | null>(null)

function openProfile(participant: Participant) {
  selectedParticipant.value = participant
}

function closeProfile() {
  selectedParticipant.value = null
}

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

// Get sort key: display_name if exists, otherwise username
function getSortKey(participant: Participant): string {
  const displayName = participant.display_name || participant.user?.display_name
  if (displayName) {
    return displayName
  }
  return participant.username || participant.user?.username || ''
}

// Sorted participants by display_name, fallback to username (ASCII order)
const sortedParticipants = computed(() => {
  return [...props.participants].sort((a, b) => {
    const keyA = getSortKey(a)
    const keyB = getSortKey(b)
    return keyA.localeCompare(keyB, undefined, { sensitivity: 'base' })
  })
})

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
      <h4 class="font-semibold text-gray-900">Участники</h4>
      <button
        @click="$emit('close')"
        class="p-1 text-gray-400 hover:text-gray-600 rounded"
      >
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>

    <!-- Add participant button -->
    <div v-if="canManageParticipants" class="px-4 py-2 border-b">
      <button
        v-if="!showAddForm"
        @click="showAddForm = true"
        class="w-full flex items-center justify-center gap-2 px-3 py-2 text-sm text-indigo-600 hover:bg-indigo-50 rounded-lg transition-colors"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        Добавить участника
      </button>

      <!-- Add participant form -->
      <div v-else class="space-y-2">
        <div v-if="addError" class="text-xs text-red-600">{{ addError }}</div>

        <!-- Selected user display -->
        <div v-if="selectedUser" class="flex items-center gap-2 p-2 bg-indigo-50 rounded border border-indigo-200">
          <img
            v-if="selectedUser.avatar_url"
            :src="selectedUser.avatar_url"
            :alt="selectedUser.display_name || selectedUser.username"
            class="w-6 h-6 rounded-full object-cover"
          />
          <div
            v-else
            class="w-6 h-6 rounded-full bg-indigo-200 flex items-center justify-center text-xs font-medium text-indigo-700"
          >
            {{ (selectedUser.display_name || selectedUser.username).charAt(0).toUpperCase() }}
          </div>
          <div class="flex-1 min-w-0">
            <p class="text-sm font-medium text-gray-900 truncate">
              {{ selectedUser.display_name || selectedUser.username }}
            </p>
            <p class="text-xs text-gray-500 truncate">@{{ selectedUser.username }}</p>
          </div>
          <button
            @click="clearSelectedUser"
            class="p-1 text-gray-400 hover:text-gray-600"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <!-- Search input -->
        <div v-else class="relative">
          <input
            v-model="searchQuery"
            type="text"
            placeholder="Поиск по имени или логину..."
            class="w-full px-2 py-1.5 text-sm border border-gray-300 rounded focus:outline-none focus:ring-1 focus:ring-indigo-500"
            @input="handleSearchInput"
          />
          <div v-if="searchLoading" class="absolute right-2 top-1/2 -translate-y-1/2">
            <svg class="w-4 h-4 animate-spin text-gray-400" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
            </svg>
          </div>

          <!-- Search results dropdown -->
          <div
            v-if="searchResults.length > 0"
            ref="searchDropdownRef"
            class="absolute z-10 w-full mt-1 bg-white border border-gray-200 rounded-lg shadow-lg max-h-64 overflow-y-auto"
            @scroll="handleDropdownScroll"
          >
            <button
              v-for="user in searchResults"
              :key="user.id"
              @click="selectUser(user)"
              class="w-full flex items-center gap-2 px-3 py-2 hover:bg-gray-50 text-left"
            >
              <img
                v-if="user.avatar_url"
                :src="user.avatar_url"
                :alt="user.display_name || user.username"
                class="w-6 h-6 rounded-full object-cover"
              />
              <div
                v-else
                class="w-6 h-6 rounded-full bg-gray-200 flex items-center justify-center text-xs font-medium text-gray-600"
              >
                {{ (user.display_name || user.username).charAt(0).toUpperCase() }}
              </div>
              <div class="flex-1 min-w-0">
                <p class="text-sm font-medium text-gray-900 truncate">
                  {{ user.display_name || user.username }}
                </p>
                <p class="text-xs text-gray-500 truncate">@{{ user.username }}</p>
              </div>
            </button>
            <!-- Loading more indicator -->
            <div v-if="searchLoadingMore" class="flex justify-center py-2">
              <svg class="w-5 h-5 animate-spin text-gray-400" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
              </svg>
            </div>
            <!-- Load more hint -->
            <div v-else-if="searchHasMore" class="text-center py-2 text-xs text-gray-400">
              Прокрутите для загрузки ещё
            </div>
          </div>

          <!-- No results message -->
          <div
            v-else-if="searchQuery.length >= 2 && !searchLoading"
            class="absolute z-10 w-full mt-1 px-3 py-2 bg-white border border-gray-200 rounded-lg shadow-lg text-sm text-gray-500"
          >
            Пользователи не найдены
          </div>
        </div>

        <select
          v-model="newParticipantRole"
          class="w-full px-2 py-1.5 text-sm border border-gray-300 rounded focus:outline-none focus:ring-1 focus:ring-indigo-500"
        >
          <option value="member">Участник</option>
          <option value="admin">Админ</option>
          <option value="readonly">Только чтение</option>
        </select>
        <div class="flex gap-2">
          <button
            @click="showAddForm = false; addError = ''; selectedUser = null; searchQuery = ''; searchResults = []; searchPage = 1; searchHasMore = false"
            class="flex-1 px-2 py-1.5 text-sm text-gray-600 hover:bg-gray-100 rounded"
          >
            Отмена
          </button>
          <button
            @click="handleAddParticipant"
            :disabled="addLoading || !selectedUser"
            class="flex-1 px-2 py-1.5 text-sm text-white bg-indigo-600 hover:bg-indigo-700 rounded disabled:opacity-50"
          >
            {{ addLoading ? '...' : 'Добавить' }}
          </button>
        </div>
      </div>
    </div>

    <div class="flex-1 overflow-y-auto">
      <ul class="divide-y divide-gray-100">
        <li
          v-for="participant in sortedParticipants"
          :key="participant.user_id"
          class="px-4 py-3 hover:bg-gray-50 cursor-pointer group"
          @click="openProfile(participant)"
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
            <!-- Remove button (visible for moderators, not for self) -->
            <button
              v-if="canManageParticipants && participant.user_id !== currentUser.id"
              @click="handleRemoveParticipant(participant.user_id, $event)"
              :disabled="removeLoading === participant.user_id"
              class="p-1 text-gray-400 hover:text-red-600 rounded opacity-0 group-hover:opacity-100 transition-opacity"
              title="Удалить участника"
            >
              <svg v-if="removeLoading !== participant.user_id" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
              <svg v-else class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
              </svg>
            </button>
          </div>
        </li>
      </ul>
      <div v-if="participants.length === 0" class="px-4 py-8 text-center text-gray-500 text-sm">
        Нет участников
      </div>
    </div>

    <!-- Profile Card Modal -->
    <ProfileCard
      v-if="selectedParticipant"
      :participant="selectedParticipant"
      @close="closeProfile"
    />
  </div>
</template>
