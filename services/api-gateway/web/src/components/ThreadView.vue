<script setup lang="ts">
import { ref, computed, onMounted, watch, nextTick } from 'vue'
import type { Thread, Message, User, ProtobufTimestamp } from '@/types'
import { api } from '@/api/client'
import { useChatStore } from '@/stores/chat'
import MessageItem from './MessageItem.vue'

const props = defineProps<{
  thread: Thread
  currentUser: User
}>()

defineEmits<{
  close: []
  back: []
  navigateToSubthreads: [thread: Thread]
}>()

const chatStore = useChatStore()
const messages = ref<Message[]>([])
const messageInput = ref('')
const loading = ref(false)
const sending = ref(false)
const error = ref<string | null>(null)
const messagesContainer = ref<HTMLElement | null>(null)

const isSystemThread = computed(() => props.thread.thread_type === 'system')

function parseTimestamp(ts: string | ProtobufTimestamp | undefined): Date {
  if (!ts) return new Date(0)
  if (typeof ts === 'string') return new Date(ts)
  if (typeof ts === 'object' && 'seconds' in ts) {
    return new Date(ts.seconds * 1000 + (ts.nanos || 0) / 1000000)
  }
  return new Date(0)
}

const sortedMessages = computed(() => {
  return [...messages.value].sort((a, b) => {
    const dateA = parseTimestamp(a.sent_at || a.created_at)
    const dateB = parseTimestamp(b.sent_at || b.created_at)
    return dateA.getTime() - dateB.getTime()
  })
})

async function loadMessages() {
  loading.value = true
  error.value = null
  try {
    const result = await api.getThreadMessages(props.thread.id)
    messages.value = result.messages || []
    nextTick(scrollToBottom)
  } catch (e) {
    error.value = 'Failed to load messages'
    console.error('Failed to load thread messages:', e)
  } finally {
    loading.value = false
  }
}

function scrollToBottom() {
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
  }
}

async function sendMessage() {
  const content = messageInput.value.trim()
  if (!content || isSystemThread.value) return

  sending.value = true
  const savedContent = messageInput.value
  messageInput.value = ''

  try {
    // Send message to chat with thread_id
    await chatStore.sendMessage({
      content,
      thread_id: props.thread.id
    })
    // Reload messages to get the new one
    await loadMessages()
  } catch (e) {
    messageInput.value = savedContent
    console.error('Failed to send message:', e)
  } finally {
    sending.value = false
  }
}

function handleKeyDown(event: KeyboardEvent) {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    sendMessage()
  }
}

function getThreadTitle(): string {
  if (props.thread.title) {
    return props.thread.title
  }
  if (props.thread.thread_type === 'system') {
    return 'Activity'
  }
  return 'Thread'
}

watch(() => props.thread.id, loadMessages)

onMounted(loadMessages)
</script>

