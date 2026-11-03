/**
 * Pure helpers for the dashboard: lab-time formatting, "continue" task selection and the
 * activity log derived from the user's own progress data (no invented events).
 */
import type { Achievement, CheckResponse, ProgressEntry, SessionResponse, TaskSummary } from '@/types'

/** Seconds -> "3s 40d" (soat / daqiqa); under an hour -> "40d". */
export function formatLabTime(totalSeconds: number): string {
  const s = Math.max(0, Math.floor(totalSeconds || 0))
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  return h > 0 ? `${h}s ${m}d` : `${m}d`
}

/** "sardor" -> "Sardor". */
export function displayName(username: string | undefined | null): string {
  if (!username) return ''
  return username.charAt(0).toUpperCase() + username.slice(1)
}

/**
 * Task the hero should offer to continue:
 * active session task -> first in-progress task -> first unlocked unfinished task -> first unfinished task.
 * Returns undefined when every task is completed (or there are none).
 */
export function pickContinueTask(tasks: TaskSummary[], activeTaskId: string | null): TaskSummary | undefined {
  const bySession = activeTaskId ? tasks.find((t) => t.id === activeTaskId) : undefined
  if (bySession) return bySession
  return (
    tasks.find((t) => t.in_progress && !t.completed) ??
    tasks.find((t) => t.unlocked && !t.completed) ??
    tasks.find((t) => !t.completed)
  )
}

/** Check progress for a task: the latest check of the running session wins, then stored progress. */
export function checkProgress(
  task: TaskSummary,
  entry: ProgressEntry | undefined,
  lastCheck: CheckResponse | null,
  activeTaskId: string | null
): { done: number; total: number } {
  if (lastCheck && activeTaskId === task.id && lastCheck.total > 0) {
    return { done: Math.min(lastCheck.score, lastCheck.total), total: lastCheck.total }
  }
  const total = entry && entry.total > 0 ? entry.total : task.checks_count
  const done = entry ? Math.min(entry.passed ? total : entry.score, total) : 0
  return { done, total }
}

export type ActivityTone = 'done' | 'check' | 'achievement' | 'session'

export interface ActivityEvent {
  key: string
  /** Epoch ms; null for events the API gives no timestamp for (attempt counts). */
  at: number | null
  text: string
  tone: ActivityTone
}

function parseTime(value: string | null | undefined): number | null {
  if (!value) return null
  const t = Date.parse(value)
  return Number.isNaN(t) ? null : t
}

/**
 * Build the activity log from real data only:
 * - completed tasks (completed_at, "+points"),
 * - unfinished tasks with attempts (best score so far; undated — the API stores no attempt time),
 * - earned achievements (earned_at),
 * - the running session (createdAt).
 * Dated events are sorted newest first; undated attempt rows go on top as "current" work.
 */
export function buildActivity(input: {
  progress: ProgressEntry[]
  tasks: TaskSummary[]
  achievements: Achievement[]
  session: SessionResponse | null
  limit?: number
}): ActivityEvent[] {
  const points = new Map(input.tasks.map((t) => [t.id, t.points]))
  const dated: ActivityEvent[] = []
  const undated: ActivityEvent[] = []

  for (const p of input.progress) {
    const completedAt = parseTime(p.completed_at)
    if (p.passed && completedAt !== null) {
      const pts = points.get(p.task_id)
      dated.push({
        key: `done-${p.task_id}`,
        at: completedAt,
        text: `${p.task_id} completed${pts ? ` +${pts}` : ''}`,
        tone: 'done'
      })
    } else if (!p.passed && p.attempts > 0) {
      undated.push({
        key: `check-${p.task_id}`,
        at: null,
        text: `${p.task_id}: check ${p.score}/${p.total} passed · ${p.attempts} urinish`,
        tone: 'check'
      })
    }
  }

  for (const a of input.achievements) {
    const earnedAt = parseTime(a.earned_at)
    if (a.earned && earnedAt !== null) {
      dated.push({ key: `ach-${a.id}`, at: earnedAt, text: `achievement: ${a.id}`, tone: 'achievement' })
    }
  }

  const sessionAt = parseTime(input.session?.createdAt)
  if (input.session && sessionAt !== null) {
    dated.push({
      key: `session-${input.session.sessionId}`,
      at: sessionAt,
      text: `session started (${input.session.taskId})`,
      tone: 'session'
    })
  }

  dated.sort((a, b) => (b.at ?? 0) - (a.at ?? 0))
  return [...undated, ...dated].slice(0, input.limit ?? 6)
}

/** Timestamp label for the log: "13:42" today, "kecha" yesterday, otherwise "01.11". */
export function formatWhen(at: number | null, now: Date = new Date()): string {
  if (at === null) return '--:--'
  const d = new Date(at)
  const startOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime()
  const pad = (n: number) => String(n).padStart(2, '0')
  if (at >= startOfToday) return `${pad(d.getHours())}:${pad(d.getMinutes())}`
  if (at >= startOfToday - 86_400_000) return 'kecha'
  return `${pad(d.getDate())}.${pad(d.getMonth() + 1)}`
}
