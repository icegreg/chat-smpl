<script setup lang="ts">
import { ref } from 'vue'
import type { Chat } from '@/types'
import CreateChatModal from './CreateChatModal.vue'

defineProps<{
  chats: Chat[]
  currentChatId?: string
  loading: boolean
}>()

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
</script>

<template>
  <aside class="w-80 bg-white border-r flex flex-col">
    <!-- Header -->
    <div class="p-4 border-b flex items-center justify-between">
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

    <!-- Chat list -->
    <div class="flex-1 overflow-y-auto">
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