<template>
  <div class="w-96 border-l flex flex-col h-full" :class="isSystemThread ? 'bg-orange-50' : 'bg-indigo-50'">
    <!-- Thread indicator banner -->
    <div
      class="px-3 py-1.5 text-xs font-medium flex items-center gap-2"
      :class="isSystemThread ? 'bg-orange-100 text-orange-700' : 'bg-indigo-100 text-indigo-700'"
    >
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
      </svg>
      <span v-if="isSystemThread">System Activity Thread</span>
      <span v-else-if="thread.depth > 0">Subthread (Level {{ thread.depth }})</span>
      <span v-else>Discussion Thread</span>
    </div>

    <!-- Header -->
    <div class="px-4 py-3 border-b bg-white/80 flex items-center gap-3">
      <button
        @click="$emit('back')"
        class="p-1 text-gray-400 hover:text-gray-600 rounded"
      >
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
        </svg>
      </button>
      <div class="flex-1 min-w-0">
        <div class="flex items-center gap-2">
          <!-- Thread icon -->
          <div
            :class="isSystemThread ? 'bg-orange-100' : 'bg-indigo-100'"
            class="p-1.5 rounded-lg"
          >
            <!-- System/Activity icon -->
            <svg v-if="isSystemThread" class="w-4 h-4 text-orange-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
            </svg>
            <!-- Reply thread icon -->
            <svg v-else-if="thread.parent_message_id" class="w-4 h-4 text-indigo-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6" />
            </svg>
            <!-- Standalone thread icon -->
            <svg v-else class="w-4 h-4 text-indigo-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
            </svg>
          </div>
          <div>
            <h4 class="font-semibold text-gray-900 truncate">{{ getThreadTitle() }}</h4>
            <p class="text-xs text-gray-500">
              {{ thread.message_count }} {{ isSystemThread ? 'events' : 'messages' }}
              <span v-if="thread.depth > 0" class="text-purple-600">&bull; Level {{ thread.depth }}</span>
            </p>
          </div>
        </div>
      </div>
      <div class="flex items-center gap-1">
        <!-- Navigate to subthreads button (only for user threads not at max depth) -->
        <button
          v-if="!isSystemThread && thread.depth < 5"
          @click="$emit('navigateToSubthreads', thread)"
          class="p-1 text-gray-400 hover:text-indigo-600 rounded"
          title="View/Create subthreads"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 4v16M7 8H17v8H7" />
          </svg>
        </button>
        <button
          @click="$emit('close')"
          class="p-1 text-gray-400 hover:text-gray-600 rounded"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    </div>

    <!-- Messages area -->
    <div ref="messagesContainer" class="flex-1 overflow-y-auto p-4 space-y-3 bg-white/60">
      <!-- Loading state -->
      <div v-if="loading" class="flex items-center justify-center py-8">
        <svg class="w-6 h-6 animate-spin text-indigo-600" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
      </div>

      <!-- Error state -->
      <div v-else-if="error" class="px-4 py-8 text-center text-red-500 text-sm">
        {{ error }}
        <button @click="loadMessages" class="block mx-auto mt-2 text-indigo-600 hover:underline">
          Retry
        </button>
      </div>

      <!-- Messages list -->
      <template v-else>
        <!-- System messages display differently -->
        <template v-if="isSystemThread">
          <div
            v-for="message in sortedMessages"
            :key="message.id"
            class="flex items-start gap-2 text-sm"
          >
            <div class="mt-1 w-2 h-2 rounded-full bg-orange-400 flex-shrink-0"></div>
            <div class="flex-1">
              <p class="text-gray-700">{{ message.content }}</p>
              <p class="text-xs text-gray-400 mt-0.5">
                {{ parseTimestamp(message.sent_at || message.created_at).toLocaleString() }}
              </p>
            </div>
          </div>
        </template>

        <!-- Regular user thread messages -->
        <template v-else>
          <MessageItem
            v-for="message in sortedMessages"
            :key="message.id"
            :message="message"
            :is-own="message.sender_id === currentUser.id"
            :current-user="currentUser"
            :compact="true"
          />
        </template>

        <!-- Empty state -->
        <div v-if="sortedMessages.length === 0" class="text-center text-gray-500 py-8">
          <p v-if="isSystemThread">No activity yet</p>
          <p v-else>No messages yet. Start the conversation!</p>
        </div>
      </template>
    </div>

    <!-- Input (only for user threads) -->
    <div v-if="!isSystemThread" class="p-4 border-t bg-white">
      <div class="flex items-end gap-2">
        <textarea
          v-model="messageInput"
          @keydown="handleKeyDown"
          placeholder="Reply to thread..."
          rows="1"
          :disabled="sending"
          class="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent resize-none disabled:opacity-50"
        />
        <button
          @click="sendMessage"
          :disabled="!messageInput.trim() || sending"
          class="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          <svg v-if="sending" class="w-5 h-5 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
          </svg>
        </button>
      </div>
    </div>

    <!-- System thread info -->
    <div v-else class="px-4 py-3 border-t bg-gray-50 text-center text-xs text-gray-500">
      System activity is logged automatically
    </div>
  </div>
</template>
