<template>
  <div class="conference-view">
    <!-- Header -->
    <div class="conference-header">
      <div class="conference-info">
        <h2 class="conference-name">{{ conference?.name || 'Conference' }}</h2>
        <span class="participant-count">{{ participants.length }} participants</span>
      </div>
      <div class="header-actions">
        <!-- Chat toggle button -->
        <button
          v-if="conference?.chat_id"
          class="header-btn"
          :class="{ active: showChat }"
          @click="toggleChat"
          title="Toggle chat"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path>
          </svg>
          <span v-if="unreadCount > 0" class="badge">{{ unreadCount }}</span>
        </button>
        <button class="header-btn" @click="showParticipantsList = !showParticipantsList" title="Participants list">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path>
            <circle cx="9" cy="7" r="4"></circle>
            <path d="M23 21v-2a4 4 0 0 0-3-3.87"></path>
            <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
          </svg>
          <span class="badge">{{ participants.length }}</span>
        </button>
        <button class="close-btn" @click="$emit('leave')" title="Leave conference">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"></line>
            <line x1="6" y1="6" x2="18" y2="18"></line>
          </svg>
        </button>
      </div>
    </div>

    <div class="conference-content">
      <!-- Participants grid -->
      <div class="participants-grid" :class="gridClass">
        <ParticipantTile
          v-for="participant in participants"
          :key="participant.user_id"
          :participant="participant"
          :is-current-user="participant.user_id === currentUserId"
          :is-host="participant.user_id === conference?.created_by"
          :can-manage="canManage"
          @mute="$emit('mute-participant', participant.user_id, $event)"
          @kick="$emit('kick-participant', participant.user_id)"
        />
      </div>

      <!-- Participants sidebar -->
      <Transition name="slide">
        <div v-if="showParticipantsList" class="participants-sidebar">
          <div class="sidebar-header">
            <h3>Participants ({{ participants.length }})</h3>
            <button class="sidebar-close" @click="showParticipantsList = false">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18"></line>
                <line x1="6" y1="6" x2="18" y2="18"></line>
              </svg>
            </button>
          </div>
          <div class="participants-list">
            <div
              v-for="participant in sortedParticipantsByRole"
              :key="participant.user_id"
              class="participant-item"
              :class="{ speaking: participant.is_speaking, muted: participant.is_muted }"
            >
              <div class="participant-avatar-small">
                <img v-if="participant.avatar_url" :src="participant.avatar_url" :alt="participant.display_name" />
                <span v-else class="avatar-initial">{{ (participant.display_name || participant.username || '?').charAt(0).toUpperCase() }}</span>
              </div>
              <div class="participant-details">
                <span class="participant-name-small">
                  {{ participant.display_name || participant.username || 'Unknown' }}
                  <span v-if="participant.user_id === currentUserId" class="you-tag">(You)</span>
                </span>
                <span class="participant-role-small" :class="getRoleClass(participant)">
                  {{ getRoleLabel(participant) }}
                </span>
              </div>
              <div class="participant-status">
                <svg v-if="participant.is_muted" class="status-muted" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" title="Muted">
                  <line x1="1" y1="1" x2="23" y2="23"></line>
                  <path d="M9 9v3a3 3 0 0 0 5.12 2.12M15 9.34V4a3 3 0 0 0-5.94-.6"></path>
                </svg>
                <div v-if="participant.is_speaking" class="speaking-dot"></div>
              </div>
            </div>
          </div>
        </div>
      </Transition>

      <!-- Chat sidebar -->
      <Transition name="slide">
        <div v-if="showChat && conference?.chat_id" class="chat-sidebar">
          <div class="sidebar-header">
            <h3>Chat</h3>
            <button class="sidebar-close" @click="showChat = false">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18"></line>
                <line x1="6" y1="6" x2="18" y2="18"></line>
              </svg>
            </button>
          </div>
          <div class="chat-messages" ref="messagesContainerRef">
            <div v-if="messagesLoading" class="loading-messages">Loading...</div>
            <div v-else-if="chatMessages.length === 0" class="no-messages">No messages yet</div>
            <div
              v-else
              v-for="message in chatMessages"
              :key="message.id"
              class="chat-message"
              :class="{ 'own-message': message.sender_id === currentUserId, 'system-message': message.is_system || message.type === 'system' }"
            >
              <div v-if="!message.is_system && message.type !== 'system'" class="message-sender">
                {{ message.sender_display_name || message.sender_username || 'Unknown' }}
              </div>
              <div class="message-content">{{ message.content }}</div>
              <div class="message-time">{{ formatTime(message.created_at) }}</div>
            </div>
          </div>
          <div class="chat-input">
            <input
              v-model="newMessage"
              type="text"
              placeholder="Type a message..."
              @keyup.enter="sendMessage"
            />
            <button @click="sendMessage" :disabled="!newMessage.trim()">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="22" y1="2" x2="11" y2="13"></line>
                <polygon points="22 2 15 22 11 13 2 9 22 2"></polygon>
              </svg>
            </button>
          </div>
        </div>
      </Transition>
    </div>

    <!-- Controls -->
    <div class="conference-controls">
      <CallControls
        :is-muted="isMuted"
        :show-duration="true"
        :duration="callDuration"
        @toggle-mute="$emit('toggle-mute')"
        @hangup="$emit('leave')"
      />

      <!-- Additional conference controls -->
      <div v-if="canManage" class="host-controls">
        <button class="host-btn" @click="$emit('end-conference')" title="End conference for all">
          End for all
        </button>
      </div>
    </div>

    <!-- Audio element is now in App.vue (global, must exist before calls) -->
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted, watch, nextTick } from 'vue'
import type { Conference, VoiceParticipant, Message } from '@/types'
import { useChatStore } from '@/stores/chat'
import CallControls from './CallControls.vue'
import ParticipantTile from './ParticipantTile.vue'

