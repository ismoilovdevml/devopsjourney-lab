/**
 * Formatting helpers and page-local metadata for the Progress page.
 */
import type { Component } from 'vue'
import { Award, CalendarCheck, Flag, Flame, Star, Target, Trophy, Zap } from 'lucide-vue-next'

const MONTHS = [
  'yanvar',
  'fevral',
  'mart',
  'aprel',
  'may',
  'iyun',
  'iyul',
  'avgust',
  'sentabr',
  'oktabr',
  'noyabr',
  'dekabr'
]

/** ISO timestamp -> "3-noyabr" (year appended when it is not the current year). Empty/invalid -> null. */
export function formatDayMonth(iso: string | null | undefined): string | null {
  if (!iso) return null
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return null
  const base = `${d.getDate()}-${MONTHS[d.getMonth()]}`
  return d.getFullYear() === new Date().getFullYear() ? base : `${base} ${d.getFullYear()}`
}

/** Seconds -> compact Uzbek duration: "18 daq", "1s 12d", "<1 daq", "0 daq". */
export function formatDurationShort(totalSeconds: number): string {
  const s = Math.max(0, Math.floor(totalSeconds || 0))
  if (s === 0) return '0 daq'
  if (s < 60) return '<1 daq'
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  return h > 0 ? `${h}s ${m}d` : `${m} daq`
}

/** Seconds -> spelled-out Uzbek duration for screen readers: "1 soat 12 daqiqa". */
export function formatDurationLong(totalSeconds: number): string {
  const s = Math.max(0, Math.floor(totalSeconds || 0))
  if (s < 60) return s === 0 ? '0 daqiqa' : '1 daqiqadan kam'
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  return h > 0 ? `${h} soat ${m} daqiqa` : `${m} daqiqa`
}

export interface AchievementMeta {
  title: string
  description: string
  icon: Component
  color: string
}

/** Uzbek copy, icon and color per backend achievement id (backend sends English text). */
export const ACHIEVEMENT_META: Record<string, AchievementMeta> = {
  'first-blood': {
    title: 'Birinchi qadam',
    description: 'Birinchi vazifani yakunlang',
    icon: Star,
    color: '#3DDC97'
  },
  'half-way': {
    title: "Yarim yo'l",
    description: 'Vazifalarning yarmini yakunlang',
    icon: Flag,
    color: '#38BDF8'
  },
  'all-done': {
    title: 'DevOps Journey',
    description: 'Barcha vazifalarni yakunlang',
    icon: Trophy,
    color: '#A78BFA'
  },
  'streak-3': {
    title: 'Ritmda',
    description: '3 kun ketma-ket mashq qiling',
    icon: Flame,
    color: '#F5B342'
  },
  'streak-7': {
    title: 'Haftalik ritm',
    description: '7 kun ketma-ket mashq qiling',
    icon: CalendarCheck,
    color: '#FB7185'
  },
  speedrun: {
    title: 'Tezkor',
    description: 'Laboratoriyani boshlab, 5 daqiqa ichida yeching',
    icon: Zap,
    color: '#2DD4BF'
  },
  perfectionist: {
    title: 'Bir urinishda',
    description: '3 ta vazifani birinchi urinishda yeching',
    icon: Target,
    color: '#F5B342'
  }
}

const FALLBACK_COLORS = ['#3DDC97', '#38BDF8', '#A78BFA', '#F5B342', '#FB7185', '#2DD4BF']

/** Metadata for an achievement; unknown ids fall back to the API text and a generic icon. */
export function achievementMeta(a: { id: string; title: string; description: string }, index: number): AchievementMeta {
  return (
    ACHIEVEMENT_META[a.id] ?? {
      title: a.title,
      description: a.description,
      icon: Award,
      color: FALLBACK_COLORS[index % FALLBACK_COLORS.length]
    }
  )
}
