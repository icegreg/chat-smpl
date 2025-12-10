<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import type { Thread } from '@/types'
import { api } from '@/api/client'

const props = defineProps<{
  chatId: string
  parentThread?: Thread // For displaying subthreads
}>()

defineEmits<{
  close: []
  selectThread: [thread: Thread]
  createThread: []
  navigateToSubthreads: [thread: Thread]
  navigateBack: []
}>()

const threads = ref<Thread[]>([])
const subthreadCounts = ref<Map<string, number>>(new Map())
const loading = ref(false)
const error = ref<string | null>(null)
const showSystemThreads = ref(true)

const filteredThreads = computed(() => {
  if (showSystemThreads.value) {
    return threads.value
  }
  return threads.value.filter(t => t.thread_type !== 'system')
})

const userThreads = computed(() => filteredThreads.value.filter(t => t.thread_type === 'user'))
const systemThreads = computed(() => filteredThreads.value.filter(t => t.thread_type === 'system'))

async function loadThreads() {
  loading.value = true
  error.value = null
  try {
    let result
    if (props.parentThread) {
      // Load subthreads of the parent thread
      result = await api.listSubthreads(props.parentThread.id)
    } else {
      // Load top-level threads
      result = await api.listThreads(props.chatId)
    }
    threads.value = result.threads || []

    // Load subthread counts for user threads (to show "has subthreads" indicator)
    if (!props.parentThread) {
      const counts = new Map<string, number>()
      for (const thread of threads.value.filter(t => t.thread_type === 'user')) {
        try {
          const subResult = await api.listSubthreads(thread.id, 1, 1)
          if (subResult.total > 0) {
            counts.set(thread.id, subResult.total)
          }
        } catch {
          // Ignore errors for subthread count
        }
      }
      subthreadCounts.value = counts
    }
  } catch (e) {
    error.value = 'Failed to load threads'
    console.error('Failed to load threads:', e)
  } finally {
    loading.value = false
  }
}

function hasSubthreads(thread: Thread): boolean {
  return (subthreadCounts.value.get(thread.id) || 0) > 0
}

function getSubthreadCount(thread: Thread): number {
  return subthreadCounts.value.get(thread.id) || 0
}

function formatDate(dateString: string | undefined): string {
  if (!dateString) return ''
  const date = new Date(dateString)
  const now = new Date()
  const diff = now.getTime() - date.getTime()

  // Less than 1 hour
  if (diff < 3600000) {
    const minutes = Math.floor(diff / 60000)
    return minutes <= 1 ? 'Just now' : `${minutes}m ago`
  }
  // Less than 24 hours
  if (diff < 86400000) {
    const hours = Math.floor(diff / 3600000)
    return `${hours}h ago`
  }
  // Less than 7 days
  if (diff < 604800000) {
    const days = Math.floor(diff / 86400000)
    return `${days}d ago`
  }
  // Older
  return date.toLocaleDateString()
}

function getThreadTitle(thread: Thread): string {
  if (thread.title) {
    return thread.title
  }
  if (thread.thread_type === 'system') {
    return 'Activity'
  }
  return 'Thread'
}

watch(() => props.chatId, loadThreads)
watch(() => props.parentThread?.id, loadThreads)

onMounted(loadThreads)

// Computed for header title
const headerTitle = computed(() => {
  if (props.parentThread) {
    return props.parentThread.title || 'Subthreads'
  }
  return 'Threads'
})
</script>

