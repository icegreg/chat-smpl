<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useVoiceStore } from '@/stores/voice'
import { useAuthStore } from '@/stores/auth'
import type { ScheduledConference, ConferenceParticipant, RSVPStatus, ConferenceRole } from '@/types'
import LeftNavPanel from '@/components/layout/LeftNavPanel.vue'

const voiceStore = useVoiceStore()
const authStore = useAuthStore()

const showUpcomingOnly = ref(true)
const selectedEvent = ref<ScheduledConference | null>(null)
const showCreateModal = ref(false)

onMounted(() => {
  voiceStore.fetchScheduledConferences(showUpcomingOnly.value)
})

// Grouped conferences by date
const groupedConferences = computed(() => {
  const groups: Record<string, ScheduledConference[]> = {}

  for (const conf of voiceStore.scheduledConferences) {
    if (!conf.scheduled_at) continue
    const date = new Date(conf.scheduled_at)
    const key = date.toLocaleDateString([], { weekday: 'long', month: 'long', day: 'numeric' })
    if (!groups[key]) groups[key] = []
    groups[key].push(conf)
  }

  // Sort each group by time
  for (const key in groups) {
    groups[key].sort((a, b) => {
      const aTime = a.scheduled_at ? new Date(a.scheduled_at).getTime() : 0
      const bTime = b.scheduled_at ? new Date(b.scheduled_at).getTime() : 0
      return aTime - bTime
    })
  }

  return groups
})

const groupKeys = computed(() => Object.keys(groupedConferences.value))

// Get my participation info for an event
function getMyParticipation(event: ScheduledConference): ConferenceParticipant | undefined {
  return event.participants?.find((p: ConferenceParticipant) => p.user_id === authStore.user?.id)
}

