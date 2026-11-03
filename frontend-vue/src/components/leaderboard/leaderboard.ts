/**
 * Pure helpers for the leaderboard page (no Vue, no API calls).
 */
import type { LeaderboardEntry, LeaderboardPeriod } from '@/types'

export const PERIODS: readonly { value: LeaderboardPeriod; label: string }[] = [
  { value: 'week', label: 'Bu hafta' },
  { value: 'month', label: 'Bu oy' },
  { value: 'all', label: 'Barcha vaqt' }
]

export const DEFAULT_PERIOD: LeaderboardPeriod = 'all'

/** Parse a `?period=` query value; anything unknown falls back to the default. */
export function parsePeriod(value: unknown): LeaderboardPeriod {
  const raw = Array.isArray(value) ? value[0] : value
  return PERIODS.some((p) => p.value === raw) ? (raw as LeaderboardPeriod) : DEFAULT_PERIOD
}

/** Medal colors by rank (1..3). */
export const MEDAL: Record<1 | 2 | 3, string> = {
  1: '#F5B342',
  2: '#C9D4E0',
  3: '#E0915A'
}

export function medalColor(rank: number): string {
  return rank >= 1 && rank <= 3 ? MEDAL[rank as 1 | 2 | 3] : '#94A3B8'
}

/** Two-letter avatar initials, same rule as the auth store. */
export function initials(username: string): string {
  return username.slice(0, 2).toUpperCase()
}

/** Rank shown in the table, e.g. 3 -> "03". */
export function rankLabel(rank: number): string {
  return String(rank).padStart(2, '0')
}

/**
 * XP needed to reach the next rank above `me`, or null when it cannot be derived from the
 * returned entries (rank 1, or the user directly above is outside the returned list).
 * Ranks are dense (equal scores share a rank), so the next rank is exactly `me.rank - 1`.
 */
export function gapToNextRank(me: LeaderboardEntry, entries: LeaderboardEntry[]): number | null {
  if (me.rank <= 1) return null
  const above = entries.find((e) => e.rank === me.rank - 1)
  if (!above) return null
  const gap = above.score - me.score
  return gap > 0 ? gap : null
}
