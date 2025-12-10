<script setup lang="ts">
import { useNetworkStore } from '@/stores/network'
import { computed } from 'vue'

const networkStore = useNetworkStore()

const showBar = computed(() => networkStore.statusType !== 'none')

const barClasses = computed(() => {
  const base = 'fixed top-0 left-0 right-0 z-50 px-4 py-2 text-center text-sm font-medium transition-all duration-300'

  switch (networkStore.statusType) {
    case 'error':
      return `${base} bg-red-500 text-white`
    case 'warning':
      return `${base} bg-yellow-500 text-black`
    case 'info':
      return `${base} bg-blue-500 text-white`
    default:
      return base
  }
})

const iconPath = computed(() => {
  switch (networkStore.statusType) {
    case 'error':
      // Wi-Fi off icon
      return 'M1 1l22 22M9.2 12.5C9.72 12.19 10.34 12 11 12s1.28.19 1.8.5M5.41 9.37c.27-.17.54-.32.83-.47M8.56 6.33c.79-.23 1.6-.35 2.44-.35.85 0 1.66.12 2.44.35'
    case 'warning':
      // Slow connection icon
      return 'M8.5 16.5l3-5 3 5M8 9h8M12 3a9 9 0 0 1 9 9v3H3v-3a9 9 0 0 1 9-9z'
    case 'info':
      // Sync icon
      return 'M21 12a9 9 0 0 1-9 9m0 0a9 9 0 0 1-9-9m9 9V8m0 13l4-4m-4 4l-4-4M3 12a9 9 0 0 1 9-9m0 0a9 9 0 0 1 9 9m-9-9v13m0-13L8 8m4-4l4 4'
    default:
      return ''
  }
})
</script>

<template>
  <Transition name="slide">
    <div v-if="showBar" :class="barClasses" data-testid="network-status-bar">
      <div class="flex items-center justify-center gap-2">
        <!-- Icon -->
        <svg
          v-if="iconPath"
          class="w-4 h-4"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          viewBox="0 0 24 24"
        >
          <path :d="iconPath" />
        </svg>

        <!-- Spinner for syncing -->
        <svg
          v-if="networkStore.statusType === 'info'"
          class="w-4 h-4 animate-spin"
          fill="none"
          viewBox="0 0 24 24"
        >
          <circle
            class="opacity-25"
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            stroke-width="4"
          />
          <path
            class="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
          />
        </svg>

        <span data-testid="network-status-message">{{ networkStore.statusMessage }}</span>

        <!-- Pending count badge -->
        <span
          v-if="networkStore.pendingCount > 0"
          class="inline-flex items-center justify-center w-5 h-5 text-xs font-bold rounded-full bg-white bg-opacity-30"
          data-testid="pending-count"
        >
          {{ networkStore.pendingCount }}
        </span>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.slide-enter-active,
.slide-leave-active {
  transition: transform 0.3s ease, opacity 0.3s ease;
}

.slide-enter-from,
.slide-leave-to {
  transform: translateY(-100%);
  opacity: 0;
}
</style>