// Format time
function formatTime(dateString: string): string {
  const date = new Date(dateString)
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

// Get event type badge
function getEventTypeBadge(type: string): { label: string; class: string } {
  switch (type) {
    case 'scheduled':
      return { label: 'Scheduled', class: 'badge-scheduled' }
    case 'recurring':
      return { label: 'Recurring', class: 'badge-recurring' }
    case 'adhoc_chat':
      return { label: 'Ad-hoc', class: 'badge-adhoc' }
    default:
      return { label: type, class: 'badge-default' }
  }
}

// Get role badge
function getRoleBadge(role: ConferenceRole): { label: string; class: string } {
  switch (role) {
    case 'originator':
      return { label: 'Organizer', class: 'role-organizer' }
    case 'moderator':
      return { label: 'Moderator', class: 'role-moderator' }
    case 'speaker':
      return { label: 'Speaker', class: 'role-speaker' }
    case 'assistant':
      return { label: 'Assistant', class: 'role-assistant' }
    default:
      return { label: 'Participant', class: 'role-participant' }
  }
}

// RSVP actions
async function updateRSVP(eventId: string, status: RSVPStatus) {
  await voiceStore.updateRSVP(eventId, status)
}

// Join event
async function joinEvent(eventId: string) {
  await voiceStore.joinConference(eventId)
}

// Cancel event (for organizer)
async function cancelEvent(eventId: string) {
  if (confirm('Are you sure you want to cancel this event?')) {
    await voiceStore.cancelConference(eventId)
    selectedEvent.value = null
  }
}

// Toggle upcoming filter
function toggleFilter() {
  showUpcomingOnly.value = !showUpcomingOnly.value
  voiceStore.fetchScheduledConferences(showUpcomingOnly.value)
}

// Get initials
function getInitials(name?: string): string {
  if (!name) return '?'
  return name.split(' ').map(n => n[0]).join('').slice(0, 2).toUpperCase()
}
</script>

<template>
  <div class="events-page">
    <LeftNavPanel />

    <main class="events-content">
      <!-- Header -->
      <header class="events-header">
        <div class="header-left">
          <h1 class="page-title">Scheduled Events</h1>
          <span class="event-count">{{ voiceStore.scheduledConferences.length }} events</span>
        </div>
        <div class="header-actions">
          <button
            class="filter-btn"
            :class="{ active: showUpcomingOnly }"
            @click="toggleFilter"
          >
            <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z" />
            </svg>
            {{ showUpcomingOnly ? 'Upcoming Only' : 'All Events' }}
          </button>
          <button class="create-btn" @click="showCreateModal = true">
            <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
            </svg>
            Schedule Event
          </button>
        </div>
      </header>

      <!-- Events List -->
      <div class="events-list">
        <div v-if="voiceStore.loading" class="loading-state">
          <div class="spinner"></div>
          <span>Loading events...</span>
        </div>

        <div v-else-if="groupKeys.length === 0" class="empty-state">
          <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
          <h3>No scheduled events</h3>
          <p>Events you're invited to will appear here</p>
        </div>

        <template v-else>
          <div v-for="dateKey in groupKeys" :key="dateKey" class="date-group">
            <h2 class="date-header">{{ dateKey }}</h2>

            <div
              v-for="event in groupedConferences[dateKey]"
              :key="event.id"
              class="event-card"
              :class="{ selected: selectedEvent?.id === event.id }"
              @click="selectedEvent = event"
            >
              <div class="event-time">
                {{ formatTime(event.scheduled_at!) }}
              </div>

              <div class="event-details">
                <div class="event-header-row">
                  <h3 class="event-name">{{ event.name }}</h3>
                  <span
                    class="event-badge"
                    :class="getEventTypeBadge(event.event_type).class"
                  >
                    {{ getEventTypeBadge(event.event_type).label }}
                  </span>
                </div>

                <div class="event-meta">
                  <span class="accepted-count">
                    <svg class="meta-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
                    </svg>
                    {{ event.accepted_count }} accepted
                  </span>

                  <template v-if="getMyParticipation(event)">
                    <span
                      class="my-role"
                      :class="getRoleBadge(getMyParticipation(event)!.role).class"
                    >
                      {{ getRoleBadge(getMyParticipation(event)!.role).label }}
                    </span>
                  </template>
                </div>

                <!-- RSVP Status / Actions -->
                <div class="event-rsvp">
                  <template v-if="getMyParticipation(event)?.rsvp_status === 'pending'">
                    <button class="rsvp-btn accept" @click.stop="updateRSVP(event.id, 'accepted')">
                      Accept
                    </button>
                    <button class="rsvp-btn decline" @click.stop="updateRSVP(event.id, 'declined')">
                      Decline
                    </button>
                  </template>
                  <template v-else-if="getMyParticipation(event)?.rsvp_status === 'accepted'">
                    <span class="rsvp-status accepted">Accepted</span>
                    <button class="join-btn" @click.stop="joinEvent(event.id)">
                      Join
                    </button>
                  </template>
                  <template v-else-if="getMyParticipation(event)?.rsvp_status === 'declined'">
                    <span class="rsvp-status declined">Declined</span>
                    <button class="rsvp-btn accept small" @click.stop="updateRSVP(event.id, 'accepted')">
                      Change to Accept
                    </button>
                  </template>
                </div>
              </div>
            </div>
          </div>
        </template>
      </div>
    </main>

    <!-- Event Detail Sidebar -->
    <aside v-if="selectedEvent" class="event-sidebar">
      <div class="sidebar-header">
        <h2 class="sidebar-title">{{ selectedEvent.name }}</h2>
        <button class="close-btn" @click="selectedEvent = null">
          <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <div class="sidebar-content">
        <!-- Event Info -->
        <div class="info-section">
          <h3 class="section-title">Details</h3>
          <div class="info-row">
            <svg class="info-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            <span>{{ new Date(selectedEvent.scheduled_at!).toLocaleDateString([], { weekday: 'long', month: 'long', day: 'numeric', year: 'numeric' }) }}</span>
          </div>
          <div class="info-row">
            <svg class="info-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <span>{{ formatTime(selectedEvent.scheduled_at!) }}</span>
          </div>
          <div class="info-row">
            <svg class="info-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
            </svg>
            <span>{{ selectedEvent.accepted_count }} accepted, {{ selectedEvent.declined_count }} declined</span>
          </div>
        </div>

        <!-- Participants -->
        <div class="participants-section">
          <h3 class="section-title">Participants ({{ selectedEvent.participants?.length || 0 }})</h3>
          <div class="participants-list">
            <div
              v-for="participant in selectedEvent.participants"
              :key="participant.user_id"
              class="participant-row"
            >
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
              <div class="participant-info">
                <span class="participant-name">
                  {{ participant.display_name || participant.username }}
                </span>
                <span
                  class="participant-role"
                  :class="getRoleBadge(participant.role).class"
                >
                  {{ getRoleBadge(participant.role).label }}
                </span>
              </div>
              <span
                class="participant-rsvp"
                :class="`rsvp-${participant.rsvp_status}`"
              >
                {{ participant.rsvp_status }}
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- Sidebar Actions -->
      <div class="sidebar-actions">
        <template v-if="getMyParticipation(selectedEvent)?.role === 'originator'">
          <button class="action-btn danger" @click="cancelEvent(selectedEvent.id)">
            Cancel Event
          </button>
        </template>
      </div>
    </aside>
  </div>
</template>

<style scoped>
.events-page {
  display: flex;
  height: 100vh;
  background: #f8fafc;
}

.events-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.events-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 24px;
  background: #fff;
  border-bottom: 1px solid #e5e7eb;
}

