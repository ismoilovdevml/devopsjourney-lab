/**
 * Profile page helpers: level titles, date/role labels and client-side validation that mirrors
 * the backend rules (backend-go/internal/api/auth.go `validateProfile`, internal/auth/password.go).
 * Keep these in sync with the backend; the server remains the source of truth.
 */
import type { Role } from '@/types'

// ---------- Sections ----------

export type ProfileSection = 'profile' | 'appearance' | 'notifications' | 'security'

export const PROFILE_SECTIONS: readonly { id: ProfileSection; label: string }[] = [
  { id: 'profile', label: 'Profil' },
  { id: 'appearance', label: "Ko'rinish" },
  { id: 'notifications', label: 'Bildirishnomalar' },
  { id: 'security', label: 'Xavfsizlik' }
]

export function isProfileSection(value: unknown): value is ProfileSection {
  return typeof value === 'string' && PROFILE_SECTIONS.some((s) => s.id === value)
}

// ---------- Level title ----------

export interface LevelInfo {
  title: string
  color: string
  /** Completed tasks needed for the next level; null at the top level. */
  nextAt: number | null
}

/**
 * Level title from the number of completed tasks (GET /api/progress `completed`).
 * Thresholds (6 tasks ship today):
 *   0      -> Trainee Ops
 *   1–2    -> Junior Ops
 *   3–4    -> Middle Ops
 *   5+     -> Senior Ops
 */
const LEVELS: readonly { min: number; title: string; color: string }[] = [
  { min: 0, title: 'Trainee Ops', color: '#94A3B8' },
  { min: 1, title: 'Junior Ops', color: '#3DDC97' },
  { min: 3, title: 'Middle Ops', color: '#38BDF8' },
  { min: 5, title: 'Senior Ops', color: '#F5B342' }
]

export function levelFor(completed: number): LevelInfo {
  const n = Math.max(0, Math.floor(completed))
  let idx = 0
  LEVELS.forEach((l, i) => {
    if (n >= l.min) idx = i
  })
  const next = LEVELS[idx + 1]
  return { title: LEVELS[idx].title, color: LEVELS[idx].color, nextAt: next ? next.min : null }
}

// ---------- Labels ----------

export const ROLE_LABEL: Record<Role, string> = {
  student: "O'quvchi",
  admin: 'Administrator'
}

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

/** "2025-yil 1-noyabrdan beri"; empty string for an invalid date. */
export function memberSince(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return `${d.getFullYear()}-yil ${d.getDate()}-${MONTHS[d.getMonth()]}dan beri`
}

// ---------- Profile validation (mirrors validateProfile) ----------

export const GOAL_MAX = 200
export const EMAIL_MAX = 254
export const GITHUB_MAX = 39

const GITHUB_RE = /^[A-Za-z0-9-]{1,39}$/
// Pragmatic check; the backend uses the `email` binding as the final word.
const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

/** Rune count (matches Go's utf8.RuneCountInString). */
export function runeLength(value: string): number {
  return Array.from(value).length
}