const props = defineProps<{
  conference: Conference | null
  participants: VoiceParticipant[]
  currentUserId?: string
  isMuted: boolean
  callStartTime?: number
}>()

defineEmits<{
  (e: 'leave'): void
  (e: 'toggle-mute'): void
  (e: 'mute-participant', userId: string, mute: boolean): void
  (e: 'kick-participant', userId: string): void
  (e: 'end-conference'): void
}>()

const chatStore = useChatStore()

const callDuration = ref(0)
const showParticipantsList = ref(false)
const showChat = ref(false)
const newMessage = ref('')
const messagesLoading = ref(false)
const messagesContainerRef = ref<HTMLElement | null>(null)
const unreadCount = ref(0)
let durationInterval: number | null = null

// Get messages for the conference chat
const chatMessages = computed<Message[]>(() => {
  if (!props.conference?.chat_id) return []
  return chatStore.messages
})

// Toggle chat panel
async function toggleChat() {
  showChat.value = !showChat.value
  if (showChat.value && props.conference?.chat_id) {
    // Load messages when opening chat
    messagesLoading.value = true
    try {
      // Select the chat by ID
      const existingChat = chatStore.chats.find(c => c.id === props.conference?.chat_id)
      if (existingChat) {
        await chatStore.selectChat(existingChat.id)
      } else {
        // Load fresh
        await chatStore.fetchChats()
        const chat = chatStore.chats.find(c => c.id === props.conference?.chat_id)
        if (chat) {
          await chatStore.selectChat(chat.id)
        }
      }
      unreadCount.value = 0
      // Scroll to bottom after messages load
      await nextTick()
      scrollToBottom()
    } catch (err) {
      console.error('Failed to load chat messages:', err)
    } finally {
      messagesLoading.value = false
    }
  }
}

// Send a message
async function sendMessage() {
  if (!newMessage.value.trim() || !props.conference?.chat_id) return

  try {
    await chatStore.sendMessage({ content: newMessage.value.trim() })
    newMessage.value = ''
    await nextTick()
    scrollToBottom()
  } catch (err) {
    console.error('Failed to send message:', err)
  }
}

// Scroll to bottom of messages
function scrollToBottom() {
  if (messagesContainerRef.value) {
    messagesContainerRef.value.scrollTop = messagesContainerRef.value.scrollHeight
  }
}

