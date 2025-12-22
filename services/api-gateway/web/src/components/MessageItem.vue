<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import type { Message, User } from '@/types'
import { useChatStore } from '@/stores/chat'

const props = defineProps<{
  message: Message
  isOwn: boolean
  currentUser: User
  compact?: boolean // For thread view - smaller styling
}>()

// Store blob URLs for authenticated file access
const blobUrls = ref<Map<string, string>>(new Map())

const emit = defineEmits<{
  reply: [message: Message]
  forward: [message: Message]
}>()

const chatStore = useChatStore()
const showActions = ref(false)
const isEditing = ref(false)
const editContent = ref('')

const commonEmojis = ['👍', '❤️', '😂', '😮', '😢', '🎉']

function parseMessageDate(value: string | { seconds: number; nanos?: number } | undefined): Date | null {
  if (!value) return null

  // Handle protobuf timestamp format { seconds: number, nanos: number }
  if (typeof value === 'object' && 'seconds' in value) {
    return new Date(value.seconds * 1000)
  }

  // Handle ISO string format
  if (typeof value === 'string') {
    const date = new Date(value)
    if (!isNaN(date.getTime())) {
      return date
    }
  }

  return null
}

function formatTime(value: string | { seconds: number; nanos?: number } | undefined): string {
  const date = parseMessageDate(value)
  if (!date) return ''
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function getSenderName(): string {
  // Priority: sender_display_name > sender.display_name > sender_username > sender.username
  if (props.message.sender_display_name) {
    return props.message.sender_display_name
  }
  if (props.message.sender?.display_name) {
    return props.message.sender.display_name
  }
  if (props.message.sender_username) {
    return props.message.sender_username
  }
  if (props.message.sender?.username) {
    return props.message.sender.username
  }
  return 'Unknown'
}

function getSenderAvatar(): string | null {
  if (props.message.sender_avatar_url) {
    return props.message.sender_avatar_url
  }
  if (props.message.sender?.avatar_url) {
    return props.message.sender.avatar_url
  }
  return null
}

function getMessageTime(): string {
  // Try sent_at first, then created_at
  return formatTime(props.message.sent_at) || formatTime(props.message.created_at)
}

function startEdit() {
  editContent.value = props.message.content
  isEditing.value = true
}

async function saveEdit() {
  if (editContent.value.trim() && editContent.value !== props.message.content) {
    await chatStore.updateMessage(props.message.id, editContent.value)
  }
  isEditing.value = false
}

function cancelEdit() {
  isEditing.value = false
  editContent.value = ''
}

async function deleteMessage() {
  if (confirm('Are you sure you want to delete this message?')) {
    await chatStore.deleteMessage(props.message.id)
  }
}

async function addReaction(emoji: string) {
  await chatStore.addReaction(props.message.id, emoji)
  showActions.value = false
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function isImage(contentType: string): boolean {
  return contentType.startsWith('image/')
}

function getFileDownloadUrl(linkId: string): string {
  return `/api/files/${linkId}`
}

// Fetch file with auth and return blob URL
async function fetchFileAsBlobUrl(linkId: string): Promise<string | null> {
  const token = localStorage.getItem('access_token')
  if (!token) return null

  try {
    const response = await fetch(`/api/files/${linkId}`, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })

    if (!response.ok) return null

    const blob = await response.blob()
    return URL.createObjectURL(blob)
  } catch {
    return null
  }
}

// Get blob URL for file (for images that need auth)
function getFileBlobUrl(linkId: string): string | undefined {
  return blobUrls.value.get(linkId)
}

// Load blob URLs for image attachments
async function loadImageBlobUrls() {
  const attachments = props.message.file_attachments
  if (!attachments?.length) return

  for (const attachment of attachments) {
    if (isImage(attachment.content_type) && !blobUrls.value.has(attachment.link_id)) {
      const blobUrl = await fetchFileAsBlobUrl(attachment.link_id)
      if (blobUrl) {
        blobUrls.value.set(attachment.link_id, blobUrl)
      }
    }
  }
}

// Clean up blob URLs on unmount
function cleanupBlobUrls() {
  blobUrls.value.forEach((url) => {
    URL.revokeObjectURL(url)
  })
  blobUrls.value.clear()
}

// Load images on mount and when attachments change
onMounted(() => {
  loadImageBlobUrls()
})

onUnmounted(() => {
  cleanupBlobUrls()
})

watch(() => props.message.file_attachments, () => {
  loadImageBlobUrls()
}, { deep: true })

// Download file with authentication
async function downloadFile(linkId: string, filename: string) {
  const token = localStorage.getItem('access_token')
  if (!token) return

  try {
    const response = await fetch(`/api/files/${linkId}`, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })

    if (!response.ok) {
      console.error('Failed to download file:', response.status)
      return
    }

    const blob = await response.blob()
    const url = URL.createObjectURL(blob)

    // Create temporary link and trigger download
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  } catch (error) {
    console.error('Failed to download file:', error)
  }
}

function handleReply() {
  emit('reply', props.message)
  showActions.value = false
}

function handleForward() {
  emit('forward', props.message)
  showActions.value = false
}

