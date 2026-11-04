/**
 * HTTP client for the backend. Mirrors docs/API.md.
 *
 * - Auth is an httpOnly cookie (`dj_token`): `withCredentials: true`, no token is ever stored in JS.
 * - Every failure is normalized to `ApiError` (use `errorMessage(err)` for UI text).
 * - 401 calls the handler registered via `setUnauthorizedHandler` (clears auth + redirects to /login),
 *   except for requests sent with `skipAuthRedirect` (GET /me bootstrap, login, password change).
 * - Only idempotent GETs are retried, on 502/503/504 or network errors, at most 2 times.
 */
import axios, { AxiosError, type AxiosInstance, type InternalAxiosRequestConfig } from 'axios'
import type {
  AdminActivity,
  AdminResources,
  AdminSession,
  AdminStats,
  AdminUser,
  AdminUserUpdate,
  ApiErrorBody,
  AuthResponse,
  ChangePasswordRequest,
  CheckResponse,
  HealthResponse,
  LeaderboardPeriod,
  LeaderboardResponse,
  LoginRequest,
  ProgressResponse,
  RegisterRequest,
  ReloadTasksResponse,
  SessionResponse,
  TaskDetail,
  TaskSummary,
  UpdateMeRequest,
  User
} from '@/types'

declare module 'axios' {
  interface AxiosRequestConfig {
    /** Do not trigger the global 401 handler for this request. */
    skipAuthRedirect?: boolean
    /** Internal retry counter. */
    _retryCount?: number
  }
}

const MAX_RETRIES = 2
const RETRY_BASE_DELAY_MS = 400
const RETRYABLE_STATUSES = new Set([502, 503, 504])

/** Normalized API error. `status` is 0 for network errors / timeouts. */
export class ApiError extends Error {
  readonly status: number
  readonly code: string | null
  readonly body: ApiErrorBody | null

  constructor(message: string, status: number, code: string | null, body: ApiErrorBody | null) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.body = body
  }
}

function isErrorBody(value: unknown): value is ApiErrorBody {
  return typeof value === 'object' && value !== null && typeof (value as { error?: unknown }).error === 'string'
}

function defaultMessage(status: number): string {
  if (status === 0) return "Serverga ulanib bo'lmadi. Internet aloqasini tekshiring."
  if (status === 401) return 'Avtorizatsiya talab qilinadi.'
  if (status === 403) return "Bu amal uchun ruxsat yo'q."
  if (status === 404) return 'Topilmadi.'
  if (status === 409) return 'Ziddiyat: amalni bajarib bo\'lmadi.'
  if (status === 429) return "Juda ko'p urinish. Birozdan so'ng qayta urinib ko'ring."
  if (status >= 500) return 'Serverda xatolik yuz berdi. Keyinroq urinib ko\'ring.'
  return "So'rov bajarilmadi."
}

function toApiError(err: unknown): ApiError {
  if (err instanceof ApiError) return err
  if (axios.isAxiosError(err)) {
    const status = err.response?.status ?? 0
    const data: unknown = err.response?.data
    const body = isErrorBody(data) ? data : null
    const message = body?.message || body?.error || defaultMessage(status)
    return new ApiError(message, status, err.code ?? null, body)
  }
  if (err instanceof Error) return new ApiError(err.message, 0, null, null)
  return new ApiError(defaultMessage(0), 0, null, null)
}

/**
 * Human-readable message for any thrown value: the server's message if it sent one,
 * otherwise the network-error text (status 0), otherwise `fallback` or a generic Uzbek text.
 */
export function errorMessage(err: unknown, fallback?: string): string {
  const e = toApiError(err)
  if (e.body || e.status === 0 || !fallback) return e.message
  return fallback
}

/** True when `err` is an ApiError with the given HTTP status. */
export function isApiStatus(err: unknown, status: number): boolean {
  return err instanceof ApiError && err.status === status
}

let unauthorizedHandler: (() => void) | null = null

/** Registered once in main.ts: clears the auth store and redirects to /login. */
export function setUnauthorizedHandler(handler: () => void): void {
  unauthorizedHandler = handler
}

const sleep = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms))

function shouldRetry(error: AxiosError, config: InternalAxiosRequestConfig): boolean {
  if ((config.method ?? 'get').toLowerCase() !== 'get') return false
  if ((config._retryCount ?? 0) >= MAX_RETRIES) return false
  if (axios.isCancel(error) || error.code === AxiosError.ERR_CANCELED) return false
  if (!error.response) return true // network error / timeout
  return RETRYABLE_STATUSES.has(error.response.status)
}

function createClient(): AxiosInstance {
  const client = axios.create({
    baseURL: import.meta.env.VITE_API_URL || '/api',
    timeout: 30000,
    withCredentials: true,
    headers: { 'Content-Type': 'application/json' }
  })

  client.interceptors.response.use(
    (response) => response,
    async (error: unknown) => {
      if (!axios.isAxiosError(error) || !error.config) return Promise.reject(toApiError(error))
      const config = error.config

      if (shouldRetry(error, config)) {
        config._retryCount = (config._retryCount ?? 0) + 1
        await sleep(RETRY_BASE_DELAY_MS * 2 ** (config._retryCount - 1))
        return client.request(config)
      }

      // A disabled account answers 403 "Account disabled": end the session like a 401.
      const body = error.response?.data as { error?: unknown } | undefined
      const accountDisabled = error.response?.status === 403 && body?.error === 'Account disabled'
      if ((error.response?.status === 401 || accountDisabled) && !config.skipAuthRedirect) {
        unauthorizedHandler?.()
      }

      return Promise.reject(toApiError(error))
    }
  )

  return client
}

