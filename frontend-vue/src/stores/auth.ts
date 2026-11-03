/**
 * Auth store. The user lives in memory only; the JWT is an httpOnly cookie the JS never sees.
 *
 * State:   user, initialized, isLoading
 * Getters: isAuthenticated, isAdmin, initials
 * Actions: init() (GET /api/me once; safe to await many times), login(), register(), logout(),
 *          refresh(), setUser(), clear()
 * login/register throw `ApiError` on failure (use `errorMessage(err)` from services/api).
 */
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api } from '@/services/api'
import type { LoginRequest, RegisterRequest, User } from '@/types'
import { useSessionStore } from './session'
import { useTaskStore } from './task'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const initialized = ref(false)
  const isLoading = ref(false)
  let initPromise: Promise<void> | null = null

  const isAuthenticated = computed(() => user.value !== null)
  const isAdmin = computed(() => user.value?.role === 'admin')
  const initials = computed(() => (user.value?.username ?? '').slice(0, 2).toUpperCase())

  /** Resolve the current user from the cookie. Runs the request only once per page load. */
  function init(): Promise<void> {
    if (!initPromise) {
      initPromise = (async () => {
        try {
          user.value = await api.getMe()
        } catch {
          user.value = null
        } finally {
          initialized.value = true
        }
      })()
    }
    return initPromise
  }

  async function refresh(): Promise<void> {
    try {
      user.value = await api.getMe()
    } catch {
      user.value = null
    }
  }

  async function login(credentials: LoginRequest): Promise<User> {
    isLoading.value = true
    try {
      user.value = await api.login(credentials)
      initialized.value = true
      return user.value
    } finally {
      isLoading.value = false
    }
  }

  async function register(payload: RegisterRequest): Promise<User> {
    isLoading.value = true
    try {
      user.value = await api.register(payload)
      initialized.value = true
      return user.value
    } finally {
      isLoading.value = false
    }
  }

  /** Clears the cookie server-side (best effort) and resets all user-scoped state. */
  async function logout(): Promise<void> {
    try {
      await api.logout()
    } catch {
      // Cookie may already be invalid; local state is cleared regardless.
    } finally {
      clear()
    }
  }

  function setUser(next: User): void {
    user.value = next
  }

  /** Drop all user-scoped client state (used on logout and on any 401). */
  function clear(): void {
    user.value = null
    useSessionStore().clear()
    useTaskStore().clear()
  }

  return {
    user,
    initialized,
    isLoading,
    isAuthenticated,
    isAdmin,
    initials,
    init,
    refresh,
    login,
    register,
    logout,
    setUser,
    clear
  }
})
