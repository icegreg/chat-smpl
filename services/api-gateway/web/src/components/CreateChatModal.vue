<script setup lang="ts">
import { ref } from 'vue'
import type { Chat } from '@/types'
import { useChatStore } from '@/stores/chat'

const emit = defineEmits<{
  close: []
  created: [chat: Chat]
}>()

const chatStore = useChatStore()

const chatType = ref<'group' | 'channel'>('group')
const name = ref('')
const description = ref('')
const participantIds = ref('')
const error = ref('')
const loading = ref(false)

async function handleSubmit() {
  error.value = ''

  if (!name.value.trim()) {
    error.value = 'Name is required'
    return
  }

  loading.value = true

  try {
    const participants = participantIds.value
      .split(',')
      .map((id) => id.trim())
      .filter((id) => id.length > 0)

    const chat = await chatStore.createChat({
      type: chatType.value,
      name: name.value.trim(),
      description: description.value.trim() || undefined,
      participant_ids: participants,
    })

    emit('created', chat)
  } catch (e) {
    error.value = chatStore.error || 'Failed to create chat'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" @click.self="$emit('close')">
    <div class="bg-white rounded-lg shadow-xl w-full max-w-md mx-4">
      <div class="px-6 py-4 border-b flex items-center justify-between">
        <h2 class="text-lg font-semibold text-gray-900">Create New Chat</h2>
        <button @click="$emit('close')" class="text-gray-500 hover:text-gray-700">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <form @submit.prevent="handleSubmit" class="p-6 space-y-4">
        <div v-if="error" class="p-3 bg-red-50 text-red-700 rounded-lg text-sm">
          {{ error }}
        </div>

        <!-- Chat type -->
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-2">Type</label>
          <div class="flex gap-4">
            <label class="flex items-center">
              <input
                v-model="chatType"
                type="radio"
                value="group"
                class="mr-2 text-indigo-600 focus:ring-indigo-500"
              />
              <span class="text-sm text-gray-700">Group</span>
            </label>
            <label class="flex items-center">
              <input
                v-model="chatType"
                type="radio"
                value="channel"
                class="mr-2 text-indigo-600 focus:ring-indigo-500"
              />
              <span class="text-sm text-gray-700">Channel</span>
            </label>
          </div>
        </div>

        <!-- Name -->
        <div>
          <label for="name" class="block text-sm font-medium text-gray-700 mb-1">Name</label>
          <input
            id="name"
            v-model="name"
            type="text"
            required
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
            placeholder="Enter chat name"
          />
        </div>

        <!-- Description -->
        <div>
          <label for="description" class="block text-sm font-medium text-gray-700 mb-1">
            Description (optional)
          </label>
          <textarea
            id="description"
            v-model="description"
            rows="2"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent resize-none"
            placeholder="What's this chat about?"
          />
        </div>

        <!-- Participants -->
        <div>
          <label for="participants" class="block text-sm font-medium text-gray-700 mb-1">
            Participant IDs (optional)
          </label>
          <input
            id="participants"
            v-model="participantIds"
            type="text"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
            placeholder="Comma-separated user IDs"
          />
          <p class="mt-1 text-xs text-gray-500">Enter user IDs separated by commas to invite them</p>
        </div>

        <!-- Actions -->
        <div class="flex justify-end gap-3 pt-4">
          <button
            type="button"
            @click="$emit('close')"
            class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
          >
            Cancel
          </button>
          <button
            type="submit"
            :disabled="loading"
            class="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 disabled:opacity-50 transition-colors"
          >
            <span v-if="loading">Creating...</span>
            <span v-else>Create Chat</span>
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
