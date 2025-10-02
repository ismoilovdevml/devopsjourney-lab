import axios, { type AxiosInstance, type InternalAxiosRequestConfig } from 'axios'
import type {
  Task,
  SessionResponse,
  CheckResponse,
  LoginRequest,
  RegisterRequest,
  AuthResponse,
  ProgressResponse,
  LeaderboardEntry
} from '@/types'

class ApiService {
  private client: AxiosInstance
  private retryConfig = {
    maxRetries: 3,
    retryDelay: 1000,
    retryableStatuses: [408, 429, 500, 502, 503, 504]
  }

  constructor() {
    // Use relative path /api for production (works with nginx proxy)
    // For development, vite.config.ts proxy will handle it
    this.client = axios.create({
      baseURL: import.meta.env.VITE_API_URL || '/api',
      timeout: 30000,
      headers: {
        'Content-Type': 'application/json'
      }
    })

    this.setupInterceptors()
  }

  private setupInterceptors() {
    // Request interceptor to add auth token and username
    this.client.interceptors.request.use(
      (config: InternalAxiosRequestConfig) => {
        const token = localStorage.getItem('auth_token')
        const username = localStorage.getItem('username')

        if (token && config.headers) {
          config.headers.Authorization = `Bearer ${token}`
        }

        if (username && config.headers) {
          config.headers['x-username'] = username
        }

        return config
      },
      (error) => Promise.reject(error)
    )

    // Response interceptor with retry logic and error handling
    this.client.interceptors.response.use(
      (response) => response,
      async (error) => {
        const config = error.config as any

        // Initialize retry count
        if (!config._retryCount) {
          config._retryCount = 0
        }

        // Check if we should retry
        const shouldRetry =
          config._retryCount < this.retryConfig.maxRetries &&
          (this.retryConfig.retryableStatuses.includes(error.response?.status) ||
           error.code === 'ECONNABORTED' ||
           error.message === 'Network Error')

        if (shouldRetry) {
          config._retryCount++

          // Exponential backoff
          const delay = this.retryConfig.retryDelay * Math.pow(2, config._retryCount - 1)

          console.log(
            `Retrying request (${config._retryCount}/${this.retryConfig.maxRetries}) after ${delay}ms...`,
            config.url
          )

          await new Promise((resolve) => setTimeout(resolve, delay))
          return this.client(config)
        }

        // Handle 401 Unauthorized
        if (error.response?.status === 401) {
          this.handleUnauthorized()
        }

        return Promise.reject(error)
      }
    )
  }

  private handleUnauthorized() {
    console.warn('Unauthorized - clearing auth and redirecting to login')
    localStorage.removeItem('auth_token')
    localStorage.removeItem('user')

    // Only redirect if not already on login page
    if (window.location.pathname !== '/login') {
      window.location.href = '/login'
    }
  }

  // Auth
  async login(credentials: LoginRequest): Promise<AuthResponse> {
    console.log('API: Attempting login...', { username: credentials.username })
    try {
      const { data } = await this.client.post<AuthResponse>('/auth/login', credentials)
      console.log('API: Login successful', data)
      if (data.token) {
        localStorage.setItem('auth_token', data.token)
        if (data.user) {
          localStorage.setItem('user', JSON.stringify(data.user))
        }
      }
      return data
    } catch (error: any) {
      console.error('API: Login failed', error.response?.data || error.message)
      throw error
    }
  }

  async register(userData: RegisterRequest): Promise<AuthResponse> {
    console.log('API: Attempting registration...', { username: userData.username, email: userData.email })
    try {
      const { data } = await this.client.post<AuthResponse>('/auth/register', userData)
      console.log('API: Registration successful', data)
      if (data.token) {
        localStorage.setItem('auth_token', data.token)
        if (data.user) {
          localStorage.setItem('user', JSON.stringify(data.user))
        }
      }
      return data
    } catch (error: any) {
      console.error('API: Registration failed', error.response?.data || error.message)
      throw error
    }
  }

  logout() {
    localStorage.removeItem('auth_token')
    localStorage.removeItem('user')
    localStorage.removeItem('session_id')
    localStorage.removeItem('time_remaining')
    window.location.href = '/login'
  }

  // Tasks
  async getTasks(): Promise<Task[]> {
    const { data } = await this.client.get<Task[]>('/tasks')
    return data
  }

  async getTask(id: string): Promise<Task> {
    const { data } = await this.client.get<Task>(`/tasks/${id}`)
    return data
  }

  // Sessions
  async createSession(taskId: string): Promise<SessionResponse> {
    const { data } = await this.client.post<SessionResponse>('/sessions', { taskId })
    return data
  }

  async deleteSession(sessionId: string): Promise<void> {
    await this.client.delete(`/sessions/${sessionId}`)
  }

  // Check solution
  async checkSolution(sessionId: string, taskId?: string): Promise<CheckResponse> {
    const { data } = await this.client.post<CheckResponse>(`/check/${sessionId}`, { taskId })
    return data
  }

  // Progress
  async getProgress(): Promise<ProgressResponse> {
    const { data } = await this.client.get<ProgressResponse>('/progress')
    return data
  }

  async getLeaderboard(): Promise<LeaderboardEntry[]> {
    const { data } = await this.client.get<LeaderboardEntry[]>('/leaderboard')
    return data
  }

  // Health
  async healthCheck(): Promise<{ status: string; active_timers?: number }> {
    const { data } = await this.client.get('/healthz')
    return data
  }
}

export const api = new ApiService()
export default api
