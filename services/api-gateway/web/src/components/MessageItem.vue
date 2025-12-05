<script setup lang="ts">
import { ref } from 'vue'
import type { Message, User } from '@/types'
import { useChatStore } from '@/stores/chat'

const props = defineProps<{
  message: Message
  isOwn: boolean
  currentUser: User
}>()

const chatStore = useChatStore()
const showActions = ref(false)
const isEditing = ref(false)
const editContent = ref('')

const commonEmojis = ['👍', '❤️', '😂', '😮', '😢', '🎉']

function formatTime(dateString: string | undefined): string {
  if (!dateString) return ''
  const date = new Date(dateString)
  if (isNaN(date.getTime())) return ''
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
</script>

<template>
  <div
    class="flex gap-3"
    :class="{ 'flex-row-reverse': isOwn }"
    @mouseenter="showActions = true"
    @mouseleave="showActions = false"
  >
    <!-- Avatar -->
    <img
      v-if="getSenderAvatar()"
      :src="getSenderAvatar()!"
      :alt="getSenderName()"
      class="w-8 h-8 rounded-full object-cover flex-shrink-0"
    />
    <div
      v-else
      class="w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0 text-white text-sm font-medium"
      :class="isOwn ? 'bg-indigo-500' : 'bg-gray-500'"
    >
      {{ getSenderName()[0].toUpperCase() }}
    </div>

    <!-- Content -->
    <div class="max-w-[70%]" :class="{ 'text-right': isOwn }">
      <!-- Sender name -->
      <div class="text-xs text-gray-500 mb-1">
        {{ getSenderName() }}
        <span class="ml-2">{{ getMessageTime() }}</span>
        <span v-if="message.is_edited" class="ml-1 italic">(edited)</span>
      </div>

      <!-- Message bubble -->
      <div
        class="relative rounded-lg px-4 py-2 inline-block"
        :class="isOwn ? 'bg-indigo-500 text-white' : 'bg-white border'"
      >
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
        <p v-else class="whitespace-pre-wrap break-words" :class="{ 'text-left': !isOwn }">
          {{ message.content }}
        </p>

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
