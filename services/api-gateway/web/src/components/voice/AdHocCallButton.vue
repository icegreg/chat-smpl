<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useVoiceStore } from '@/stores/voice'
import { useChatStore } from '@/stores/chat'
import { useAuthStore } from '@/stores/auth'
import type { Participant } from '@/types'

const props = defineProps<{
  chatId: string
  chatName?: string
}>()

const voiceStore = useVoiceStore()
const chatStore = useChatStore()
const authStore = useAuthStore()

const showDropdown = ref(false)
const showParticipantSelector = ref(false)
const selectedParticipants = ref<Set<string>>(new Set())
const dropdownRef = ref<HTMLElement | null>(null)

// Get participants of the chat
const participants = computed(() => {
  return chatStore.participants.filter(
    (p: Participant) => p.user_id !== authStore.user?.id
  )
})

const loading = computed(() => voiceStore.loading)
const isInCall = computed(() => voiceStore.isInCall)

// Active conference for this chat
const activeConference = computed(() => voiceStore.getActiveConference(props.chatId))
const hasActiveConference = computed(() => voiceStore.hasActiveConference(props.chatId))

// Click outside handler
function handleClickOutside(event: MouseEvent) {
  if (dropdownRef.value && !dropdownRef.value.contains(event.target as Node)) {
    showDropdown.value = false
    showParticipantSelector.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})

function toggleDropdown(event: Event) {
  event.stopPropagation()
  showDropdown.value = !showDropdown.value
  showParticipantSelector.value = false
}

// Start call with all participants
async function startCallAll() {
  console.log('[AdHocCallButton] startCallAll called, chatId:', props.chatId, 'chatName:', props.chatName)
  showDropdown.value = false
  try {
    // Use startChatCall which creates conference + gets credentials + joins in one call
    console.log('[AdHocCallButton] Calling voiceStore.startChatCall...')
    const result = await voiceStore.startChatCall(props.chatId, props.chatName)
    console.log('[AdHocCallButton] startChatCall result:', result)
  } catch (err) {
    console.error('[AdHocCallButton] startChatCall error:', err)
  }
}

// Open participant selector
function openParticipantSelector() {
  showParticipantSelector.value = true
  selectedParticipants.value = new Set()
}

// Toggle participant selection
function toggleParticipant(userId: string) {
  if (selectedParticipants.value.has(userId)) {
    selectedParticipants.value.delete(userId)
  } else {
    selectedParticipants.value.add(userId)
  }
  // Force reactivity
  selectedParticipants.value = new Set(selectedParticipants.value)
}

// Start call with selected participants
async function startCallSelected() {
  if (selectedParticipants.value.size === 0) return
  showDropdown.value = false
  showParticipantSelector.value = false
  const conference = await voiceStore.createAdHocFromChat(
    props.chatId,
    Array.from(selectedParticipants.value)
  )
  if (conference) {
    await voiceStore.joinConference(conference.id)
  }
}

// Join existing active conference
async function joinExistingCall() {
  if (!activeConference.value) return
  try {
    await voiceStore.joinConference(activeConference.value.id)
  } catch (err) {
    console.error('[AdHocCallButton] joinExistingCall error:', err)
  }
}

// End current call
async function endCall() {
  if (voiceStore.currentConference) {
    await voiceStore.leaveConference()
  } else {
    await voiceStore.hangupCall()
  }
}

// Get avatar initials
function getInitials(name?: string): string {
  if (!name) return '?'
  return name.split(' ').map(n => n[0]).join('').slice(0, 2).toUpperCase()
}
</script>

