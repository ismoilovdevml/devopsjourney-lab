import { defineStore } from 'pinia'
import { ref, computed, onScopeDispose } from 'vue'
import { api } from '@/services/api'
import type { SessionResponse, CheckResponse } from '@/types'

export const useSessionStore = defineStore('session', () => {
  // Load from localStorage on init
  const sessionId = ref<string | null>(localStorage.getItem('session_id'))
  const expiresAt = ref<Date | null>(null)
  const timeRemaining = ref(parseInt(localStorage.getItem('time_remaining') || '0'))
  const isCreating = ref(false)
  const isChecking = ref(false)
  const checkResult = ref<CheckResponse | null>(null)
  const error = ref<string | null>(null)

  // Track interval ID for cleanup
  let countdownInterval: number | null = null

  const hasActiveSession = computed(() => !!sessionId.value)

  async function createSession(taskId: string) {
    isCreating.value = true
    error.value = null
    console.log('Creating session for task:', taskId)
    try {
      const response: SessionResponse = await api.createSession(taskId)
      console.log('Session created:', response)
      sessionId.value = response.sessionId
      expiresAt.value = new Date(response.expiresAt)
      timeRemaining.value = response.ttlSeconds
      checkResult.value = null

      // Persist to localStorage
      localStorage.setItem('session_id', response.sessionId)
      localStorage.setItem('time_remaining', response.ttlSeconds.toString())

      // Start countdown
      startCountdown()

      return response
    } catch (err: any) {
      console.error('Session creation error:', err)
      error.value = err.response?.data?.error || err.message || 'Failed to create session'
      throw err
    } finally {
      isCreating.value = false
    }
  }

  async function deleteSession() {
    if (!sessionId.value) return

    try {
      await api.deleteSession(sessionId.value)
      clearSession()
    } catch (err: any) {
      console.error('Failed to delete session:', err)
    }
  }

  async function checkSolution(taskId?: string) {
    if (!sessionId.value) return

    isChecking.value = true
    error.value = null
    try {
      checkResult.value = await api.checkSolution(sessionId.value, taskId)
      return checkResult.value
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Failed to check solution'
      throw err
    } finally {
      isChecking.value = false
    }
  }

  function stopCountdown() {
    if (countdownInterval !== null) {
      clearInterval(countdownInterval)
      countdownInterval = null
    }
  }

  function clearSession() {
    stopCountdown()
    sessionId.value = null
    expiresAt.value = null
    timeRemaining.value = 0
    checkResult.value = null
    error.value = null

    // Clear localStorage
    localStorage.removeItem('session_id')
    localStorage.removeItem('time_remaining')
  }

  function startCountdown() {
    // Stop existing interval first (prevent memory leak)
    stopCountdown()

    countdownInterval = window.setInterval(() => {
      if (timeRemaining.value > 0) {
        timeRemaining.value--
        localStorage.setItem('time_remaining', timeRemaining.value.toString())
      } else {
        stopCountdown()
        clearSession()
      }
    }, 1000)
  }

  // Cleanup when store is disposed (fixes memory leak)
  onScopeDispose(() => {
    stopCountdown()
  })

  return {
    sessionId,
    expiresAt,
    timeRemaining,
    isCreating,
    isChecking,
    checkResult,
    error,
    hasActiveSession,
    createSession,
    deleteSession,
    checkSolution,
    clearSession,
    stopCountdown
  }
})
