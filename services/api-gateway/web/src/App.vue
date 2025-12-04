<script setup lang="ts">
import { onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()

onMounted(async () => {
  await authStore.init()
})
</script>

<template>
  <div class="min-h-screen bg-gray-100">
    <router-view />
  </div>
</template>