.header-left {
  display: flex;
  align-items: baseline;
  gap: 12px;
}

.page-title {
  font-size: 24px;
  font-weight: 600;
  color: #1f2937;
  margin: 0;
}

.event-count {
  font-size: 14px;
  color: #6b7280;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.filter-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
  font-size: 14px;
  color: #6b7280;
  cursor: pointer;
  transition: all 0.15s ease;
}

.filter-btn:hover {
  border-color: #d1d5db;
  color: #374151;
}

.filter-btn.active {
  background: #eef2ff;
  border-color: #c7d2fe;
  color: #4f46e5;
}

.filter-btn .icon {
  width: 16px;
  height: 16px;
}

.create-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border: none;
  border-radius: 8px;
  background: #4f46e5;
  font-size: 14px;
  font-weight: 500;
  color: #fff;
  cursor: pointer;
  transition: all 0.15s ease;
}

.create-btn:hover {
  background: #4338ca;
}

.create-btn .icon {
  width: 16px;
  height: 16px;
}

.events-list {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
}

.loading-state,
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  color: #6b7280;
}

.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid #e5e7eb;
  border-top-color: #4f46e5;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  margin-bottom: 16px;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.empty-icon {
  width: 64px;
  height: 64px;
  color: #d1d5db;
  margin-bottom: 16px;
}

.empty-state h3 {
  margin: 0 0 8px;
  font-size: 18px;
  font-weight: 600;
  color: #374151;
}

.empty-state p {
  margin: 0;
  font-size: 14px;
}

.date-group {
  margin-bottom: 24px;
}