/** Accepts "user", "@user" or a github.com URL and returns the bare username. */
export function normalizeGithub(value: string): string {
  return value
    .trim()
    .replace(/^(https?:\/\/)?(www\.)?github\.com\//i, '')
    .replace(/^@/, '')
    .replace(/\/+$/, '')
}

export interface ProfileFields {
  email: string
  github: string
  goal: string
}

export type ProfileErrors = Partial<Record<keyof ProfileFields, string>>

/** Normalized values exactly as they would be sent to PUT /api/me. */
export function normalizeProfile(f: ProfileFields): ProfileFields {
  return {
    email: f.email.trim().toLowerCase(),
    github: normalizeGithub(f.github),
    goal: f.goal.trim()
  }
}

export function validateProfile(f: ProfileFields): ProfileErrors {
  const v = normalizeProfile(f)
  const errors: ProfileErrors = {}
  if (!v.email) errors.email = 'Email kiritilishi shart.'
  else if (v.email.length > EMAIL_MAX || !EMAIL_RE.test(v.email)) errors.email = "Email manzili noto'g'ri."
  if (v.github && !GITHUB_RE.test(v.github)) {
    errors.github = `GitHub foydalanuvchi nomi: harf, raqam va '-', ko'pi bilan ${GITHUB_MAX} belgi.`
  }
  if (runeLength(v.goal) > GOAL_MAX) errors.goal = `Maqsad ko'pi bilan ${GOAL_MAX} belgi bo'lishi mumkin.`
  else if (v.goal.includes('\u0000')) errors.goal = "Maqsadda ruxsat etilmagan belgilar bor."
  return errors
}

/** Maps a backend 400 message (English) to a field + Uzbek text. */
export function profileServerError(message: string): ProfileErrors | null {
  const m = message.toLowerCase()
  if (m.includes('github')) return { github: `GitHub foydalanuvchi nomi noto'g'ri.` }
  if (m.includes('goal')) return { goal: `Maqsad ko'pi bilan ${GOAL_MAX} belgi bo'lishi mumkin.` }
  if (m.includes('email')) return { email: "Email manzili noto'g'ri." }
  return null
}

// ---------- Password rules (mirrors ValidatePasswordStrength) ----------

export const PASSWORD_MIN = 8
export const PASSWORD_MAX_BYTES = 72

// Same blocklist as backend-go/internal/auth/password.go `commonPasswords`.
const COMMON_PASSWORDS = new Set([
  'password',
  'password1',
  'password123',
  'passw0rd',
  '12345678',
  '123456789',
  '1234567890',
  '87654321',
  'qwerty123',
  'qwertyuiop',
  '11111111',
  'abc12345',
  'admin123',
  'admin1234',
  'administrator',
  'iloveyou',
  'letmein1',
  'welcome1',
  'changeme',
  'devopslab'
])

function byteLength(value: string): number {
  return new TextEncoder().encode(value).length
}

/** First violated backend rule as Uzbek text, or null when the password is acceptable. */
export function passwordRuleError(password: string, username: string, current = ''): string | null {
  const chars = Array.from(password)
  if (chars.length < PASSWORD_MIN) return `Parol kamida ${PASSWORD_MIN} belgidan iborat bo'lishi kerak.`
  if (byteLength(password) > PASSWORD_MAX_BYTES) return `Parol ${PASSWORD_MAX_BYTES} baytdan oshmasligi kerak.`
  if (chars.every((c) => c === chars[0])) return 'Parol bitta takrorlanuvchi belgidan iborat bo\'lmasin.'
  if (username && password.toLowerCase() === username.toLowerCase()) {
    return 'Parol foydalanuvchi nomi bilan bir xil bo\'lmasin.'
  }
  if (COMMON_PASSWORDS.has(password.toLowerCase())) return 'Bu parol juda keng tarqalgan — boshqasini tanlang.'
  if (current && password === current) return 'Yangi parol joriy paroldan farq qilishi kerak.'
  return null
}

/** Maps a backend "Weak password" message (English) to Uzbek. */
export function passwordServerError(message: string): string {
  const m = message.toLowerCase()
  if (m.includes('at least')) return `Parol kamida ${PASSWORD_MIN} belgidan iborat bo'lishi kerak.`
  if (m.includes('at most')) return `Parol ${PASSWORD_MAX_BYTES} baytdan oshmasligi kerak.`
  if (m.includes('username')) return 'Parol foydalanuvchi nomi bilan bir xil bo\'lmasin.'
  if (m.includes('differ')) return 'Yangi parol joriy paroldan farq qilishi kerak.'
  if (m.includes('weak')) return 'Parol juda oddiy — uzunroq va turli belgilardan foydalaning.'
  return message
}

export interface PasswordStrength {
  /** 0 = empty, 1..4 = weak..strong. */
  score: 0 | 1 | 2 | 3 | 4
  label: string
  color: string
}

const STRENGTH: Record<1 | 2 | 3 | 4, { label: string; color: string }> = {
  1: { label: 'Zaif', color: '#F87171' },
  2: { label: "O'rtacha", color: '#F5B342' },
  3: { label: 'Yaxshi', color: '#38BDF8' },
  4: { label: 'Kuchli', color: '#3DDC97' }
}

/** Heuristic meter: rule violations are always "Zaif"; then length and character variety. */
export function passwordStrength(password: string, username: string): PasswordStrength {
  if (!password) return { score: 0, label: '', color: '#243247' }
  if (passwordRuleError(password, username)) return { score: 1, ...STRENGTH[1] }
  const len = Array.from(password).length
  const classes = [/[a-z]/, /[A-Z]/, /[0-9]/, /[^A-Za-z0-9]/].filter((re) => re.test(password)).length
  let points = 0
  if (len >= 12) points += 1
  if (len >= 16) points += 1
  if (classes >= 2) points += 1
  if (classes >= 3) points += 1
  const score = Math.min(4, points + 1) as 1 | 2 | 3 | 4
  return { score, ...STRENGTH[score] }
}
