/**
 * Active lab session (one per user, enforced by the backend).
 *
 * State:   session (SessionResponse | null), lastCheck, expired, isRestoring/isStarting/isRecreating/
 *          isStopping/isChecking
 * Getters: sessionId, taskId, expiresAt (Date | null), isActive, remainingSeconds (ticks every 1s),
 *          remainingLabel ("mm:ss" or "h:mm:ss")
 * Actions: restore() (GET /api/sessions/active), start(taskId), recreate(), stop(), check(), clear()
 * Actions throw `ApiError` on failure. When the timer reaches 0 the session is cleared and
 * `expired` is set to true (reset by the next start/restore).
 */
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api } from '@/services/api'
import type { CheckResponse, SessionResponse } from '@/types'

function deadlineOf(s: SessionResponse): number {
  const t = Date.parse(s.expiresAt)
  return Number.isNaN(t) ? Date.now() + s.ttlSeconds * 1000 : t
}

export function formatDuration(totalSeconds: number): string {
  const s = Math.max(0, Math.floor(totalSeconds))
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const sec = s % 60
  const mm = String(m).padStart(2, '0')
  const ss = String(sec).padStart(2, '0')
  return h > 0 ? `${h}:${mm}:${ss}` : `${mm}:${ss}`
}

export const useSessionStore = defineStore('session', () => {
  const session = ref<SessionResponse | null>(null)
  const lastCheck = ref<CheckResponse | null>(null)
  const expired = ref(false)
  const isRestoring = ref(false)
  const isStarting = ref(false)
  const isRecreating = ref(false)
  const isStopping = ref(false)
  const isChecking = ref(false)

  const now = ref(Date.now())
  let ticker: ReturnType<typeof setInterval> | null = null

  const sessionId = computed(() => session.value?.sessionId ?? null)
  const taskId = computed(() => session.value?.taskId ?? null)
  const expiresAt = computed(() => (session.value ? new Date(deadlineOf(session.value)) : null))
  const isActive = computed(() => session.value !== null)
  const remainingSeconds = computed(() =>
    session.value ? Math.max(0, Math.floor((deadlineOf(session.value) - now.value) / 1000)) : 0
  )
  const remainingLabel = computed(() => formatDuration(remainingSeconds.value))

  function stopTicker(): void {
    if (ticker !== null) {
      clearInterval(ticker)
      ticker = null
    }
  }

  function startTicker(): void {
    stopTicker()
    now.value = Date.now()
    ticker = setInterval(() => {
      now.value = Date.now()
      if (session.value && remainingSeconds.value <= 0) {
        clear()
        expired.value = true
      }
    }, 1000)
  }

  function set(next: SessionResponse | null): void {
    session.value = next
    expired.value = false
    if (next) startTicker()
    else stopTicker()
  }

  /** Restore the active session after a page load. Returns null when there is none. */
  async function restore(): Promise<SessionResponse | null> {
    isRestoring.value = true
    try {
      const active = await api.getActiveSession()
      if (active?.sessionId !== session.value?.sessionId) lastCheck.value = null
      set(active)
      return active
    } finally {
      isRestoring.value = false
    }
  }

  /** Start a container for `taskId` (stops any previous session server-side). */
  async function start(taskIdToStart: string): Promise<SessionResponse> {
    isStarting.value = true
    try {
      const created = await api.createSession(taskIdToStart)
      lastCheck.value = null
      set(created)
      return created
    } finally {
      isStarting.value = false
    }
  }

  /** Replace the current container with a fresh one (new sessionId). */
  async function recreate(): Promise<SessionResponse | null> {
    if (!session.value) return null
    isRecreating.value = true
    try {
      const next = await api.recreateSession(session.value.sessionId)
      lastCheck.value = null
      set(next)
      return next
    } finally {
      isRecreating.value = false
    }
  }

  /** Stop the current session and remove its container. */
  async function stop(): Promise<void> {
    if (!session.value) return
    isStopping.value = true
    try {
      await api.deleteSession(session.value.sessionId)
      clear()
    } finally {
      isStopping.value = false
    }
  }

  /** Run the task checks in the current container. */
  async function check(): Promise<CheckResponse | null> {
    if (!session.value) return null
    isChecking.value = true
    try {
      lastCheck.value = await api.checkSolution(session.value.sessionId)
      return lastCheck.value
    } finally {
      isChecking.value = false
    }
  }

  function clear(): void {
    stopTicker()
    session.value = null
    lastCheck.value = null
    expired.value = false
  }

  return {
    session,
    lastCheck,
    expired,
    isRestoring,
    isStarting,
    isRecreating,
    isStopping,
    isChecking,
    sessionId,
    taskId,
    expiresAt,
    isActive,
    remainingSeconds,
    remainingLabel,
    restore,
    start,
    recreate,
    stop,
    check,
    clear
  }
})