.date-header {
  font-size: 14px;
  font-weight: 600;
  color: #6b7280;
  margin: 0 0 12px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.event-card {
  display: flex;
  gap: 16px;
  padding: 16px;
  background: #fff;
  border-radius: 12px;
  border: 1px solid #e5e7eb;
  margin-bottom: 12px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.event-card:hover {
  border-color: #c7d2fe;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
}

.event-card.selected {
  border-color: #4f46e5;
  box-shadow: 0 0 0 3px rgba(79, 70, 229, 0.1);
}

.event-time {
  font-size: 14px;
  font-weight: 600;
  color: #4f46e5;
  min-width: 60px;
}

.event-details {
  flex: 1;
}

.event-header-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 8px;
}

.event-name {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #1f2937;
}

.event-badge {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 500;
  text-transform: uppercase;
}

.badge-scheduled {
  background: #dbeafe;
  color: #1d4ed8;
}

.badge-recurring {
  background: #fef3c7;
  color: #b45309;
}

.badge-adhoc {
  background: #d1fae5;
  color: #047857;
}

.badge-default {
  background: #e5e7eb;
  color: #6b7280;
}

.event-meta {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 12px;
}

.accepted-count {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: #6b7280;
}

.meta-icon {
  width: 14px;
  height: 14px;
}

.my-role {
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 500;
}

.role-organizer {
  background: #fef3c7;
  color: #b45309;
}

.role-moderator {
  background: #dbeafe;
  color: #1d4ed8;
}

.role-speaker {
  background: #d1fae5;
  color: #047857;
}

.role-assistant {
  background: #ede9fe;
  color: #6d28d9;
}

.role-participant {
  background: #e5e7eb;
  color: #6b7280;
}

.event-rsvp {
  display: flex;
  align-items: center;
  gap: 10px;
}

.rsvp-btn {
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
}

.rsvp-btn.accept {
  border: none;
  background: #22c55e;
  color: #fff;
}

.rsvp-btn.accept:hover {
  background: #16a34a;
}

.rsvp-btn.decline {
  border: 1px solid #e5e7eb;
  background: #fff;
  color: #6b7280;
}

.rsvp-btn.decline:hover {
  border-color: #ef4444;
  color: #ef4444;
}

.rsvp-btn.small {
  font-size: 12px;
  padding: 4px 10px;
}

.rsvp-status {
  font-size: 13px;
  font-weight: 500;
}

.rsvp-status.accepted {
  color: #22c55e;
}

.rsvp-status.declined {
  color: #ef4444;
}

.join-btn {
  padding: 6px 14px;
  border: none;
  border-radius: 6px;
  background: #4f46e5;
  color: #fff;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
}

.join-btn:hover {
  background: #4338ca;
}

/* Sidebar */
.event-sidebar {
  width: 360px;
  background: #fff;
  border-left: 1px solid #e5e7eb;
  display: flex;
  flex-direction: column;
}

.sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid #e5e7eb;
}

.sidebar-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #1f2937;
}

.close-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: #6b7280;
  cursor: pointer;
}

.close-btn:hover {
  background: #f3f4f6;
}

.close-btn .icon {
  width: 18px;
  height: 18px;
}

.sidebar-content {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.info-section,
.participants-section {
  margin-bottom: 24px;
}

.section-title {
  font-size: 12px;
  font-weight: 600;
  color: #6b7280;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin: 0 0 12px;
}

.info-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 0;
  font-size: 14px;
  color: #374151;
}

.info-icon {
  width: 18px;
  height: 18px;
  color: #6b7280;
}

.participants-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.participant-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px;
  border-radius: 8px;
}

.participant-row:hover {
  background: #f9fafb;
}

.participant-avatar {
  width: 36px;
  height: 36px;
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
  font-size: 13px;
  font-weight: 600;
  color: #6b7280;
}

.participant-info {
  flex: 1;
}

.participant-name {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: #1f2937;
}

.participant-role {
  font-size: 11px;
  padding: 1px 4px;
  border-radius: 3px;
}

.participant-rsvp {
  font-size: 12px;
  font-weight: 500;
  text-transform: capitalize;
}

.rsvp-pending {
  color: #f59e0b;
}

.rsvp-accepted {
  color: #22c55e;
}

.rsvp-declined {
  color: #ef4444;
}

.sidebar-actions {
  padding: 16px 20px;
  border-top: 1px solid #e5e7eb;
}

.action-btn {
  width: 100%;
  padding: 10px 16px;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
}

.action-btn.danger {
  background: #fef2f2;
  color: #ef4444;
}

.action-btn.danger:hover {
  background: #fee2e2;
}
</style>
