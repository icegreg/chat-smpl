<script setup lang="ts">
import { ref, computed, nextTick, watch } from 'vue'
import type { Chat, Message, Participant, User, Thread } from '@/types'
import { useChatStore } from '@/stores/chat'
import { api } from '@/api/client'
import MessageItem from './MessageItem.vue'
import ParticipantsPanel from './ParticipantsPanel.vue'
import ThreadList from './ThreadList.vue'
import ThreadView from './ThreadView.vue'
import ForwardMessageModal from './ForwardMessageModal.vue'
import AdHocCallButton from './voice/AdHocCallButton.vue'
import ScheduledEventWidget from './voice/ScheduledEventWidget.vue'
import EventHistoryPanel from './EventHistoryPanel.vue'

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
const showThreads = ref(false)
const showHistory = ref(false)
const selectedThread = ref<Thread | null>(null)
const fileInputRef = ref<HTMLInputElement | null>(null)
const pendingFiles = ref<{ file: File; linkId: string; uploading: boolean }[]>([])
const isUploading = ref(false)
const replyToMessages = ref<Message[]>([])  // Support multiple replies
const forwardMessage = ref<Message | null>(null)
const showForwardModal = ref(false)

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

// Normalize participant role - handle both string names and numeric enum values from protobuf
function normalizeParticipantRole(role: string | number): string {
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
  return String(role).toLowerCase()
}

// Check if current user can moderate (system owner/moderator OR chat admin)
const isModerator = computed(() => {
  // Check system-wide role
  const userRole = props.currentUser.role
  if (userRole === 'owner' || userRole === 'moderator') {
    return true
  }

  // Check chat participant role (normalize to handle protobuf enum values)
  const currentParticipant = props.participants.find(p => p.user_id === props.currentUser.id)
  if (currentParticipant) {
    const normalizedRole = normalizeParticipantRole(currentParticipant.role)
    if (normalizedRole === 'admin') {
      return true
    }
  }

  return false
})

// Delete chat functionality
const deleteLoading = ref(false)

async function handleDeleteChat() {
  if (!confirm(`Удалить чат "${props.chat.name}"? Это действие нельзя отменить.`)) return

  deleteLoading.value = true
  try {
    await chatStore.deleteChat(props.chat.id)
    // Chat will be removed from list via WebSocket event
  } catch (e) {
    console.error('Failed to delete chat:', e)
    alert('Не удалось удалить чат')
  } finally {
    deleteLoading.value = false
  }
}

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
  const savedReplyTo = [...replyToMessages.value]
  messageInput.value = ''
  pendingFiles.value = []
  replyToMessages.value = []

  try {
    await chatStore.sendMessage({
      content,
      file_link_ids: fileLinkIds.length > 0 ? fileLinkIds : undefined,
      reply_to_ids: savedReplyTo.length > 0 ? savedReplyTo.map(m => m.id) : undefined
    })
    // Reset typing timestamp so next input will send typing immediately
    lastTypingSentAt.value = 0
  } catch {
    // Restore message and files on error
    messageInput.value = savedContent
    pendingFiles.value = savedFiles
    replyToMessages.value = savedReplyTo
  }
}

function handleReplyToMessage(message: Message) {
  // Toggle message in the reply list
  const index = replyToMessages.value.findIndex(m => m.id === message.id)
  if (index !== -1) {
    // Remove if already selected
    replyToMessages.value.splice(index, 1)
  } else {
    // Add to reply list
    replyToMessages.value.push(message)
  }
}

function removeReplyMessage(messageId: string) {
  const index = replyToMessages.value.findIndex(m => m.id === messageId)
  if (index !== -1) {
    replyToMessages.value.splice(index, 1)
  }
}

function cancelReply() {
  replyToMessages.value = []
}

function handleForwardMessage(message: Message) {
  forwardMessage.value = message
  showForwardModal.value = true
}

function closeForwardModal() {
  showForwardModal.value = false
  forwardMessage.value = null
}

async function handleForwardToChat(targetChatId: string, comment: string) {
  if (!forwardMessage.value) return

  try {
    await chatStore.forwardMessage(
      forwardMessage.value,
      props.chat.id,
      targetChatId,
      comment
    )
    closeForwardModal()
  } catch (e) {
    console.error('Failed to forward message:', e)
  }
}