<template>
  <div class="adhoc-call-button" ref="dropdownRef">
    <!-- Join existing call button (when there's an active conference and we're not in it) -->
    <button
      v-if="hasActiveConference && !isInCall"
      class="main-btn join-btn"
      :class="{ loading: loading }"
      :disabled="loading"
      @click="joinExistingCall"
      :title="`Присоединиться к звонку (${activeConference?.participant_count || 0} участников)`"
    >
      <span v-if="loading" class="spinner"></span>
      <template v-else>
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" />
        </svg>
        <span class="join-text">Join ({{ activeConference?.participant_count || 0 }})</span>
      </template>
    </button>

    <!-- Main button (when no active conference or we're in call) -->
    <button
      v-else
      class="main-btn"
      :class="{ active: isInCall, loading: loading }"
      :disabled="loading"
      @click="isInCall ? endCall() : toggleDropdown($event)"
    >
      <span v-if="loading" class="spinner"></span>
      <svg v-else-if="isInCall" class="icon hangup" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M23 16.67v3.33a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 5.11 2h3.33a2 2 0 0 1 2 1.72c.127.96.361 1.903.7 2.81a2 2 0 0 1-.45 2.11L9.09 10.24a16 16 0 0 0 6 6l1.6-1.6a2 2 0 0 1 2.11-.45c.907.339 1.85.573 2.81.7a2 2 0 0 1 1.72 2.03v.75z" transform="rotate(135 12 12)"></path>
      </svg>
      <template v-else>
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" />
        </svg>
        <svg class="chevron" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
        </svg>
      </template>
    </button>

    <!-- Dropdown -->
    <div v-if="showDropdown && !isInCall" class="dropdown">
      <!-- Default options -->
      <div v-if="!showParticipantSelector" class="dropdown-options">
        <button class="dropdown-item" @click="startCallAll">
          <svg class="item-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
          </svg>
          Call All
        </button>
        <button class="dropdown-item" @click="openParticipantSelector">
          <svg class="item-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
          </svg>
          Select Participants...
        </button>
      </div>

      <!-- Participant selector -->
      <div v-else class="participant-selector">
        <div class="selector-header">
          <button class="back-btn" @click="showParticipantSelector = false">
            <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" />
            </svg>
          </button>
          <span class="selector-title">Select participants</span>
        </div>

        <div class="participant-list">
          <label
            v-for="participant in participants"
            :key="participant.user_id"
            class="participant-item"
            :class="{ selected: selectedParticipants.has(participant.user_id) }"
          >
            <input
              type="checkbox"
              :checked="selectedParticipants.has(participant.user_id)"
              @change="toggleParticipant(participant.user_id)"
            />
            <div class="participant-avatar">
              <img
                v-if="participant.avatar_url"
                :src="participant.avatar_url"
                :alt="participant.display_name || participant.username"
              />
              <span v-else class="initials">
                {{ getInitials(participant.display_name || participant.username) }}
              </span>
            </div>
            <span class="participant-name">
              {{ participant.display_name || participant.username }}
            </span>
          </label>
        </div>

        <button
          class="start-call-btn"
          :disabled="selectedParticipants.size === 0"
          @click="startCallSelected"
        >
          <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" />
          </svg>
          Start Call ({{ selectedParticipants.size }})
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.adhoc-call-button {
  position: relative;
}

.main-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 8px 12px;
  border-radius: 8px;
  border: none;
  background: #22c55e;
  color: #fff;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.main-btn:hover:not(:disabled) {
  background: #16a34a;
}

.main-btn:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}

.main-btn.active {
  background: #ef4444;
}

.main-btn.active:hover:not(:disabled) {
  background: #dc2626;
}

.main-btn.join-btn {
  background: #3b82f6;
  animation: pulse-join 2s infinite;
}

.main-btn.join-btn:hover:not(:disabled) {
  background: #2563eb;
}

.join-text {
  margin-left: 4px;
  font-size: 13px;
}

@keyframes pulse-join {
  0%, 100% {
    box-shadow: 0 0 0 0 rgba(59, 130, 246, 0.5);
  }
  50% {
    box-shadow: 0 0 0 6px rgba(59, 130, 246, 0);
  }
}

.main-btn.loading {
  background: #6b7280;
}

.icon {
  width: 18px;
  height: 18px;
}

.chevron {
  width: 14px;
  height: 14px;
  margin-left: 2px;
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

.dropdown {
  position: absolute;
  top: 100%;
  right: 0;
  margin-top: 4px;
  min-width: 200px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.15);
  overflow: hidden;
  z-index: 100;
}

.dropdown-options {
  padding: 8px;
}

.dropdown-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 10px 12px;
  border: none;
  background: transparent;
  border-radius: 8px;
  font-size: 14px;
  color: #374151;
  cursor: pointer;
  transition: all 0.15s ease;
  text-align: left;
}

.dropdown-item:hover {
  background: #f3f4f6;
}

.item-icon {
  width: 18px;
  height: 18px;
  color: #6b7280;
}

.participant-selector {
  max-height: 300px;
  display: flex;
  flex-direction: column;
}

.selector-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px;
  border-bottom: 1px solid #e5e7eb;
}

.back-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  background: transparent;
  border-radius: 6px;
  cursor: pointer;
  color: #6b7280;
}

.back-btn:hover {
  background: #f3f4f6;
}

.back-btn .icon {
  width: 16px;
  height: 16px;
}

.selector-title {
  font-size: 14px;
  font-weight: 500;
  color: #374151;
}

.participant-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.participant-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.participant-item:hover {
  background: #f3f4f6;
}

.participant-item.selected {
  background: #eef2ff;
}

.participant-item input {
  display: none;
}

.participant-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: #e5e7eb;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
}

.participant-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.participant-avatar .initials {
  font-size: 12px;
  font-weight: 600;
  color: #6b7280;
}

.participant-name {
  flex: 1;
  font-size: 14px;
  color: #374151;
}

.start-call-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin: 8px;
  padding: 10px 16px;
  border: none;
  border-radius: 8px;
  background: #22c55e;
  color: #fff;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.start-call-btn:hover:not(:disabled) {
  background: #16a34a;
}

.start-call-btn:disabled {
  background: #9ca3af;
  cursor: not-allowed;
}

.start-call-btn .icon {
  width: 16px;
  height: 16px;
}
</style>
