<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import type { Message } from '@/types'
import { useChatStore } from '@/stores/chat'

const props = defineProps<{
  message: Message
  currentChatId: string
}>()

const emit = defineEmits<{
  close: []
  forward: [targetChatId: string, comment: string]
}>()

const chatStore = useChatStore()
const searchQuery = ref('')
const selectedChatId = ref<string | null>(null)
const comment = ref('')
const sending = ref(false)

// Filter out current chat and search
const availableChats = computed(() => {
  const query = searchQuery.value.toLowerCase()
  return chatStore.chats
    .filter(chat => chat.id !== props.currentChatId)
    .filter(chat => !query || chat.name.toLowerCase().includes(query))
})

function selectChat(chatId: string) {
  selectedChatId.value = chatId
}

async function handleForward() {
  if (!selectedChatId.value) return

  sending.value = true
  try {
    emit('forward', selectedChatId.value, comment.value)
  } finally {
    sending.value = false
  }
}

function getMessagePreview(): string {
  if (props.message.content) {
    return props.message.content.length > 100
      ? props.message.content.substring(0, 100) + '...'
      : props.message.content
  }
  if (props.message.file_attachments?.length) {
    return `📎 ${props.message.file_attachments.length} file(s)`
  }
  return 'Message'
}

function getSenderName(): string {
  if (props.message.sender_display_name) return props.message.sender_display_name
  if (props.message.sender?.display_name) return props.message.sender.display_name
  if (props.message.sender_username) return props.message.sender_username
  if (props.message.sender?.username) return props.message.sender.username
  return 'Unknown'
}

onMounted(() => {
  // Load chats if not loaded
  if (chatStore.chats.length === 0) {
    chatStore.fetchChats()
  }
})
</script>

<template>
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="$emit('close')">
    <div class="bg-white rounded-xl shadow-2xl w-full max-w-md mx-4 overflow-hidden" data-testid="forward-modal">
      <!-- Header -->
      <div class="px-4 py-3 border-b bg-gray-50 flex items-center justify-between">
        <h3 class="font-semibold text-gray-900">Forward Message</h3>
        <button
          @click="$emit('close')"
          class="p-1 text-gray-400 hover:text-gray-600 rounded"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <!-- Message Preview -->
      <div class="px-4 py-3 bg-indigo-50 border-b">
        <div class="flex items-start gap-2">
          <svg class="w-5 h-5 text-indigo-500 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 5l7 7-7 7M5 5l7 7-7 7" />
          </svg>
          <div class="flex-1 min-w-0">
            <p class="text-xs text-indigo-600 font-medium">{{ getSenderName() }}</p>
            <p class="text-sm text-gray-700 line-clamp-2">{{ getMessagePreview() }}</p>
          </div>
        </div>
      </div>

      <!-- Search -->
      <div class="px-4 py-2 border-b">
        <input
          v-model="searchQuery"
          type="text"
          placeholder="Search chats..."
          class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent text-sm"
          data-testid="forward-search-input"
        />
      </div>

      <!-- Chat List -->
      <div class="max-h-64 overflow-y-auto">
        <div v-if="availableChats.length === 0" class="px-4 py-8 text-center text-gray-500 text-sm">
          No chats available
        </div>
        <button
          v-for="chat in availableChats"
          :key="chat.id"
          @click="selectChat(chat.id)"
          class="w-full px-4 py-3 flex items-center gap-3 hover:bg-gray-50 transition-colors text-left"
          :class="{ 'bg-indigo-50': selectedChatId === chat.id }"
          data-testid="forward-chat-item"
        >
          <!-- Chat avatar -->
          <div
            class="w-10 h-10 rounded-full flex items-center justify-center text-white font-medium"
            :class="chat.type === 'channel' ? 'bg-purple-500' : 'bg-indigo-500'"
          >
            {{ chat.name[0].toUpperCase() }}
          </div>
          <div class="flex-1 min-w-0">
            <p class="font-medium text-gray-900 truncate">{{ chat.name }}</p>
            <p class="text-xs text-gray-500">
              {{ chat.type === 'channel' ? 'Channel' : 'Group' }} · {{ chat.participant_count }} members
            </p>
          </div>
          <!-- Selected indicator -->
          <div v-if="selectedChatId === chat.id" class="w-5 h-5 rounded-full bg-indigo-500 flex items-center justify-center">
            <svg class="w-3 h-3 text-white" fill="currentColor" viewBox="0 0 20 20">
              <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
            </svg>
          </div>
        </button>
      </div>

      <!-- Comment input -->
      <div class="px-4 py-3 border-t">
        <textarea
          v-model="comment"
          placeholder="Add a comment (optional)..."
          rows="2"
          class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent text-sm resize-none"
          data-testid="forward-comment-input"
        />
      </div>

      <!-- Actions -->
      <div class="px-4 py-3 border-t bg-gray-50 flex justify-end gap-2">
        <button
          @click="$emit('close')"
          class="px-4 py-2 text-gray-700 hover:bg-gray-200 rounded-lg transition-colors"
        >
          Cancel
        </button>
        <button
          @click="handleForward"
          :disabled="!selectedChatId || sending"
          class="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center gap-2"
          data-testid="forward-submit-button"
        >
          <svg v-if="sending" class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          <svg v-else class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 5l7 7-7 7M5 5l7 7-7 7" />
          </svg>
          Forward
        </button>
      </div>
    </div>
  </div>
</template>