<template>
  <div class="w-72 border-l bg-white flex flex-col h-full">
    <!-- Header -->
    <div class="px-4 py-3 border-b flex items-center justify-between">
      <div class="flex items-center gap-2">
        <!-- Back button for subthreads -->
        <button
          v-if="parentThread"
          @click="$emit('navigateBack')"
          class="p-1 text-gray-400 hover:text-gray-600 rounded"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
          </svg>
        </button>
        <div>
          <h4 class="font-semibold text-gray-900">{{ headerTitle }}</h4>
          <p v-if="parentThread" class="text-xs text-gray-500">Depth: {{ parentThread.depth + 1 }}</p>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <button
          @click="$emit('createThread')"
          class="p-1 text-gray-400 hover:text-indigo-600 rounded"
          title="Create new thread"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
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

    <!-- Filter toggle -->
    <div class="px-4 py-2 border-b">
      <label class="flex items-center gap-2 text-sm text-gray-600 cursor-pointer">
        <input
          v-model="showSystemThreads"
          type="checkbox"
          class="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
        />
        Show activity threads
      </label>
    </div>

    <!-- Content -->
    <div class="flex-1 overflow-y-auto">
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
        <button @click="loadThreads" class="block mx-auto mt-2 text-indigo-600 hover:underline">
          Retry
        </button>
      </div>

      <!-- Threads list -->
      <template v-else>
        <!-- System threads section -->
        <div v-if="systemThreads.length > 0">
          <div class="px-4 py-2 text-xs font-medium text-gray-500 uppercase tracking-wider bg-gray-50">
            Activity
          </div>
          <ul>
            <li
              v-for="thread in systemThreads"
              :key="thread.id"
              @click="$emit('selectThread', thread)"
              class="px-4 py-3 hover:bg-gray-50 cursor-pointer border-b border-gray-100"
            >
              <div class="flex items-start gap-3">
                <div class="mt-0.5 p-1.5 bg-orange-100 rounded-lg">
                  <svg class="w-4 h-4 text-orange-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
                  </svg>
                </div>
                <div class="flex-1 min-w-0">
                  <div class="flex items-center justify-between">
                    <p class="text-sm font-medium text-gray-900 truncate">
                      {{ getThreadTitle(thread) }}
                    </p>
                    <span class="text-xs text-gray-400">
                      {{ formatDate(thread.last_message_at) }}
                    </span>
                  </div>
                  <p class="text-xs text-gray-500 mt-0.5">
                    {{ thread.message_count }} {{ thread.message_count === 1 ? 'event' : 'events' }}
                  </p>
                </div>
              </div>
            </li>
          </ul>
        </div>

        <!-- User threads section -->
        <div v-if="userThreads.length > 0">
          <div class="px-4 py-2 text-xs font-medium text-gray-500 uppercase tracking-wider bg-gray-50">
            {{ parentThread ? 'Subthreads' : 'Discussions' }}
          </div>
          <ul>
            <li
              v-for="thread in userThreads"
              :key="thread.id"
              class="px-4 py-3 hover:bg-gray-50 border-b border-gray-100"
            >
              <div class="flex items-start gap-3">
                <div
                  class="mt-0.5 p-1.5 rounded-lg cursor-pointer"
                  :class="thread.depth > 0 ? 'bg-purple-100' : 'bg-indigo-100'"
                  @click="$emit('selectThread', thread)"
                >
                  <!-- Subthread icon -->
                  <svg v-if="thread.depth > 0" class="w-4 h-4 text-purple-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 4v16M7 8H17v8H7" />
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
                <div class="flex-1 min-w-0 cursor-pointer" @click="$emit('selectThread', thread)">
                  <div class="flex items-center justify-between">
                    <p class="text-sm font-medium text-gray-900 truncate">
                      {{ getThreadTitle(thread) }}
                    </p>
                    <span class="text-xs text-gray-400">
                      {{ formatDate(thread.last_message_at) }}
                    </span>
                  </div>
                  <p class="text-xs text-gray-500 mt-0.5">
                    {{ thread.message_count }} {{ thread.message_count === 1 ? 'message' : 'messages' }}
                  </p>
                  <div class="flex items-center gap-2 mt-1">
                    <span
                      v-if="thread.is_archived"
                      class="inline-block px-1.5 py-0.5 text-xs bg-gray-100 text-gray-600 rounded"
                    >
                      Archived
                    </span>
                    <span
                      v-if="thread.depth > 0"
                      class="inline-block px-1.5 py-0.5 text-xs bg-purple-100 text-purple-700 rounded"
                    >
                      Level {{ thread.depth }}
                    </span>
                  </div>
                </div>
                <!-- Subthreads navigation button -->
                <button
                  v-if="hasSubthreads(thread) || thread.depth < 5"
                  @click.stop="$emit('navigateToSubthreads', thread)"
                  class="flex items-center gap-1 px-2 py-1 text-xs text-indigo-600 hover:bg-indigo-50 rounded"
                  :title="hasSubthreads(thread) ? `${getSubthreadCount(thread)} subthreads` : 'Create subthread'"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
                  </svg>
                  <span v-if="hasSubthreads(thread)">{{ getSubthreadCount(thread) }}</span>
                </button>
              </div>
            </li>
          </ul>
        </div>

        <!-- Empty state -->
        <div v-if="filteredThreads.length === 0" class="px-4 py-8 text-center text-gray-500 text-sm">
          <svg class="w-12 h-12 mx-auto text-gray-300 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
          </svg>
          <p>No threads yet</p>
          <button
            @click="$emit('createThread')"
            class="mt-2 text-indigo-600 hover:underline"
          >
            Start a new thread
          </button>
        </div>
      </template>
    </div>
  </div>
</template>
