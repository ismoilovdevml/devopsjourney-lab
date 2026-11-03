/**
 * Tasks page filter model. Filters live in the URL query so they survive reloads and can be shared:
 *   /tasks?q=nginx&status=new&level=easy&task=nginx-health
 */
import type { LocationQuery, LocationQueryValue } from 'vue-router'
import { categoryLabel, type TaskStatus } from '@/constants/tasks'
import type { TaskSummary } from '@/types'

export type StatusFilter = 'all' | 'new' | 'active' | 'done'
export type LevelFilter = 'all' | 'easy' | 'medium' | 'hard'

export const STATUS_OPTIONS: { value: StatusFilter; label: string }[] = [
  { value: 'all', label: 'Barchasi' },
  { value: 'new', label: 'Yangi' },
  { value: 'active', label: 'Jarayonda' },
  { value: 'done', label: 'Yakunlangan' }
]

export const LEVEL_OPTIONS: { value: LevelFilter; label: string }[] = [
  { value: 'all', label: 'Har qanday' },
  { value: 'easy', label: 'Oson' },
  { value: 'medium', label: "O'rta" },
  { value: 'hard', label: 'Qiyin' }
]

const LEVEL_DIFFICULTY: Record<Exclude<LevelFilter, 'all'>, number> = { easy: 1, medium: 2, hard: 3 }

/** First string value of a query param ('' when absent). */
export function queryString(value: LocationQueryValue | LocationQueryValue[] | undefined): string {
  const v = Array.isArray(value) ? value[0] : value
  return typeof v === 'string' ? v : ''
}

export function parseStatus(query: LocationQuery): StatusFilter {
  const v = queryString(query.status)
  return STATUS_OPTIONS.some((o) => o.value === v) ? (v as StatusFilter) : 'all'
}

export function parseLevel(query: LocationQuery): LevelFilter {
  const v = queryString(query.level)
  return LEVEL_OPTIONS.some((o) => o.value === v) ? (v as LevelFilter) : 'all'
}

/** "Yangi" also lists locked tasks (not started yet), as in the design. */
export function matchesStatus(status: TaskStatus, filter: StatusFilter): boolean {
  if (filter === 'all') return true
  if (filter === 'new') return status === 'open' || status === 'locked'
  return status === filter
}

export function matchesLevel(difficulty: number, filter: LevelFilter): boolean {
  return filter === 'all' || Math.round(difficulty) === LEVEL_DIFFICULTY[filter]
}

/** Case-insensitive match on title, description, category (raw + Uzbek label) and id. */
export function matchesSearch(task: TaskSummary, search: string): boolean {
  const q = search.trim().toLowerCase()
  if (!q) return true
  const haystack = [task.title, task.description, task.category, categoryLabel(task.category), task.id]
    .join(' ')
    .toLowerCase()
  return haystack.includes(q)
}