const http = createClient()
const enc = encodeURIComponent

export const api = {
  // ---------- Auth ----------
  async login(body: LoginRequest): Promise<User> {
    const { data } = await http.post<AuthResponse>('/auth/login', body, { skipAuthRedirect: true })
    return data.user
  },

  async register(body: RegisterRequest): Promise<User> {
    const { data } = await http.post<AuthResponse>('/auth/register', body, { skipAuthRedirect: true })
    return data.user
  },

  async logout(): Promise<void> {
    await http.post('/auth/logout', undefined, { skipAuthRedirect: true })
  },

  // ---------- Me ----------
  /** Bootstrap call: never triggers the global 401 redirect. */
  async getMe(): Promise<User> {
    const { data } = await http.get<User>('/me', { skipAuthRedirect: true, timeout: 10000 })
    return data
  },

  async updateMe(body: UpdateMeRequest): Promise<User> {
    const { data } = await http.put<User>('/me', body)
    return data
  },

  /** 400 = weak password, 401 = wrong current password (does not log out). */
  async changePassword(body: ChangePasswordRequest): Promise<void> {
    await http.post('/me/password', body, { skipAuthRedirect: true })
  },

  // ---------- Tasks ----------
  async getTasks(): Promise<TaskSummary[]> {
    const { data } = await http.get<TaskSummary[]>('/tasks')
    return data
  },

  async getTask(id: string): Promise<TaskDetail> {
    const { data } = await http.get<TaskDetail>(`/tasks/${enc(id)}`)
    return data
  },

  // ---------- Sessions ----------
  /** 409 if the task is locked. Starting a new session stops the previous one. */
  async createSession(taskId: string): Promise<SessionResponse> {
    const { data } = await http.post<SessionResponse>('/sessions', { taskId })
    return data
  },

  /** Returns null when there is no active session (204). */
  async getActiveSession(): Promise<SessionResponse | null> {
    const res = await http.get<SessionResponse | ''>('/sessions/active')
    if (res.status === 204 || !res.data) return null
    return res.data
  },

  async getSession(id: string): Promise<SessionResponse> {
    const { data } = await http.get<SessionResponse>(`/sessions/${enc(id)}`)
    return data
  },

  /** Replaces the container; the returned session has a new sessionId. */
  async recreateSession(id: string): Promise<SessionResponse> {
    const { data } = await http.post<SessionResponse>(`/sessions/${enc(id)}/recreate`)
    return data
  },

  async deleteSession(id: string): Promise<void> {
    await http.delete(`/sessions/${enc(id)}`)
  },

  async checkSolution(sessionId: string): Promise<CheckResponse> {
    const { data } = await http.post<CheckResponse>(`/check/${enc(sessionId)}`)
    return data
  },

  // ---------- Progress & leaderboard ----------
  async getProgress(): Promise<ProgressResponse> {
    const { data } = await http.get<ProgressResponse>('/progress')
    return data
  },

  async resetProgress(): Promise<void> {
    await http.delete('/progress')
  },

  async getLeaderboard(period: LeaderboardPeriod = 'all', limit = 50): Promise<LeaderboardResponse> {
    const { data } = await http.get<LeaderboardResponse>('/leaderboard', { params: { period, limit } })
    return data
  },

  // ---------- Admin ----------
  async adminStats(): Promise<AdminStats> {
    const { data } = await http.get<AdminStats>('/admin/stats')
    return data
  },

  /** Newest users first; `q` filters by username or email on the server. */
  async adminUsers(q = ''): Promise<AdminUser[]> {
    const { data } = await http.get<AdminUser[]>('/admin/users', { params: q ? { q } : undefined })
    return data
  },

  async adminUpdateUser(id: number, body: AdminUserUpdate): Promise<User> {
    const { data } = await http.put<User>(`/admin/users/${id}`, body)
    return data
  },

  async adminDeleteUser(id: number): Promise<void> {
    await http.delete(`/admin/users/${id}`)
  },

  async adminSessions(): Promise<AdminSession[]> {
    const { data } = await http.get<AdminSession[]>('/admin/sessions')
    return data
  },

  async adminTerminateSession(id: string): Promise<void> {
    await http.post(`/admin/sessions/${enc(id)}/terminate`)
  },

  async adminActivity(limit = 100): Promise<AdminActivity[]> {
    const { data } = await http.get<AdminActivity[]>('/admin/activity', { params: { limit } })
    return data
  },

  async adminResources(): Promise<AdminResources> {
    const { data } = await http.get<AdminResources>('/admin/resources')
    return data
  },

  async adminReloadTasks(): Promise<ReloadTasksResponse> {
    const { data } = await http.post<ReloadTasksResponse>('/admin/tasks/reload')
    return data
  },

  // ---------- Health (outside /api) ----------
  async health(): Promise<HealthResponse> {
    const { data } = await http.get<HealthResponse>('/healthz', { baseURL: '/', skipAuthRedirect: true })
    if (typeof data !== 'object' || data === null || typeof data.status !== 'string') {
      throw new ApiError('Invalid health response', 0, null, null)
    }
    return data
  }
}

export type Api = typeof api
export default api
