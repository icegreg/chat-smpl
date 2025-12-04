import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { User, LoginRequest, RegisterRequest } from '@/types'
import { api, ApiError } from '@/api/client'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const isAuthenticated = computed(() => !!user.value)
  const isGuest = computed(() => user.value?.role === 'guest')

  async function init() {
    const token = localStorage.getItem('access_token')
    if (token) {
      try {
        await fetchCurrentUser()
      } catch {
        // Token expired, try to refresh
        try {
          await api.refreshToken()
          await fetchCurrentUser()
        } catch {
          // Refresh failed, clear auth state
          logout()
        }
      }
    }
  }

  async function login(data: LoginRequest) {
    loading.value = true
    error.value = null
    try {
      await api.login(data)
      await fetchCurrentUser()
    } catch (e) {
      error.value = e instanceof ApiError ? e.message : 'Login failed'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function register(data: RegisterRequest) {
    loading.value = true
    error.value = null
    try {
      await api.register(data)
      await fetchCurrentUser()
    } catch (e) {
      error.value = e instanceof ApiError ? e.message : 'Registration failed'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function logout() {
    try {
      await api.logout()
    } catch {
      // Ignore logout errors
    }
    user.value = null
    api.setAccessToken(null)
    localStorage.removeItem('refresh_token')
  }

  async function fetchCurrentUser() {
    user.value = await api.getCurrentUser()
  }

  async function updateProfile(data: Partial<User>) {
    loading.value = true
    error.value = null
    try {
      user.value = await api.updateCurrentUser(data)
    } catch (e) {
      error.value = e instanceof ApiError ? e.message : 'Update failed'
      throw e
    } finally {
      loading.value = false
    }
  }

  return {
    user,
    loading,
    error,
    isAuthenticated,
    isGuest,
    init,
    login,
    register,
    logout,
    fetchCurrentUser,
    updateProfile,
  }
})
