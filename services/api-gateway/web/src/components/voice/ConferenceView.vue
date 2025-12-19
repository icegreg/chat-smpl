<template>
  <div class="conference-view">
    <!-- Header -->
    <div class="conference-header">
      <div class="conference-info">
        <h2 class="conference-name">{{ conference?.name || 'Conference' }}</h2>
        <span class="participant-count">{{ participants.length }} participants</span>
      </div>
      <button class="close-btn" @click="$emit('leave')" title="Leave conference">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="18" y1="6" x2="6" y2="18"></line>
          <line x1="6" y1="6" x2="18" y2="18"></line>
        </svg>
      </button>
    </div>

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

    <!-- Audio element for Verto -->
    <audio id="verto-audio" autoplay></audio>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue'
import type { Conference, VoiceParticipant } from '@/types'
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

const callDuration = ref(0)
let durationInterval: number | null = null

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

.participants-grid {
  flex: 1;
  display: grid;
  gap: 8px;
  padding: 16px;
  overflow-y: auto;
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

#verto-audio {
  display: none;
}
</style>
