export interface User {
  id: number
  username: string
  email: string
  role: string
  active: boolean
  created_at: string
}

export interface Task {
  id: string
  title: string
  description: string
  image?: string
  order: number
  difficulty: number
  points: number
  hints?: string[]
  unlocked?: boolean
  completed?: boolean
  setup?: string[]
  checks?: Check[]
}

export interface Check {
  name: string
  cmd: string
}

export interface Session {
  id: string
  user_id: number
  task_id: string
  container_id: string
  status: string
  created_at: string
  expires_at: string
}

export interface SessionResponse {
  sessionId: string
  ttlSeconds: number
  expiresAt: string
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

export interface Progress {
  id?: number
  user_id?: number
  task_id: string
  score: number
  total: number
  passed: boolean
  attempts?: number
  time_spent?: number
  completed_at: string
}

export interface ProgressResponse {
  progress: Progress[]
  totalScore: number
  totalPossible: number
  percentage: number
}

export interface LeaderboardEntry {
  user_id: number
  username: string
  score: number
  completed: number
  rank: number
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

export interface AuthResponse {
  token: string
  user: User
}
