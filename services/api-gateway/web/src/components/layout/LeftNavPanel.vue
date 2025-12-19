<script setup lang="ts">
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useVoiceStore } from '@/stores/voice'

const router = useRouter()
const route = useRoute()
const voiceStore = useVoiceStore()

const hasUpcomingReminders = computed(() => voiceStore.hasUpcomingReminders)
const upcomingCount = computed(() => voiceStore.upcomingConferences.length)

const currentPath = computed(() => route.path)

function goToChats() {
  router.push('/chats')
}

function goToEvents() {
  router.push('/events')
}

async function startQuickCall() {
  const conference = await voiceStore.createQuickAdHoc()
  if (conference) {
    // Join the created conference
    await voiceStore.joinConference(conference.id)
  }
}
</script>

<template>
  <nav class="left-nav-panel">
    <!-- Chats -->
    <button
      class="nav-btn"
      :class="{ active: currentPath.startsWith('/chats') }"
      @click="goToChats"
      title="Chats"
    >
      <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
      </svg>
    </button>

    <!-- Quick Call -->
    <button
      class="nav-btn call-btn"
      :class="{ loading: voiceStore.loading }"
      :disabled="voiceStore.loading"
      @click="startQuickCall"
      title="Quick Call"
    >
      <span v-if="voiceStore.loading" class="spinner"></span>
      <svg v-else class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" />
      </svg>
    </button>

    <!-- Events -->
    <button
      class="nav-btn"
      :class="{ active: currentPath.startsWith('/events') }"
      @click="goToEvents"
      title="Events"
    >
      <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
      </svg>
      <!-- Badge for upcoming events or reminders -->
      <span
        v-if="hasUpcomingReminders || upcomingCount > 0"
        class="badge"
        :class="{ alert: hasUpcomingReminders }"
      >
        {{ hasUpcomingReminders ? '!' : upcomingCount }}
      </span>
    </button>

    <!-- Spacer -->
    <div class="spacer"></div>

    <!-- Settings (future) -->
    <button
      class="nav-btn"
      title="Settings"
      disabled
    >
      <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
        <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
      </svg>
    </button>
  </nav>
</template>

<style scoped>
.left-nav-panel {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 56px;
  min-width: 56px;
  background: #1e293b;
  padding: 12px 0;
  gap: 8px;
}

.nav-btn {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 12px;
  border: none;
  background: transparent;
  color: #94a3b8;
  cursor: pointer;
  transition: all 0.2s ease;
}

.nav-btn:hover:not(:disabled) {
  background: #334155;
  color: #f1f5f9;
}

.nav-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.nav-btn.active {
  background: #4f46e5;
  color: #fff;
}

.nav-btn.call-btn {
  background: #22c55e;
  color: #fff;
}

.nav-btn.call-btn:hover:not(:disabled) {
  background: #16a34a;
}

.nav-btn.call-btn.loading {
  background: #6b7280;
}

.icon {
  width: 22px;
  height: 22px;
}

.badge {
  position: absolute;
  top: -2px;
  right: -2px;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  font-size: 11px;
  font-weight: 600;
  line-height: 18px;
  text-align: center;
  background: #4f46e5;
  color: #fff;
  border-radius: 9px;
}

.badge.alert {
  background: #ef4444;
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.7;
  }
}

.spinner {
  width: 18px;
  height: 18px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.spacer {
  flex: 1;
}
</style>
