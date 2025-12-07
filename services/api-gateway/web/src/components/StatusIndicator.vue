<script setup lang="ts">
import { computed } from 'vue'
import { usePresenceStore } from '@/stores/presence'
import type { PresenceInfo } from '@/types'

const props = defineProps<{
  userId?: string
  presence?: PresenceInfo | null
  size?: 'sm' | 'md' | 'lg'
  showLabel?: boolean
}>()

const presenceStore = usePresenceStore()

const effectivePresence = computed(() => {
  if (props.presence !== undefined) {
    return props.presence
  }
  if (props.userId) {
    return presenceStore.getUserPresence(props.userId)
  }
  return null
})

const color = computed(() => presenceStore.getStatusColor(effectivePresence.value))
const label = computed(() => presenceStore.getStatusLabel(effectivePresence.value))

const sizeClass = computed(() => {
  switch (props.size) {
    case 'sm':
      return 'w-2 h-2'
    case 'lg':
      return 'w-4 h-4'
    default:
      return 'w-3 h-3'
  }
})
</script>

<template>
  <div class="status-indicator" :class="{ 'with-label': showLabel }">
    <span class="status-dot" :class="sizeClass" :style="{ backgroundColor: color }" :title="label"></span>
    <span v-if="showLabel" class="status-label">{{ label }}</span>
  </div>
</template>

<style scoped>
.status-indicator {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
}

.status-dot {
  display: inline-block;
  border-radius: 50%;
  flex-shrink: 0;
}

.status-label {
  font-size: 0.75rem;
  color: #6b7280;
}
</style>