function getReplyToSenderName(message: Message): string {
  if (message.sender_display_name) return message.sender_display_name
  if (message.sender?.display_name) return message.sender.display_name
  if (message.sender_username) return message.sender_username
  if (message.sender?.username) return message.sender.username
  return 'Unknown'
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

// Thread functions
function toggleThreads() {
  showThreads.value = !showThreads.value
  if (!showThreads.value) {
    selectedThread.value = null
  }
  // Close history when opening threads
  if (showThreads.value) {
    showHistory.value = false
  }
}

// History panel functions
function toggleHistory() {
  showHistory.value = !showHistory.value
  // Close threads when opening history
  if (showHistory.value) {
    showThreads.value = false
    selectedThread.value = null
  }
}

function selectThread(thread: Thread) {
  selectedThread.value = thread
}

function closeThreads() {
  showThreads.value = false
  selectedThread.value = null
}

function backToThreadList() {
  selectedThread.value = null
}

async function createThread() {
  try {
    const thread = await api.createThread(props.chat.id, {
      thread_type: 'user',
      title: 'New Discussion'
    })
    selectedThread.value = thread
  } catch (e) {
    console.error('Failed to create thread:', e)
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
          <!-- Voice call button with participant selection -->
          <AdHocCallButton :chat-id="chat.id" :chat-name="chat.name" />

          <button
            @click.stop="toggleThreads"
            data-testid="threads-button"
            class="p-2 text-gray-500 hover:text-indigo-600 rounded-lg hover:bg-gray-100"
            :class="{ 'text-indigo-600 bg-indigo-50': showThreads }"
            title="View threads"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
            </svg>
          </button>
          <button
            @click.stop="toggleHistory"
            data-testid="history-button"
            class="p-2 text-gray-500 hover:text-indigo-600 rounded-lg hover:bg-gray-100"
            :class="{ 'text-indigo-600 bg-indigo-50': showHistory }"
            title="View history"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          </button>
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
          <!-- Delete chat button (only for moderators) -->
          <button
            v-if="isModerator"
            @click.stop="handleDeleteChat"
            :disabled="deleteLoading"
            class="p-2 text-gray-500 hover:text-red-600 rounded-lg hover:bg-gray-100"
            title="Удалить чат"
          >
            <svg v-if="!deleteLoading" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
            <svg v-else class="w-5 h-5 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
            </svg>
          </button>
        </div>
      </div>

      <!-- Scheduled Event Widget -->
      <div class="px-4 py-2">
        <ScheduledEventWidget :chat-id="chat.id" />
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
            @reply="handleReplyToMessage"
            @forward="handleForwardMessage"
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
          <!-- Reply preview - multiple replies -->
          <div v-if="replyToMessages.length > 0" data-testid="reply-preview" class="mb-2 p-2 bg-gray-50 rounded-lg border-l-4 border-indigo-500">
            <div class="flex items-center justify-between mb-1">
              <span class="flex items-center gap-1 text-xs text-indigo-600 font-medium">
                <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6" />
                </svg>
                Replying to {{ replyToMessages.length }} message{{ replyToMessages.length > 1 ? 's' : '' }}
              </span>
              <button
                @click="cancelReply"
                data-testid="reply-preview-cancel"
                class="p-1 text-gray-400 hover:text-gray-600 rounded flex-shrink-0"
                title="Clear all replies"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
            <div class="space-y-1 max-h-24 overflow-y-auto">
              <div
                v-for="reply in replyToMessages"
                :key="reply.id"
                class="flex items-start gap-2 text-sm"
              >
                <div class="flex-1 min-w-0">
                  <span data-testid="reply-preview-sender" class="font-medium text-gray-700">{{ getReplyToSenderName(reply) }}:</span>
                  <span data-testid="reply-preview-content" class="text-gray-600 truncate ml-1">{{ reply.content }}</span>
                </div>
                <button
                  @click="removeReplyMessage(reply.id)"
                  class="p-0.5 text-gray-400 hover:text-gray-600 rounded flex-shrink-0"
                  title="Remove this reply"
                >
                  <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            </div>
          </div>

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
      :current-user="currentUser"
      :chat-id="chat.id"
      @close="showParticipants = false"
    />

    <!-- Thread panels -->
    <template v-if="showThreads">
      <!-- Thread view (when a thread is selected) -->
      <ThreadView
        v-if="selectedThread"
        :thread="selectedThread"
        :current-user="currentUser"
        @close="closeThreads"
        @back="backToThreadList"
      />
      <!-- Thread list (when no thread is selected) -->
      <ThreadList
        v-else
        :chat-id="chat.id"
        @close="closeThreads"
        @select-thread="selectThread"
        @create-thread="createThread"
      />
    </template>

    <!-- Forward message modal -->
    <ForwardMessageModal
      v-if="showForwardModal && forwardMessage"
      :message="forwardMessage"
      :current-chat-id="chat.id"
      @close="closeForwardModal"
      @forward="handleForwardToChat"
    />

    <!-- Event history panel -->
    <EventHistoryPanel
      v-if="showHistory"
      :chat-id="chat.id"
      :is-moderator="isModerator"
      @close="showHistory = false"
    />
  </div>
</template>
