<script setup lang="ts">
import { ref, computed, nextTick, watch } from 'vue'
import type { Chat, Message, Participant, User } from '@/types'
import { useChatStore } from '@/stores/chat'
import { api } from '@/api/client'
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
const fileInputRef = ref<HTMLInputElement | null>(null)
const pendingFiles = ref<{ file: File; linkId: string; uploading: boolean }[]>([])
const isUploading = ref(false)

const TYPING_SEND_INTERVAL = 5000 // Send typing indicator every 5 seconds

interface MessageGroup {
  date: string
  dateLabel: string
  messages: Message[]
}

function getMessageDate(message: Message): Date {
  // Try sent_at first (protobuf timestamp format), then created_at
  if (message.sent_at) {
    if (typeof message.sent_at === 'object' && 'seconds' in message.sent_at) {
      return new Date((message.sent_at as { seconds: number }).seconds * 1000)
    }
    return new Date(message.sent_at as string)
  }
  return new Date(message.created_at)
}

function formatDateLabel(date: Date): string {
  const today = new Date()
  const yesterday = new Date(today)
  yesterday.setDate(yesterday.getDate() - 1)

  const isToday = date.toDateString() === today.toDateString()
  const isYesterday = date.toDateString() === yesterday.toDateString()

  if (isToday) {
    return 'Today'
  }
  if (isYesterday) {
    return 'Yesterday'
  }

  // For other dates, show full date
  return date.toLocaleDateString(undefined, {
    weekday: 'long',
    year: 'numeric',
    month: 'long',
    day: 'numeric'
  })
}

const sortedMessages = computed(() => {
  return [...props.messages].sort(
    (a, b) => getMessageDate(a).getTime() - getMessageDate(b).getTime()
  )
})

const groupedMessages = computed((): MessageGroup[] => {
  const groups: MessageGroup[] = []
  let currentDateStr = ''
  let currentGroup: MessageGroup | null = null

  for (const message of sortedMessages.value) {
    const msgDate = getMessageDate(message)
    const dateStr = msgDate.toDateString()

    if (dateStr !== currentDateStr) {
      currentDateStr = dateStr
      currentGroup = {
        date: dateStr,
        dateLabel: formatDateLabel(msgDate),
        messages: []
      }
      groups.push(currentGroup)
    }

    currentGroup!.messages.push(message)
  }

  return groups
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
  const fileLinkIds = pendingFiles.value.filter(f => f.linkId).map(f => f.linkId)

  if (!content && fileLinkIds.length === 0) return

  const savedContent = messageInput.value
  const savedFiles = [...pendingFiles.value]
  messageInput.value = ''
  pendingFiles.value = []

  try {
    await chatStore.sendMessage({ content, file_link_ids: fileLinkIds.length > 0 ? fileLinkIds : undefined })
    // Reset typing timestamp so next input will send typing immediately
    lastTypingSentAt.value = 0
  } catch {
    // Restore message and files on error
    messageInput.value = savedContent
    pendingFiles.value = savedFiles
  }
}

function openFilePicker() {
  fileInputRef.value?.click()
}

async function handleFileSelect(event: Event) {
  const input = event.target as HTMLInputElement
  const files = input.files
  if (!files || files.length === 0) return

  for (const file of Array.from(files)) {
    await uploadFile(file)
  }

  // Reset input
  input.value = ''
}

async function uploadFile(file: File) {
  const pendingFile = { file, linkId: '', uploading: true }
  pendingFiles.value.push(pendingFile)
  isUploading.value = true

  try {
    const result = await api.uploadFile(file)
    // Find and update the file in the array to trigger reactivity
    const index = pendingFiles.value.findIndex(f => f.file === file)
    if (index !== -1) {
      pendingFiles.value[index] = { ...pendingFiles.value[index], linkId: result.link_id, uploading: false }
    }
  } catch (error) {
    console.error('Failed to upload file:', error)
    // Remove failed file from pending
    const index = pendingFiles.value.findIndex(f => f.file === file)
    if (index > -1) {
      pendingFiles.value.splice(index, 1)
    }
  } finally {
    isUploading.value = pendingFiles.value.some(f => f.uploading)
  }
}

function removeFile(index: number) {
  pendingFiles.value.splice(index, 1)
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function getFilePreviewUrl(file: File): string {
  return URL.createObjectURL(file)
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
        <template v-for="group in groupedMessages" :key="group.date">
          <!-- Date separator -->
          <div data-testid="date-separator" class="flex items-center justify-center my-4">
            <div class="flex-1 border-t border-gray-200"></div>
            <span data-testid="date-label" class="px-4 text-xs text-gray-500 bg-gray-50 rounded-full py-1">
              {{ group.dateLabel }}
            </span>
            <div class="flex-1 border-t border-gray-200"></div>
          </div>

          <!-- Messages for this date -->
          <MessageItem
            v-for="message in group.messages"
            :key="message.id"
            :message="message"
            :is-own="message.sender_id === currentUser.id"
            :current-user="currentUser"
          />
        </template>

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
        <div v-else>
          <!-- Pending files preview -->
          <div v-if="pendingFiles.length > 0" data-testid="pending-files-container" class="mb-2 flex flex-wrap gap-2">
            <div
              v-for="(pf, index) in pendingFiles"
              :key="index"
              data-testid="pending-file"
              class="relative flex items-center gap-2 px-3 py-2 bg-gray-100 rounded-lg text-sm"
            >
              <!-- File icon or image preview -->
              <div v-if="pf.file.type.startsWith('image/')" class="w-8 h-8 rounded overflow-hidden">
                <img
                  :src="getFilePreviewUrl(pf.file)"
                  class="w-full h-full object-cover"
                />
              </div>
              <svg v-else class="w-5 h-5 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
              <div class="flex flex-col">
                <span class="truncate max-w-[150px]">{{ pf.file.name }}</span>
                <span class="text-xs text-gray-500">{{ formatFileSize(pf.file.size) }}</span>
              </div>
              <!-- Loading indicator -->
              <svg v-if="pf.uploading" data-testid="file-uploading-spinner" class="w-4 h-4 animate-spin text-indigo-600" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
              <!-- Remove button -->
              <button
                v-else
                data-testid="remove-pending-file"
                @click="removeFile(index)"
                class="ml-1 text-gray-400 hover:text-gray-600"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
          </div>

          <div class="flex items-end gap-2">
            <!-- Hidden file input -->
            <input
              ref="fileInputRef"
              type="file"
              multiple
              class="hidden"
              @change="handleFileSelect"
            />

            <!-- Attach file button -->
            <button
              @click="openFilePicker"
              class="p-2 text-gray-500 hover:text-gray-700 rounded-lg hover:bg-gray-100"
              title="Attach file"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13" />
              </svg>
            </button>

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
              :disabled="!messageInput.trim() && pendingFiles.length === 0"
              class="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
              </svg>
            </button>
          </div>
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
