/**
 * Client-side validation for the login / register form. Mirrors the backend rules
 * (backend-go/internal/auth/password.go, models.RegisterRequest) so most mistakes are caught
 * before a rate-limited request is spent. The backend stays the source of truth.
 */
import { ApiError } from '@/services/api'

export type AuthMode = 'login' | 'register'
export type AuthField = 'username' | 'email' | 'password'
export type FieldErrors = Partial<Record<AuthField, string>>

export interface AuthFormValues {
  username: string
  email: string
  password: string
}

const USERNAME_RE = /^[a-z0-9_-]{3,32}$/
// Pragmatic email check: one "@", no spaces, a dot in the domain part.
const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
const PASSWORD_MIN_CHARS = 8
const PASSWORD_MAX_BYTES = 72
const EMAIL_MAX = 254

/** Usernames are stored lowercase; trim and lowercase whatever the user typed. */
export function normalizeUsername(value: string): string {
  return value.trim().toLowerCase()
}

export function normalizeEmail(value: string): string {
  return value.trim().toLowerCase()
}

function byteLength(value: string): number {
  return new TextEncoder().encode(value).length
}

function charLength(value: string): number {
  return Array.from(value).length
}

export function validateUsername(raw: string, mode: AuthMode): string | undefined {
  const value = normalizeUsername(raw)
  if (!value) return 'Foydalanuvchi nomini kiriting'
  // Login only needs a value: the backend answers 401 for anything that does not exist.
  if (mode === 'login') return value.length > 64 ? 'Foydalanuvchi nomi juda uzun' : undefined
  if (value.length < 3 || value.length > 32) return "3–32 ta belgi bo'lishi kerak"
  if (!USERNAME_RE.test(value)) return "Faqat kichik lotin harflari, raqamlar, “_” va “-” ishlatiladi"
  return undefined
}

export function validateEmail(raw: string): string | undefined {
  const value = normalizeEmail(raw)
  if (!value) return 'Email manzilini kiriting'
  if (value.length > EMAIL_MAX || !EMAIL_RE.test(value)) return "Email manzili noto'g'ri"
  return undefined
}

export function validatePassword(value: string, mode: AuthMode, username: string): string | undefined {
  if (!value) return 'Parolni kiriting'
  if (byteLength(value) > PASSWORD_MAX_BYTES) return 'Parol juda uzun (ko‘pi bilan 72 bayt)'
  if (mode === 'login') return undefined
  if (charLength(value) < PASSWORD_MIN_CHARS) return `Kamida ${PASSWORD_MIN_CHARS} ta belgi bo'lishi kerak`
  const first = Array.from(value)[0]
  if (Array.from(value).every((ch) => ch === first)) return 'Parol bir xil belgidan iborat bo‘lmasin'
  const name = normalizeUsername(username)
  if (name && value.toLowerCase() === name) return 'Parol foydalanuvchi nomi bilan bir xil bo‘lmasin'
  return undefined
}

export function validateAuthForm(values: AuthFormValues, mode: AuthMode): FieldErrors {
  const errors: FieldErrors = {}
  const username = validateUsername(values.username, mode)
  if (username) errors.username = username
  if (mode === 'register') {
    const email = validateEmail(values.email)
    if (email) errors.email = email
  }
  const password = validatePassword(values.password, mode, values.username)
  if (password) errors.password = password
  return errors
}

/** Validated, normalized values emitted by AuthCard. */
export interface AuthSubmitPayload extends AuthFormValues {
  mode: AuthMode
}

export interface AuthErrorView {
  /** Form-level message shown above the submit button. */
  form?: string
  /** Message that belongs to a single field. */
  fields?: FieldErrors
}

/** Translate a failed login/register request into Uzbek UI messages. */
export function mapAuthError(err: unknown, mode: AuthMode): AuthErrorView {
  const status = err instanceof ApiError ? err.status : 0
  const kind = err instanceof ApiError ? err.body?.error ?? '' : ''

  if (status === 401) return { form: "Login yoki parol noto'g'ri" }
  if (status === 409) return { form: 'Bu foydalanuvchi nomi yoki email band' }
  if (status === 429) return { form: "Juda ko'p urinish, bir daqiqadan so'ng qayta urinib ko'ring" }
  if (status === 403) {
    if (kind === 'Account disabled') return { form: 'Hisobingiz faol emas. Administratorga murojaat qiling.' }
    return { form: "So'rov rad etildi. Sahifani yangilab, qayta urinib ko'ring." }
  }
  if (status === 400) {
    if (kind === 'Invalid username') {
      return { fields: { username: "Faqat kichik lotin harflari, raqamlar, “_” va “-” (3–32 ta belgi)" } }
    }
    if (kind === 'Weak password') {
      return { fields: { password: 'Parol juda oddiy — boshqa, murakkabroq parol tanlang' } }
    }
    return {
      form:
        mode === 'register'
          ? "Ma'lumotlar noto'g'ri. Maydonlarni tekshirib, qayta urinib ko'ring."
          : "Login yoki parol noto'g'ri"
    }
  }
  if (status === 0) return { form: "Serverga ulanib bo'lmadi. Internet aloqasini tekshiring." }
  return { form: "Serverda xatolik yuz berdi. Birozdan so'ng qayta urinib ko'ring." }
}
