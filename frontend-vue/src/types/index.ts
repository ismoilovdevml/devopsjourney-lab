// Types mirror docs/API.md exactly. Keep this file in sync with the contract.

// ---------- Common ----------

/** Error body returned by every failing endpoint. */
export interface ApiErrorBody {
  error: string
  message?: string
  code: number
}

// ---------- Auth / users ----------

export type Role = 'student' | 'admin'

export interface User {
  id: number
  username: string
  email: string
  role: Role
  active: boolean
  github: string
  goal: string
  created_at: string
}

export interface LoginRequest {
  username: string
  password: string
}

export interface RegisterRequest {
  username: string
  email: string
  password: string
}

/** Response of POST /api/auth/login and /api/auth/register (JWT is set as an httpOnly cookie). */
export interface AuthResponse {
  user: User
}

export interface UpdateMeRequest {
  email?: string
  github?: string
  goal?: string
}

export interface ChangePasswordRequest {
  current_password: string
  new_password: string
}

// ---------- Tasks ----------

export type Difficulty = 1 | 2 | 3

/** Known categories from task.yaml; unknown strings are still accepted. */
export type TaskCategory = 'web' | 'containers' | 'services' | 'users' | 'network' | 'automation'

export interface TaskSummary {
  id: string
  title: string
  description: string
  category: TaskCategory | string
  difficulty: Difficulty
  points: number
  order: number
  checks_count: number
  unlocked: boolean
  completed: boolean
  in_progress: boolean
  /** False when the server cannot run this task safely (e.g. NET_ADMIN without a sandbox runtime). */
  available: boolean
  unavailable_reason?: string
}

export interface TaskCheck {
  name: string
}

export interface TaskDetail extends TaskSummary {
  image: string
  /** Markdown. */
  readme: string
  hints: string[]
  checks: TaskCheck[]
}

// ---------- Sessions ----------

export interface SessionResponse {
  sessionId: string
  taskId: string
  ttlSeconds: number
  /** ISO timestamp. */
  expiresAt: string
  /** ISO timestamp. */
  createdAt: string
}

export interface CheckResult {
  name: string
  ok: boolean
  output?: string
}

export interface CheckResponse {
  passed: boolean
  score: number
  total: number
  results: CheckResult[]
}

/** Client -> server WebSocket control frame. Raw text frames are stdin. */
export interface TerminalResizeFrame {
  type: 'resize'
  cols: number
  rows: number
}

// ---------- Progress & leaderboard ----------

export interface ProgressEntry {
  task_id: string
  score: number
  total: number
  passed: boolean
  attempts: number
  /** Seconds. */
  time_spent: number
  /** ISO timestamp, null/empty when not completed. */
  completed_at: string | null
}

export interface SkillProgress {
  category: TaskCategory | string
  percent: number
}

export interface Achievement {
  id: string
  title: string
  description: string
  earned: boolean
  earned_at?: string
}

export interface ProgressResponse {
  progress: ProgressEntry[]
  totalScore: number
  totalPossible: number
  percentage: number
  completed: number
  totalTasks: number
  timeInLabSeconds: number
  streakDays: number
  skills: SkillProgress[]
  achievements: Achievement[]
}

export type LeaderboardPeriod = 'week' | 'month' | 'all'

export interface LeaderboardEntry {
  user_id: number
  username: string
  score: number
  completed: number
  rank: number
}

export interface LeaderboardResponse {
  entries: LeaderboardEntry[]
  me: LeaderboardEntry | null
}

// ---------- Admin ----------

export interface TaskCompletionStat {
  task_id: string
  attempts: number
  passed: number
}

export interface AdminStats {
  total_users: number
  active_users: number
  active_sessions: number
  total_attempts: number
  completion_rate: number
  /** 24 hourly buckets. */
  sessions_24h: number[]
  task_completion: TaskCompletionStat[]
}

export interface AdminUser extends User {
  score: number
  completed: number
  last_active: string | null
}

export interface AdminUserUpdate {
  role?: Role
  active?: boolean
}

export interface AdminSession {
  id: string
  user_id: number
  username: string
  task_id: string
  status: string
  created_at: string
  expires_at: string
}

/** Activity log level; backend emits INFO | WARN | ERROR (compare case-insensitively). */
export type ActivityLevel = 'INFO' | 'WARN' | 'ERROR'

export interface AdminActivity {
  id: number
  username: string
  action: string
  details: string
  ip_address: string
  level: ActivityLevel | string
  created_at: string
}

export interface AdminSessionResource {
  session_id: string
  username: string
  task_id: string
  cpu_percent: number
  mem_bytes: number
  mem_limit_bytes: number
}

export interface AdminResources {
  containers_running: number
  containers_limit: number
  sessions: AdminSessionResource[]
}

export interface ReloadTasksResponse {
  count: number
}

// ---------- Health ----------

export interface HealthResponse {
  status: string
  timestamp: string
  version: string
  uptime: string | number
}
