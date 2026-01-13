<script setup lang="ts">
import type { ConferenceHistory } from '@/types'

defineProps<{
  conferences: ConferenceHistory[]
  loading: boolean
}>()

defineEmits<{
  select: [conferenceId: string]
}>()

function formatDateTime(dateStr: string): string {
  const date = new Date(dateStr)
  return date.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

function formatDuration(startedAt?: string, endedAt?: string): string {
  if (!startedAt) return ''
  const start = new Date(startedAt).getTime()
  const end = endedAt ? new Date(endedAt).getTime() : Date.now()
  const durationMs = end - start
  const minutes = Math.floor(durationMs / 60000)
  if (minutes < 60) return `${minutes} min`
  const hours = Math.floor(minutes / 60)
  const remainingMinutes = minutes % 60
  return remainingMinutes > 0 ? `${hours}h ${remainingMinutes}m` : `${hours}h`
}

function getStatusColor(status: string): string {
  return status === 'active' ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-600'
}
</script>

<template>
  <div>
    <!-- Loading state -->
    <div v-if="loading" class="flex justify-center py-12">
      <svg class="w-6 h-6 animate-spin text-indigo-600" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
      </svg>
    </div>

    <!-- Empty state -->
    <div v-else-if="conferences.length === 0" class="text-center py-12 px-4">
      <svg class="w-12 h-12 mx-auto text-gray-300 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z" />
      </svg>
      <p class="text-gray-500 text-sm">No events yet</p>
      <p class="text-gray-400 text-xs mt-1">Events will appear here after they end</p>
    </div>

    <!-- Conference list -->
    <ul v-else class="divide-y divide-gray-100">
      <li
        v-for="conf in conferences"
        :key="conf.id"
        @click="$emit('select', conf.id)"
        class="px-4 py-3 hover:bg-gray-50 cursor-pointer transition-colors"
      >
        <div class="flex items-start gap-3">
          <!-- Icon -->
          <div class="mt-0.5 p-2 bg-indigo-100 rounded-lg flex-shrink-0">
            <svg class="w-4 h-4 text-indigo-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
          </div>

          <!-- Content -->
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-2">
              <p class="text-sm font-medium text-gray-900 truncate">{{ conf.name }}</p>
              <span
                v-if="conf.status === 'active'"
                :class="['text-xs px-1.5 py-0.5 rounded-full', getStatusColor(conf.status)]"
              >
                Live
              </span>
            </div>
            <p class="text-xs text-gray-500 mt-0.5">
              {{ conf.started_at ? formatDateTime(conf.started_at) : formatDateTime(conf.created_at) }}
            </p>
            <div class="flex items-center gap-3 mt-1.5 text-xs text-gray-400">
              <span class="flex items-center gap-1">
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
                </svg>
                {{ conf.participant_count }}
              </span>
              <span v-if="conf.started_at" class="flex items-center gap-1">
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                {{ formatDuration(conf.started_at, conf.ended_at) }}
              </span>
            </div>
          </div>

          <!-- Arrow -->
          <svg class="w-4 h-4 text-gray-400 flex-shrink-0 mt-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
          </svg>
        </div>
      </li>
    </ul>
  </div>
</template>
