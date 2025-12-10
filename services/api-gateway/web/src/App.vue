<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useNetworkStore } from '@/stores/network'
import NetworkStatusBar from '@/components/NetworkStatusBar.vue'

const authStore = useAuthStore()
const networkStore = useNetworkStore()

onMounted(async () => {
  // Инициализируем отслеживание сети
  networkStore.init()

  await authStore.init()
})

onUnmounted(() => {
  networkStore.cleanup()
})
</script>

<template>
  <div class="min-h-screen bg-gray-100">
    <!-- Network status bar (shows when offline/slow/syncing) -->
    <NetworkStatusBar />

    <router-view />
  </div>
</template>
