/**
 * Task presentation metadata (covers, accents, labels) that the API does not provide.
 *
 * Lookup order for a task: known task id -> task.category from the API -> neutral fallback.
 * Usage:
 *   const v = taskVisual(task)              // { cover, accent, categoryLabel }
 *   const d = difficultyMeta(task.difficulty) // { label: 'Oson' | "O'rta" | 'Qiyin', color }
 *   const s = taskStatus(task, session.taskId) // 'done' | 'active' | 'open' | 'locked'
 */
import type { TaskSummary } from '@/types'

export interface TaskVisual {
  /** Public URL of the cover SVG. */
  cover: string
  /** Hex accent color. */
  accent: string
  /** Uzbek category label. */
  categoryLabel: string
}

export interface DifficultyMeta {
  label: 'Oson' | "O'rta" | 'Qiyin'
  color: string
}

export type TaskStatus = 'done' | 'active' | 'open' | 'locked'

const cover = (name: string) => `/assets/cover-${name}.svg`

/** Per-category visuals; also used as the fallback for unknown task ids. */
export const CATEGORY_VISUALS: Record<string, TaskVisual> = {
  web: { cover: cover('nginx'), accent: '#3DDC97', categoryLabel: 'Web' },
  containers: { cover: cover('docker'), accent: '#38BDF8', categoryLabel: 'Docker' },
  services: { cover: cover('systemd'), accent: '#A78BFA', categoryLabel: 'Servislar' },
  users: { cover: cover('users'), accent: '#F5B342', categoryLabel: 'Foydalanuvchilar' },
  network: { cover: cover('firewall'), accent: '#FB7185', categoryLabel: 'Tarmoq xavfsizligi' },
  automation: { cover: cover('cron'), accent: '#2DD4BF', categoryLabel: 'Avtomatlashtirish' }
}

/** Known task ids (backend-go/tasks/<id>). */
export const TASK_VISUALS: Record<string, TaskVisual> = {
  'nginx-health': CATEGORY_VISUALS.web,
  'docker-basics': CATEGORY_VISUALS.containers,
  'systemd-service': CATEGORY_VISUALS.services,
  'user-management': CATEGORY_VISUALS.users,
  'firewall-config': CATEGORY_VISUALS.network,
  'cron-backup': CATEGORY_VISUALS.automation
}

const FALLBACK_VISUAL: TaskVisual = {
  cover: '/assets/servers.svg',
  accent: '#94A3B8',
  categoryLabel: 'Boshqa'
}

function capitalize(value: string): string {
  return value ? value.charAt(0).toUpperCase() + value.slice(1) : value
}

/** Uzbek label for an API category string (falls back to the capitalized raw value). */
export function categoryLabel(category: string | null | undefined): string {
  if (!category) return FALLBACK_VISUAL.categoryLabel
  return CATEGORY_VISUALS[category]?.categoryLabel ?? capitalize(category)
}

/** Visuals for a task; never throws, works for unknown ids. */
export function taskVisual(task: { id: string; category?: string | null }): TaskVisual {
  const byId = TASK_VISUALS[task.id]
  if (byId) return byId
  const byCategory = task.category ? CATEGORY_VISUALS[task.category] : undefined
  if (byCategory) return byCategory
  return { ...FALLBACK_VISUAL, categoryLabel: categoryLabel(task.category) }
}

export const DIFFICULTY: Record<1 | 2 | 3, DifficultyMeta> = {
  1: { label: 'Oson', color: '#3DDC97' },
  2: { label: "O'rta", color: '#F5B342' },
  3: { label: 'Qiyin', color: '#FB7185' }
}

/** Difficulty label + color; out-of-range values are clamped to 1..3. */
export function difficultyMeta(difficulty: number): DifficultyMeta {
  const d = Math.min(3, Math.max(1, Math.round(difficulty || 1))) as 1 | 2 | 3
  return DIFFICULTY[d]
}

export const TASK_STATUS_LABEL: Record<TaskStatus, string> = {
  done: 'Yakunlangan',
  active: 'Jarayonda',
  open: 'Yangi',
  locked: 'Qulflangan'
}

/**
 * Derive the UI status of a task. `activeTaskId` (from the session store) marks the task
 * with a running container as active even if the API has not flagged it yet.
 */
export function taskStatus(
  task: Pick<TaskSummary, 'id' | 'completed' | 'in_progress' | 'unlocked'>,
  activeTaskId?: string | null
): TaskStatus {
  if (task.completed) return 'done'
  if (task.in_progress || (activeTaskId && task.id === activeTaskId)) return 'active'
  if (!task.unlocked) return 'locked'
  return 'open'
}

/** Two-digit task number from `order`, e.g. 3 -> "03". */
export function taskNumber(order: number): string {
  return String(order).padStart(2, '0')
}