function getReplyToSenderName(replyTo: Message): string {
  if (replyTo.sender_display_name) return replyTo.sender_display_name
  if (replyTo.sender?.display_name) return replyTo.sender.display_name
  if (replyTo.sender_username) return replyTo.sender_username
  if (replyTo.sender?.username) return replyTo.sender.username
  return 'Unknown'
}

function truncateContent(content: string, maxLength: number = 100): string {
  if (content.length <= maxLength) return content
  return content.substring(0, maxLength) + '...'
}

function getReplyToTime(replyTo: Message): string {
  return formatTime(replyTo.sent_at) || formatTime(replyTo.created_at)
}

function hasReplyToAttachments(replyTo: Message): boolean {
  return (replyTo.file_attachments?.length ?? 0) > 0
}

function getReplyToAttachmentInfo(replyTo: Message): string {
  const attachments = replyTo.file_attachments
  if (!attachments || attachments.length === 0) return ''
  if (attachments.length === 1) {
    const att = attachments[0]
    if (att.content_type.startsWith('image/')) return '📷 Photo'
    return `📎 ${att.original_filename}`
  }
  return `📎 ${attachments.length} files`
}

// Get reply messages - support both old reply_to and new reply_to_messages
function getReplyToMessages(): Message[] {
  // New format: array of reply messages
  if (props.message.reply_to_messages && props.message.reply_to_messages.length > 0) {
    return props.message.reply_to_messages
  }
  // Old format: single reply_to
  if (props.message.reply_to) {
    return [props.message.reply_to]
  }
  return []
}
</script>

