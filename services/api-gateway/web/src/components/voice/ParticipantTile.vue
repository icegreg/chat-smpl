<template>
  <div
    class="participant-tile"
    :class="{
      speaking: participant.is_speaking,
      muted: participant.is_muted,
      'current-user': isCurrentUser
    }"
  >
    <!-- Avatar -->
    <div class="participant-avatar">
      <img v-if="participant.avatar_url" :src="participant.avatar_url" :alt="displayName" />
      <span v-else class="avatar-placeholder">{{ initial }}</span>

      <!-- Speaking indicator -->
      <div v-if="participant.is_speaking" class="speaking-indicator">
        <span class="wave"></span>
        <span class="wave"></span>
        <span class="wave"></span>
      </div>
    </div>

    <!-- Name and status -->
    <div class="participant-info">
      <span class="participant-name">
        {{ displayName }}
        <span v-if="isCurrentUser" class="you-badge">(You)</span>
      </span>
      <span class="participant-role" :class="roleClass">{{ roleLabel }}</span>
    </div>

    <!-- Status icons -->
    <div class="status-icons">
      <div v-if="participant.is_muted" class="status-icon muted" title="Muted">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="1" y1="1" x2="23" y2="23"></line>
          <path d="M9 9v3a3 3 0 0 0 5.12 2.12M15 9.34V4a3 3 0 0 0-5.94-.6"></path>
          <path d="M17 16.95A7 7 0 0 1 5 12v-2m14 0v2a7 7 0 0 1-.11 1.23"></path>
        </svg>
      </div>
    </div>

    <!-- Context menu for host -->
    <div v-if="canManage && !isCurrentUser" class="participant-actions">
      <button class="action-btn" @click.stop="showMenu = !showMenu">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="1"></circle>
          <circle cx="12" cy="5" r="1"></circle>
          <circle cx="12" cy="19" r="1"></circle>
        </svg>
      </button>

      <div v-if="showMenu" class="action-menu" @click.stop>
        <button @click="toggleMute">
          {{ participant.is_muted ? 'Unmute' : 'Mute' }}
        </button>
        <button class="danger" @click="kickParticipant">
          Remove
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { VoiceParticipant } from '@/types'

const props = defineProps<{
  participant: VoiceParticipant
  isCurrentUser?: boolean
  isHost?: boolean
  canManage?: boolean
}>()

const emit = defineEmits<{
  (e: 'mute', mute: boolean): void
  (e: 'kick'): void
}>()

const showMenu = ref(false)

const displayName = computed(() => {
  return props.participant.display_name || props.participant.username || 'Unknown'
})

const initial = computed(() => {
  return displayName.value.charAt(0).toUpperCase()
})

const roleLabel = computed(() => {
  const role = props.participant.role
  if (props.isHost) return 'Organizer'
  switch (role) {
    case 'originator': return 'Organizer'
    case 'moderator': return 'Moderator'
    case 'speaker': return 'Speaker'
    case 'assistant': return 'Assistant'
    case 'participant': return 'Participant'
    default: return 'Participant'
  }
})

const roleClass = computed(() => {
  const role = props.participant.role
  if (props.isHost) return 'role-originator'
  return role ? `role-${role}` : 'role-participant'
})

function toggleMute() {
  emit('mute', !props.participant.is_muted)
  showMenu.value = false
}

function kickParticipant() {
  emit('kick')
  showMenu.value = false
}
</script>

<style scoped>
.participant-tile {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 24px;
  background: #2d2d44;
  border-radius: 12px;
  min-height: 180px;
  transition: all 0.2s ease;
}

.participant-tile.speaking {
  box-shadow: 0 0 0 3px #22c55e;
}

.participant-tile.current-user {
  background: #3d3d5c;
}

.participant-avatar {
  position: relative;
  width: 80px;
  height: 80px;
  border-radius: 50%;
  overflow: hidden;
  background: #4a4a6a;
  display: flex;
  align-items: center;
  justify-content: center;
}

.participant-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar-placeholder {
  font-size: 32px;
  font-weight: 600;
  color: #fff;
}

.speaking-indicator {
  position: absolute;
  bottom: -4px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  gap: 2px;
  padding: 4px 8px;
  background: #22c55e;
  border-radius: 10px;
}

.wave {
  width: 3px;
  height: 12px;
  background: #fff;
  border-radius: 2px;
  animation: wave 0.5s infinite ease-in-out;
}

.wave:nth-child(2) {
  animation-delay: 0.1s;
}

.wave:nth-child(3) {
  animation-delay: 0.2s;
}

@keyframes wave {
  0%, 100% {
    height: 4px;
  }
  50% {
    height: 12px;
  }
}

.participant-info {
  text-align: center;
}

.participant-name {
  font-size: 14px;
  font-weight: 500;
  color: #fff;
}

.you-badge {
  display: inline-block;
  margin-left: 4px;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 600;
  background: rgba(59, 130, 246, 0.3);
  color: #60a5fa;
}

.participant-role {
  display: block;
  margin-top: 4px;
  font-size: 11px;
  font-weight: 500;
  padding: 2px 8px;
  border-radius: 10px;
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

.status-icons {
  position: absolute;
  top: 8px;
  right: 8px;
  display: flex;
  gap: 4px;
}

.status-icon {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.status-icon.muted {
  background: rgba(239, 68, 68, 0.3);
  color: #ef4444;
}

.status-icon svg {
  width: 16px;
  height: 16px;
}

.participant-actions {
  position: absolute;
  top: 8px;
  left: 8px;
}

.action-btn {
  width: 32px;
  height: 32px;
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

.action-btn:hover {
  background: rgba(255, 255, 255, 0.2);
}

.action-btn svg {
  width: 16px;
  height: 16px;
}

.action-menu {
  position: absolute;
  top: 100%;
  left: 0;
  margin-top: 4px;
  background: #3d3d5c;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  z-index: 10;
}

.action-menu button {
  display: block;
  width: 100%;
  padding: 10px 16px;
  border: none;
  background: transparent;
  color: #fff;
  font-size: 14px;
  text-align: left;
  cursor: pointer;
  transition: background 0.2s;
}

.action-menu button:hover {
  background: rgba(255, 255, 255, 0.1);
}

.action-menu button.danger {
  color: #ef4444;
}

.action-menu button.danger:hover {
  background: rgba(239, 68, 68, 0.2);
}
</style>
