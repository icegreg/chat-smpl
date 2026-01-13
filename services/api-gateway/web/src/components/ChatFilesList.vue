<script setup lang="ts">
import type { ChatFile } from '@/types'

defineProps<{
  files: ChatFile[]
  loading: boolean
}>()

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString(undefined, {
    month: 'short',
    day: 'numeric',
    year: 'numeric'
  })
}

function getFileIcon(filename: string): string {
  const ext = filename.split('.').pop()?.toLowerCase() || ''

  // Images
  if (['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp'].includes(ext)) {
    return 'image'
  }
  // Documents
  if (['pdf', 'doc', 'docx', 'txt', 'rtf', 'odt'].includes(ext)) {
    return 'document'
  }
  // Spreadsheets
  if (['xls', 'xlsx', 'csv', 'ods'].includes(ext)) {
    return 'spreadsheet'
  }
  // Archives
  if (['zip', 'rar', '7z', 'tar', 'gz'].includes(ext)) {
    return 'archive'
  }
  // Audio
  if (['mp3', 'wav', 'ogg', 'flac', 'aac'].includes(ext)) {
    return 'audio'
  }
  // Video
  if (['mp4', 'avi', 'mkv', 'mov', 'webm'].includes(ext)) {
    return 'video'
  }

  return 'file'
}

function downloadFile(file: ChatFile) {
  window.open(`/api/files/${file.link_id}/download`, '_blank')
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
    <div v-else-if="files.length === 0" class="text-center py-12 px-4">
      <svg class="w-12 h-12 mx-auto text-gray-300 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
      </svg>
      <p class="text-gray-500 text-sm">No files in this chat</p>
      <p class="text-gray-400 text-xs mt-1">Uploaded files will appear here</p>
    </div>

    <!-- Files list -->
    <ul v-else class="divide-y divide-gray-100">
      <li
        v-for="file in files"
        :key="file.id"
        @click="downloadFile(file)"
        class="px-4 py-3 hover:bg-gray-50 cursor-pointer transition-colors"
      >
        <div class="flex items-start gap-3">
          <!-- File type icon -->
          <div class="mt-0.5 p-2 rounded-lg flex-shrink-0" :class="{
            'bg-blue-100': getFileIcon(file.filename) === 'image',
            'bg-red-100': getFileIcon(file.filename) === 'document',
            'bg-green-100': getFileIcon(file.filename) === 'spreadsheet',
            'bg-yellow-100': getFileIcon(file.filename) === 'archive',
            'bg-purple-100': getFileIcon(file.filename) === 'audio',
            'bg-pink-100': getFileIcon(file.filename) === 'video',
            'bg-gray-100': getFileIcon(file.filename) === 'file'
          }">
            <!-- Image icon -->
            <svg v-if="getFileIcon(file.filename) === 'image'" class="w-4 h-4 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            <!-- Document icon -->
            <svg v-else-if="getFileIcon(file.filename) === 'document'" class="w-4 h-4 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
            </svg>
            <!-- Spreadsheet icon -->
            <svg v-else-if="getFileIcon(file.filename) === 'spreadsheet'" class="w-4 h-4 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 10h18M3 14h18m-9-4v8m-7 0h14a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" />
            </svg>
            <!-- Archive icon -->
            <svg v-else-if="getFileIcon(file.filename) === 'archive'" class="w-4 h-4 text-yellow-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4" />
            </svg>
            <!-- Audio icon -->
            <svg v-else-if="getFileIcon(file.filename) === 'audio'" class="w-4 h-4 text-purple-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3" />
            </svg>
            <!-- Video icon -->
            <svg v-else-if="getFileIcon(file.filename) === 'video'" class="w-4 h-4 text-pink-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" />
            </svg>
            <!-- Default file icon -->
            <svg v-else class="w-4 h-4 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
            </svg>
          </div>

          <!-- File info -->
          <div class="flex-1 min-w-0">
            <p class="text-sm font-medium text-gray-900 truncate">{{ file.filename }}</p>
            <div class="flex items-center gap-2 mt-0.5 text-xs text-gray-500">
              <span>{{ formatFileSize(file.size) }}</span>
              <span class="text-gray-300">|</span>
              <span>{{ formatDate(file.uploaded_at) }}</span>
            </div>
            <p v-if="file.uploader_name" class="text-xs text-gray-400 mt-0.5">
              by {{ file.uploader_name }}
            </p>
          </div>

          <!-- Download icon -->
          <svg class="w-4 h-4 text-gray-400 flex-shrink-0 mt-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
          </svg>
        </div>
      </li>
    </ul>
  </div>
</template>