<template>
  <div
    data-testid="message-item"
    class="flex gap-3"
    :class="{ 'flex-row-reverse': isOwn }"
    @mouseenter="showActions = true"
    @mouseleave="showActions = false"
  >
    <!-- Avatar -->
    <img
      v-if="getSenderAvatar()"
      data-testid="message-avatar"
      :src="getSenderAvatar()!"
      :alt="getSenderName()"
      class="w-8 h-8 rounded-full object-cover flex-shrink-0"
    />
    <div
      v-else
      data-testid="message-avatar-placeholder"
      class="w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0 text-white text-sm font-medium"
      :class="isOwn ? 'bg-indigo-500' : 'bg-gray-500'"
    >
      {{ getSenderName()[0].toUpperCase() }}
    </div>

    <!-- Content -->
    <div class="max-w-[70%]" :class="{ 'text-right': isOwn }">
      <!-- Sender name -->
      <div class="text-xs text-gray-500 mb-1">
        <span data-testid="message-sender-name">{{ getSenderName() }}</span>
        <span data-testid="message-time" class="ml-2">{{ getMessageTime() }}</span>
        <span v-if="message.is_edited" class="ml-1 italic">(edited)</span>
        <!-- Pending indicator -->
        <span v-if="message.is_pending" class="ml-2 inline-flex items-center text-yellow-600" data-testid="message-pending">
          <svg class="w-3 h-3 mr-1 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          Отправка...
        </span>
      </div>

      <!-- Message bubble -->
      <div
        class="relative rounded-lg px-4 py-2 inline-block"
        :class="isOwn ? 'bg-indigo-500 text-white' : 'bg-white border'"
      >
        <!-- Forwarded indicator -->
        <div
          v-if="message.is_forwarded || message.forwarded_from_id"
          data-testid="message-forwarded"
          class="mb-2 flex items-center gap-1 text-xs"
          :class="isOwn ? 'text-indigo-200' : 'text-gray-500'"
        >
          <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 5l7 7-7 7M5 5l7 7-7 7" />
          </svg>
          <span>Forwarded</span>
          <span v-if="message.forwarded_from?.original_sender_name" class="font-medium">
            from {{ message.forwarded_from.original_sender_name }}
          </span>
        </div>

        <!-- Reply-to quotes (supports multiple) -->
        <div
          v-if="getReplyToMessages().length > 0"
          data-testid="message-quote"
          class="mb-2 space-y-1"
        >
          <div
            v-for="replyTo in getReplyToMessages()"
            :key="replyTo.id"
            class="pl-2 border-l-2 text-xs rounded-r cursor-pointer hover:opacity-80 transition-opacity"
            :class="isOwn ? 'border-indigo-300 bg-indigo-400/20' : 'border-gray-300 bg-gray-100'"
          >
            <div class="flex items-center gap-1">
              <svg class="w-3 h-3 flex-shrink-0" :class="isOwn ? 'text-indigo-200' : 'text-gray-400'" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6" />
              </svg>
              <span data-testid="message-quote-sender" class="font-medium" :class="isOwn ? 'text-indigo-200' : 'text-gray-600'">
                {{ getReplyToSenderName(replyTo) }}
              </span>
              <span v-if="getReplyToTime(replyTo)" class="opacity-70">
                {{ getReplyToTime(replyTo) }}
              </span>
            </div>
            <div data-testid="message-quote-content" class="mt-0.5" :class="isOwn ? 'text-indigo-100' : 'text-gray-600'">
              <span v-if="replyTo.content" class="line-clamp-2">
                {{ truncateContent(replyTo.content, 150) }}
              </span>
              <span v-if="hasReplyToAttachments(replyTo)" class="block text-xs opacity-80 mt-0.5">
                {{ getReplyToAttachmentInfo(replyTo) }}
              </span>
            </div>
          </div>
        </div>

        <!-- Edit mode -->
        <div v-if="isEditing" class="min-w-[200px]">
          <textarea
            v-model="editContent"
            class="w-full px-2 py-1 text-gray-900 border rounded resize-none"
            rows="2"
            @keydown.enter.prevent="saveEdit"
            @keydown.escape="cancelEdit"
          />
          <div class="flex justify-end gap-2 mt-2">
            <button @click="cancelEdit" class="text-xs text-gray-500 hover:text-gray-700">Cancel</button>
            <button @click="saveEdit" class="text-xs text-indigo-600 hover:text-indigo-800">Save</button>
          </div>
        </div>

        <!-- Normal display -->
        <p v-else-if="message.content" class="whitespace-pre-wrap break-words" :class="{ 'text-left': !isOwn }">
          {{ message.content }}
        </p>

        <!-- File attachments -->
        <div v-if="message.file_attachments?.length" class="mt-2 space-y-2">
          <template v-for="attachment in message.file_attachments" :key="attachment.link_id">
            <!-- Image preview -->
            <a
              v-if="isImage(attachment.content_type)"
              :href="getFileDownloadUrl(attachment.link_id)"
              target="_blank"
              class="block"
              @click.prevent="downloadFile(attachment.link_id, attachment.original_filename)"
            >
              <img
                v-if="getFileBlobUrl(attachment.link_id)"
                :src="getFileBlobUrl(attachment.link_id)"
                :alt="attachment.original_filename"
                class="max-w-full max-h-64 rounded cursor-pointer hover:opacity-90 transition-opacity"
              />
              <!-- Loading placeholder while fetching image -->
              <div
                v-else
                class="w-32 h-32 bg-gray-200 rounded flex items-center justify-center animate-pulse"
              >
                <svg class="w-8 h-8 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
              </div>
            </a>
            <!-- Other file types -->
            <a
              v-else
              href="#"
              @click.prevent="downloadFile(attachment.link_id, attachment.original_filename)"
              class="flex items-center gap-2 p-2 rounded hover:bg-gray-100 transition-colors cursor-pointer"
              :class="isOwn ? 'bg-indigo-400/30 hover:bg-indigo-400/50' : 'bg-gray-50'"
            >
              <svg class="w-8 h-8 text-gray-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
              <div class="min-w-0 flex-1">
                <div class="text-sm font-medium truncate" :class="isOwn ? 'text-white' : 'text-gray-900'">
                  {{ attachment.original_filename }}
                </div>
                <div class="text-xs" :class="isOwn ? 'text-indigo-200' : 'text-gray-500'">
                  {{ formatFileSize(attachment.size) }}
                </div>
              </div>
              <svg class="w-5 h-5 flex-shrink-0" :class="isOwn ? 'text-indigo-200' : 'text-gray-400'" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
              </svg>
            </a>
          </template>
        </div>

        <!-- Actions -->
        <div
          v-if="showActions && !isEditing"
          class="absolute -top-8 flex items-center gap-1 bg-white border rounded-lg shadow-lg p-1"
          :class="isOwn ? 'right-0' : 'left-0'"
        >
          <!-- Quick reactions -->
          <button
            v-for="emoji in commonEmojis"
            :key="emoji"
            @click="addReaction(emoji)"
            class="p-1 hover:bg-gray-100 rounded text-sm"
          >
            {{ emoji }}
          </button>

          <!-- Reply button -->
          <button
            @click="handleReply"
            data-testid="reply-button"
            class="p-1 hover:bg-gray-100 rounded text-gray-500"
            title="Reply"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6" />
            </svg>
          </button>

          <!-- Forward button -->
          <button
            @click="handleForward"
            data-testid="forward-button"
            class="p-1 hover:bg-gray-100 rounded text-gray-500"
            title="Forward to another chat"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 5l7 7-7 7M5 5l7 7-7 7" />
            </svg>
          </button>

          <!-- Edit (own messages only) -->
          <button
            v-if="isOwn"
            @click="startEdit"
            class="p-1 hover:bg-gray-100 rounded text-gray-500"
            title="Edit"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
            </svg>
          </button>

          <!-- Delete (own messages only) -->
          <button
            v-if="isOwn"
            @click="deleteMessage"
            class="p-1 hover:bg-gray-100 rounded text-red-500"
            title="Delete"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
          </button>
        </div>
      </div>

      <!-- Reactions -->
      <div v-if="message.reactions?.length" class="flex flex-wrap gap-1 mt-1" :class="{ 'justify-end': isOwn }">
        <button
          v-for="reaction in message.reactions"
          :key="reaction.emoji"
          class="px-2 py-0.5 text-xs bg-gray-100 rounded-full hover:bg-gray-200 flex items-center gap-1"
        >
          <span>{{ reaction.emoji }}</span>
          <span class="text-gray-600">{{ reaction.count }}</span>
        </button>
      </div>
    </div>
  </div>
</template>
