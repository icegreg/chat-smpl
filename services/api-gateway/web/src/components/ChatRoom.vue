<script setup lang="ts">
import { ref, computed, nextTick, watch } from 'vue'
import type { Chat, Message, Participant, User } from '@/types'
import { useChatStore } from '@/stores/chat'
import MessageItem from './MessageItem.vue'
import ParticipantsPanel from './ParticipantsPanel.vue'

const props = defineProps<{
  chat: Chat
  messages: Message[]
  participants: Participant[]
  currentUser: User
  isGuest: boolean
}>()

const chatStore = useChatStore()
const messageInput = ref('')
const messagesContainer = ref<HTMLElement | null>(null)
const lastTypingSentAt = ref<number>(0)
const showParticipants = ref(false)

const TYPING_SEND_INTERVAL = 5000 // Send typing indicator every 5 seconds

const sortedMessages = computed(() => {
  return [...props.messages].sort(
    (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
  )
})

const typingIndicator = computed(() => {
  const chatTyping = chatStore.typingUsers.get(props.chat.id)
  if (!chatTyping || chatTyping.size === 0) return null

  const users = Array.from(chatTyping).filter((id) => id !== props.currentUser.id)
  if (users.length === 0) return null

  if (users.length === 1) return 'Someone is typing...'
  return `${users.length} people are typing...`
})

watch(
  () => props.messages.length,
  () => {
    nextTick(() => {
      scrollToBottom()
    })
  }
)

function scrollToBottom() {
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
  }
}

async function sendMessage() {
  const content = messageInput.value.trim()
  if (!content) return

  messageInput.value = ''

  try {
    await chatStore.sendMessage({ content })
    // Reset typing timestamp so next input will send typing immediately
    lastTypingSentAt.value = 0
  } catch {
    // Restore message on error
    messageInput.value = content
  }
}

function handleInput() {
  const now = Date.now()

  // Send typing indicator only if 5 seconds have passed since last send
  if (now - lastTypingSentAt.value >= TYPING_SEND_INTERVAL) {
    chatStore.sendTyping(true)
    lastTypingSentAt.value = now
  }
}

function handleKeyDown(event: KeyboardEvent) {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    sendMessage()
  }
}
</script>

<template>
  <div class="flex h-full">
    <!-- Main chat area -->
    <div class="flex-1 flex flex-col min-w-0">
      <!-- Header -->
      <div class="px-4 py-3 border-b bg-white flex items-center justify-between">
        <div
          data-testid="chat-header-clickable"
          class="cursor-pointer hover:bg-gray-50 rounded-lg px-2 py-1 -ml-2 transition-colors"
          @click="showParticipants = !showParticipants"
        >
          <h3 class="font-semibold text-gray-900 flex items-center gap-2">
            {{ chat.name }}
            <svg
              class="w-4 h-4 text-gray-400 transition-transform"
              :class="{ 'rotate-180': showParticipants }"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
            </svg>
          </h3>
          <p v-if="chat.description" class="text-sm text-gray-500">{{ chat.description }}</p>
          <p class="text-xs text-gray-400">{{ participants.length }} participants</p>
        </div>
        <div class="flex items-center gap-2">
          <button
            @click.stop="chatStore.toggleFavorite(chat.id)"
            class="p-2 text-gray-500 hover:text-yellow-500 rounded-lg hover:bg-gray-100"
            :class="{ 'text-yellow-500': chat.is_favorite }"
            title="Toggle favorite"
          >
            <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
              <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
            </svg>
          </button>
        </div>
      </div>

      <!-- Messages -->
      <div ref="messagesContainer" class="flex-1 overflow-y-auto p-4 space-y-4">
        <MessageItem
          v-for="message in sortedMessages"
          :key="message.id"
          :message="message"
          :is-own="message.sender_id === currentUser.id"
          :current-user="currentUser"
        />

        <div v-if="sortedMessages.length === 0" class="text-center text-gray-500 py-8">
          No messages yet. Start the conversation!
        </div>
      </div>

      <!-- Typing indicator -->
      <div v-if="typingIndicator" class="px-4 py-1 text-sm text-gray-500 italic">
        {{ typingIndicator }}
      </div>

      <!-- Input -->
      <div class="p-4 border-t bg-white">
        <div v-if="isGuest" class="text-center text-gray-500 py-2">
          Guests cannot send messages. Contact an admin to upgrade your account.
        </div>
        <div v-else class="flex items-end gap-2">
          <textarea
            v-model="messageInput"
            @input="handleInput"
            @keydown="handleKeyDown"
            placeholder="Type a message..."
            rows="1"
            class="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent resize-none"
          />
          <button
            @click="sendMessage"
            :disabled="!messageInput.trim()"
            class="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
            </svg>
          </button>
        </div>
      </div>
    </div>

    <!-- Participants panel -->
    <ParticipantsPanel
      v-if="showParticipants"
      :participants="participants"
      @close="showParticipants = false"
    />
  </div>
</template>
