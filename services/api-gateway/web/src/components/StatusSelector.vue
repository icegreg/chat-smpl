<script setup lang="ts">
import { ref, computed } from 'vue'
import { usePresenceStore } from '@/stores/presence'
import type { UserStatus } from '@/types'

const presenceStore = usePresenceStore()
const isOpen = ref(false)

const statuses: { value: UserStatus; label: string }[] = [
  { value: 'available', label: 'Available' },
  { value: 'busy', label: 'Busy' },
  { value: 'away', label: 'Away' },
  { value: 'dnd', label: 'Do Not Disturb' },
]

const currentColor = computed(() => presenceStore.getStatusColor(presenceStore.myPresence))
const currentLabel = computed(() => presenceStore.getStatusLabel(presenceStore.myPresence))

async function selectStatus(status: UserStatus) {
  await presenceStore.setStatus(status)
  isOpen.value = false
}

function toggleDropdown() {
  isOpen.value = !isOpen.value
}

function closeDropdown() {
  isOpen.value = false
}
</script>

<template>
  <div class="status-selector" @mouseleave="closeDropdown">
    <button class="status-button" @click="toggleDropdown" :title="currentLabel">
      <span class="status-dot" :style="{ backgroundColor: currentColor }"></span>
      <span class="status-text">{{ currentLabel }}</span>
      <svg class="chevron" :class="{ open: isOpen }" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
        <path fill-rule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clip-rule="evenodd" />
      </svg>
    </button>

    <div v-if="isOpen" class="status-dropdown">
      <button
        v-for="status in statuses"
        :key="status.value"
        class="status-option"
        :class="{ active: presenceStore.myStatus === status.value }"
        @click="selectStatus(status.value)"
      >
        <span class="status-dot" :style="{ backgroundColor: presenceStore.statusColors[status.value] }"></span>
        <span>{{ status.label }}</span>
      </button>
    </div>
  </div>
</template>

<style scoped>
.status-selector {
  position: relative;
}

.status-button {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.375rem 0.75rem;
  background: transparent;
  border: 1px solid #e5e7eb;
  border-radius: 0.375rem;
  cursor: pointer;
  font-size: 0.875rem;
  color: #374151;
  transition: all 0.15s;
}

.status-button:hover {
  background: #f3f4f6;
  border-color: #d1d5db;
}

.status-dot {
  width: 0.625rem;
  height: 0.625rem;
  border-radius: 50%;
  flex-shrink: 0;
}

.status-text {
  max-width: 100px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chevron {
  width: 1rem;
  height: 1rem;
  color: #9ca3af;
  transition: transform 0.15s;
}

.chevron.open {
  transform: rotate(180deg);
}

.status-dropdown {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  margin-top: 0.25rem;
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 0.375rem;
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
  z-index: 50;
  min-width: 160px;
}

.status-option {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  width: 100%;
  padding: 0.5rem 0.75rem;
  background: transparent;
  border: none;
  cursor: pointer;
  font-size: 0.875rem;
  color: #374151;
  text-align: left;
  transition: background 0.15s;
}

.status-option:hover {
  background: #f3f4f6;
}

.status-option.active {
  background: #eff6ff;
  color: #2563eb;
}

.status-option:first-child {
  border-radius: 0.375rem 0.375rem 0 0;
}

.status-option:last-child {
  border-radius: 0 0 0.375rem 0.375rem;
}
</style>