// Format time for display
function formatTime(timestamp: string): string {
  const date = new Date(timestamp)
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

// Watch for new messages and scroll/update unread
watch(() => chatStore.messages.length, () => {
  if (showChat.value) {
    nextTick(() => scrollToBottom())
  } else if (props.conference?.chat_id && chatStore.currentChat?.id === props.conference.chat_id) {
    unreadCount.value++
  }
})

const canManage = computed(() => {
  return props.currentUserId === props.conference?.created_by
})

const gridClass = computed(() => {
  const count = props.participants.length
  if (count <= 1) return 'grid-1'
  if (count <= 2) return 'grid-2'
  if (count <= 4) return 'grid-4'
  if (count <= 6) return 'grid-6'
  return 'grid-many'
})

// Role priority for sorting
const rolePriority: Record<string, number> = {
  originator: 1,
  moderator: 2,
  speaker: 3,
  assistant: 4,
  participant: 5,
}

const sortedParticipantsByRole = computed(() => {
  return [...props.participants].sort((a, b) => {
    // Host (creator) always first
    if (a.user_id === props.conference?.created_by) return -1
    if (b.user_id === props.conference?.created_by) return 1
    // Then by role
    const aPriority = rolePriority[a.role || 'participant'] || 5
    const bPriority = rolePriority[b.role || 'participant'] || 5
    if (aPriority !== bPriority) return aPriority - bPriority
    // Then by name
    const aName = a.display_name || a.username || ''
    const bName = b.display_name || b.username || ''
    return aName.localeCompare(bName)
  })
})

function getRoleLabel(participant: VoiceParticipant): string {
  if (participant.user_id === props.conference?.created_by) return 'Organizer'
  switch (participant.role) {
    case 'originator': return 'Organizer'
    case 'moderator': return 'Moderator'
    case 'speaker': return 'Speaker'
    case 'assistant': return 'Assistant'
    case 'participant': return 'Participant'
    default: return 'Participant'
  }
}

function getRoleClass(participant: VoiceParticipant): string {
  if (participant.user_id === props.conference?.created_by) return 'role-originator'
  return participant.role ? `role-${participant.role}` : 'role-participant'
}

onMounted(() => {
  // Update call duration every second
  durationInterval = window.setInterval(() => {
    if (props.callStartTime) {
      callDuration.value = Math.floor((Date.now() - props.callStartTime) / 1000)
    }
  }, 1000)
})

onUnmounted(() => {
  if (durationInterval) {
    clearInterval(durationInterval)
  }
})
</script>

<style scoped>
.conference-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #1a1a2e;
  color: #fff;
}

.conference-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  background: rgba(0, 0, 0, 0.3);
}

.conference-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.conference-name {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}

.participant-count {
  font-size: 14px;
  color: rgba(255, 255, 255, 0.6);
}

.close-btn {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  border: none;
  background: rgba(255, 255, 255, 0.1);
  color: #fff;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.2s;
}

.close-btn:hover {
  background: rgba(255, 255, 255, 0.2);
}

.close-btn svg {
  width: 20px;
  height: 20px;
}

.grid-1 {
  grid-template-columns: 1fr;
  max-width: 600px;
  margin: 0 auto;
}

.grid-2 {
  grid-template-columns: repeat(2, 1fr);
}

.grid-4 {
  grid-template-columns: repeat(2, 1fr);
  grid-template-rows: repeat(2, 1fr);
}

.grid-6 {
  grid-template-columns: repeat(3, 1fr);
  grid-template-rows: repeat(2, 1fr);
}

.grid-many {
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
}

.conference-controls {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  padding: 24px;
  background: rgba(0, 0, 0, 0.3);
}

.host-controls {
  display: flex;
  gap: 12px;
}

.host-btn {
  padding: 8px 16px;
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.3);
  background: transparent;
  color: #fff;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.host-btn:hover {
  background: rgba(255, 255, 255, 0.1);
  border-color: rgba(255, 255, 255, 0.5);
}

/* Header actions */
.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-btn {
  position: relative;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  border: none;
  background: rgba(255, 255, 255, 0.1);
  color: #fff;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.2s;
}

.header-btn:hover {
  background: rgba(255, 255, 255, 0.2);
}

.header-btn svg {
  width: 20px;
  height: 20px;
}

