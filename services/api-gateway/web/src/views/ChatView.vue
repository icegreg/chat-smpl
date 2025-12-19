<script setup lang="ts">
import { onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useChatStore } from '@/stores/chat'
import LeftNavPanel from '@/components/layout/LeftNavPanel.vue'
import ChatSidebar from '@/components/ChatSidebar.vue'
import ChatRoom from '@/components/ChatRoom.vue'
import ChatEmpty from '@/components/ChatEmpty.vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const chatStore = useChatStore()

onMounted(async () => {
  // Wait for auth to be initialized (user fetched from API)
  // This is needed because App.vue's authStore.init() runs async
  if (!authStore.user) {
    // Wait up to 3 seconds for auth to initialize
    let attempts = 0
    while (!authStore.user && attempts < 30) {
      await new Promise(resolve => setTimeout(resolve, 100))
      attempts++
    }
  }

  // If still no user after waiting, the global auth failure handler
  // in App.vue will redirect to login when API calls fail
  if (!authStore.user) {
    console.log('[ChatView] No user after waiting, will be redirected by auth failure handler')
    return
  }

  await chatStore.initCentrifuge()
  await chatStore.fetchChats()

  // Select chat from route if provided
  if (route.params.id) {
    await chatStore.selectChat(route.params.id as string)
  }
})

onUnmounted(() => {
  chatStore.cleanup()
})

// Watch for route changes
watch(
  () => route.params.id,
  async (newId) => {
    if (newId) {
      await chatStore.selectChat(newId as string)
    } else {
      chatStore.currentChat = null
    }
  }
)

async function handleLogout() {
  await authStore.logout()
  router.push('/login')
}

function handleChatSelect(chatId: string) {
  router.push(`/chats/${chatId}`)
}
</script>

<template>
  <div class="h-screen flex">
    <!-- Left Navigation Panel -->
    <LeftNavPanel />

    <!-- Main App Area -->
    <div class="flex-1 flex flex-col overflow-hidden">
      <!-- Header -->
      <header class="bg-white shadow-sm border-b px-4 py-3 flex items-center justify-between">
        <h1 class="text-xl font-semibold text-gray-800">Chat App</h1>
        <div class="flex items-center gap-4">
          <span class="text-sm text-gray-600">{{ authStore.user?.display_name || authStore.user?.username }}</span>
          <button
            @click="handleLogout"
            class="text-sm text-gray-500 hover:text-gray-700"
          >
            Logout
          </button>
        </div>
      </header>

      <!-- Main content -->
      <div class="flex-1 flex overflow-hidden">
        <!-- Sidebar -->
        <ChatSidebar
          :chats="chatStore.sortedChats"
          :current-chat-id="chatStore.currentChat?.id"
          :loading="chatStore.loading"
          @select="handleChatSelect"
        />

        <!-- Chat area -->
        <main class="flex-1 flex flex-col bg-gray-50">
          <ChatRoom
            v-if="chatStore.currentChat"
            :chat="chatStore.currentChat"
            :messages="chatStore.messages"
            :participants="chatStore.participants"
            :current-user="authStore.user!"
            :is-guest="authStore.isGuest"
          />
          <ChatEmpty v-else />
        </main>
      </div>
    </div>
  </div>
</template>
