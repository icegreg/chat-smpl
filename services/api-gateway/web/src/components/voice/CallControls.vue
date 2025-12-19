<template>
  <div class="call-controls">
    <!-- Call duration -->
    <div v-if="showDuration && (duration ?? 0) > 0" class="call-duration">
      {{ formattedDuration }}
    </div>

    <!-- Control buttons -->
    <div class="controls-row">
      <!-- Mute button -->
      <button
        class="control-btn"
        :class="{ active: isMuted }"
        @click="$emit('toggle-mute')"
        :title="isMuted ? 'Unmute' : 'Mute'"
      >
        <svg v-if="isMuted" class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="1" y1="1" x2="23" y2="23"></line>
          <path d="M9 9v3a3 3 0 0 0 5.12 2.12M15 9.34V4a3 3 0 0 0-5.94-.6"></path>
          <path d="M17 16.95A7 7 0 0 1 5 12v-2m14 0v2a7 7 0 0 1-.11 1.23"></path>
          <line x1="12" y1="19" x2="12" y2="23"></line>
          <line x1="8" y1="23" x2="16" y2="23"></line>
        </svg>
        <svg v-else class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"></path>
          <path d="M19 10v2a7 7 0 0 1-14 0v-2"></path>
          <line x1="12" y1="19" x2="12" y2="23"></line>
          <line x1="8" y1="23" x2="16" y2="23"></line>
        </svg>
      </button>

      <!-- Hangup button -->
      <button
        class="control-btn hangup"
        @click="$emit('hangup')"
        title="End call"
      >
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M23 16.67v3.33a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 5.11 2h3.33a2 2 0 0 1 2 1.72c.127.96.361 1.903.7 2.81a2 2 0 0 1-.45 2.11L9.09 10.24a16 16 0 0 0 6 6l1.6-1.6a2 2 0 0 1 2.11-.45c.907.339 1.85.573 2.81.7a2 2 0 0 1 1.72 2.03v.75z" transform="rotate(135 12 12)"></path>
        </svg>
      </button>

      <!-- Speaker button (optional) -->
      <button
        v-if="showSpeaker"
        class="control-btn"
        :class="{ active: isSpeakerOn }"
        @click="$emit('toggle-speaker')"
        :title="isSpeakerOn ? 'Speaker off' : 'Speaker on'"
      >
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"></polygon>
          <path v-if="isSpeakerOn" d="M19.07 4.93a10 10 0 0 1 0 14.14M15.54 8.46a5 5 0 0 1 0 7.07"></path>
        </svg>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  isMuted: boolean
  isSpeakerOn?: boolean
  showSpeaker?: boolean
  showDuration?: boolean
  duration?: number // in seconds
}>()

defineEmits<{
  (e: 'toggle-mute'): void
  (e: 'hangup'): void
  (e: 'toggle-speaker'): void
}>()

const formattedDuration = computed(() => {
  const d = props.duration || 0
  const minutes = Math.floor(d / 60)
  const seconds = d % 60
  return `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`
})
</script>

<style scoped>
.call-controls {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 16px;
  background: rgba(0, 0, 0, 0.8);
  border-radius: 16px;
}

.call-duration {
  font-size: 24px;
  font-weight: 500;
  color: #fff;
  font-variant-numeric: tabular-nums;
}

.controls-row {
  display: flex;
  gap: 16px;
  align-items: center;
}

.control-btn {
  width: 56px;
  height: 56px;
  border-radius: 50%;
  border: none;
  background: rgba(255, 255, 255, 0.2);
  color: #fff;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.control-btn:hover {
  background: rgba(255, 255, 255, 0.3);
}

.control-btn.active {
  background: #ff4444;
}

.control-btn.hangup {
  background: #ff4444;
  width: 64px;
  height: 64px;
}

.control-btn.hangup:hover {
  background: #ff2222;
}

.icon {
  width: 24px;
  height: 24px;
}

.hangup .icon {
  width: 28px;
  height: 28px;
}
</style>
