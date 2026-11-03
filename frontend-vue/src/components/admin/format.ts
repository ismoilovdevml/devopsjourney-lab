/**
 * Formatting helpers shared by the admin panel sections.
 * All functions are pure and tolerate missing / malformed input (they return "—").
 */

const DASH = '—'

/** Parses an ISO timestamp; null for empty, invalid or Go zero-value ("0001-01-01...") dates. */
export function parseTime(iso: string | null | undefined): Date | null {
  if (!iso) return null
  const d = new Date(iso)
  if (Number.isNaN(d.getTime()) || d.getFullYear() < 2000) return null
  return d
}

/** "hozir", "5 daq oldin", "2 soat oldin", "3 kun oldin", or a date for older values. */
export function relativeTime(iso: string | null | undefined, now = Date.now()): string {
  const d = parseTime(iso)
  if (!d) return DASH
  const sec = Math.max(0, Math.round((now - d.getTime()) / 1000))
  if (sec < 60) return 'hozir'
  const min = Math.floor(sec / 60)
  if (min < 60) return `${min} daq oldin`
  const hours = Math.floor(min / 60)
  if (hours < 24) return `${hours} soat oldin`
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days} kun oldin`
  return shortDate(d)
}

const pad = (n: number) => String(n).padStart(2, '0')

function shortDate(d: Date): string {
  return `${pad(d.getDate())}.${pad(d.getMonth() + 1)}.${d.getFullYear()}`
}

/** Full local date-time for tooltips, e.g. "03.11.2026 13:42:07". */
export function fullDateTime(iso: string | null | undefined): string {
  const d = parseTime(iso)
  if (!d) return DASH
  return `${shortDate(d)} ${clock(d)}`
}

function clock(d: Date): string {
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

/** "13:42:07" for today, "02.11 13:42:07" otherwise (journalctl-like log timestamp). */
export function logTime(iso: string | null | undefined, now = new Date()): string {
  const d = parseTime(iso)
  if (!d) return DASH
  const sameDay = d.toDateString() === now.toDateString()
  return sameDay ? clock(d) : `${pad(d.getDate())}.${pad(d.getMonth() + 1)} ${clock(d)}`
}

/** Seconds -> "mm:ss" or "h:mm:ss"; negative values clamp to 00:00. */
export function duration(totalSeconds: number): string {
  const s = Math.max(0, Math.floor(totalSeconds))
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const sec = s % 60
  return h > 0 ? `${h}:${pad(m)}:${pad(sec)}` : `${pad(m)}:${pad(sec)}`
}

/** Docker-style short byte size with binary units: "512K", "92M", "1.5G". */
export function bytes(value: number | null | undefined): string {
  if (value === null || value === undefined || !Number.isFinite(value) || value < 0) return DASH
  const units = ['B', 'K', 'M', 'G', 'T']
  let v = value
  let i = 0
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i += 1
  }
  const digits = v >= 10 || i === 0 ? 0 : 1
  return `${v.toFixed(digits)}${units[i]}`
}

/** Clamp to 0..100 (NaN -> 0). */
export function clampPercent(value: number): number {
  if (!Number.isFinite(value)) return 0
  return Math.min(100, Math.max(0, value))
}

/** Percent with at most one decimal, without a trailing ".0". */
export function percentLabel(value: number): string {
  if (!Number.isFinite(value)) return DASH
  const rounded = Math.round(value * 10) / 10
  return `${rounded % 1 === 0 ? rounded.toFixed(0) : rounded.toFixed(1)}%`
}

/** Integer with space-separated thousands (Uzbek convention), e.g. "12 345". */
export function count(value: number | null | undefined): string {
  if (value === null || value === undefined || !Number.isFinite(value)) return DASH
  return Math.round(value).toLocaleString('ru-RU').replace(/[\u00a0\u202f]/g, ' ')
}