.header-btn .badge {
  position: absolute;
  top: -4px;
  right: -4px;
  min-width: 18px;
  height: 18px;
  padding: 0 4px;
  border-radius: 9px;
  background: #3b82f6;
  color: #fff;
  font-size: 11px;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* Conference content wrapper */
.conference-content {
  flex: 1;
  display: flex;
  overflow: hidden;
  position: relative;
}

.participants-grid {
  flex: 1;
  display: grid;
  gap: 8px;
  padding: 16px;
  overflow-y: auto;
}

/* Participants sidebar */
.participants-sidebar {
  width: 300px;
  background: #252538;
  border-left: 1px solid rgba(255, 255, 255, 0.1);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.sidebar-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.sidebar-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.sidebar-close {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  border: none;
  background: rgba(255, 255, 255, 0.1);
  color: #fff;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}

.sidebar-close svg {
  width: 14px;
  height: 14px;
}

.sidebar-close:hover {
  background: rgba(255, 255, 255, 0.2);
}

.participants-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.participant-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 8px;
  transition: background 0.2s;
}

.participant-item:hover {
  background: rgba(255, 255, 255, 0.05);
}

.participant-item.speaking {
  background: rgba(34, 197, 94, 0.15);
}

.participant-avatar-small {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: #4a4a6a;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  flex-shrink: 0;
}

.participant-avatar-small img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar-initial {
  font-size: 14px;
  font-weight: 600;
  color: #fff;
}

.participant-details {
  flex: 1;
  min-width: 0;
}

.participant-name-small {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: #fff;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.you-tag {
  font-size: 11px;
  color: #60a5fa;
  margin-left: 4px;
}

.participant-role-small {
  display: inline-block;
  margin-top: 2px;
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 8px;
}

.role-originator {
  background: rgba(251, 191, 36, 0.25);
  color: #fbbf24;
}

.role-moderator {
  background: rgba(168, 85, 247, 0.25);
  color: #c084fc;
}

.role-speaker {
  background: rgba(34, 197, 94, 0.25);
  color: #4ade80;
}

.role-assistant {
  background: rgba(59, 130, 246, 0.25);
  color: #60a5fa;
}

.role-participant {
  background: rgba(148, 163, 184, 0.2);
  color: #94a3b8;
}

.participant-status {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.status-muted {
  width: 16px;
  height: 16px;
  color: #ef4444;
}

.speaking-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #22c55e;
  animation: pulse-dot 1s infinite;
}

@keyframes pulse-dot {
  0%, 100% {
    opacity: 1;
    transform: scale(1);
  }
  50% {
    opacity: 0.6;
    transform: scale(1.2);
  }
}

/* Sidebar slide transition */
.slide-enter-active,
.slide-leave-active {
  transition: transform 0.3s ease, opacity 0.3s ease;
}

.slide-enter-from,
.slide-leave-to {
  transform: translateX(100%);
  opacity: 0;
}

/* Header button active state */
.header-btn.active {
  background: rgba(59, 130, 246, 0.3);
  color: #60a5fa;
}

/* Chat sidebar */
.chat-sidebar {
  width: 350px;
  background: #252538;
  border-left: 1px solid rgba(255, 255, 255, 0.1);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.loading-messages,
.no-messages {
  text-align: center;
  color: rgba(255, 255, 255, 0.5);
  padding: 20px;
  font-size: 14px;
}

.chat-message {
  max-width: 85%;
  padding: 8px 12px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.1);
  align-self: flex-start;
}

.chat-message.own-message {
  background: rgba(59, 130, 246, 0.3);
  align-self: flex-end;
}

.chat-message.system-message {
  background: transparent;
  border: 1px dashed rgba(255, 255, 255, 0.2);
  align-self: center;
  max-width: 90%;
  text-align: center;
  font-style: italic;
  color: rgba(255, 255, 255, 0.6);
  font-size: 12px;
}

.chat-message.system-message .message-sender {
  display: none;
}

.message-sender {
  font-size: 11px;
  font-weight: 600;
  color: #60a5fa;
  margin-bottom: 2px;
}

.own-message .message-sender {
  display: none;
}

.message-content {
  font-size: 14px;
  color: #fff;
  word-break: break-word;
}

.message-time {
  font-size: 10px;
  color: rgba(255, 255, 255, 0.4);
  margin-top: 4px;
  text-align: right;
}

.chat-input {
  display: flex;
  gap: 8px;
  padding: 12px;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(0, 0, 0, 0.2);
}

.chat-input input {
  flex: 1;
  padding: 10px 14px;
  border-radius: 20px;
  border: 1px solid rgba(255, 255, 255, 0.2);
  background: rgba(255, 255, 255, 0.05);
  color: #fff;
  font-size: 14px;
  outline: none;
  transition: border-color 0.2s;
}

.chat-input input::placeholder {
  color: rgba(255, 255, 255, 0.4);
}

.chat-input input:focus {
  border-color: rgba(59, 130, 246, 0.5);
}

.chat-input button {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  border: none;
  background: #3b82f6;
  color: #fff;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.2s;
}

.chat-input button:hover:not(:disabled) {
  background: #2563eb;
}

.chat-input button:disabled {
  background: rgba(255, 255, 255, 0.1);
  cursor: not-allowed;
}

.chat-input button svg {
  width: 18px;
  height: 18px;
}
</style>
