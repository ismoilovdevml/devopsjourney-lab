import { createRouter, createWebHistory, type RouteLocationRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

declare module 'vue-router' {
  interface RouteMeta {
    /** Page title; rendered as "<title> · DevOps Journey Lab". */
    title?: string
    /** Default true. Set false for public pages. */
    requiresAuth?: boolean
    requiresAdmin?: boolean
  }
}

const APP_NAME = 'DevOps Journey Lab'

const router = createRouter({
  history: createWebHistory(),
  scrollBehavior(to, _from, saved) {
    if (saved) return saved
    if (to.hash) return { el: to.hash }
    return { top: 0 }
  },
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { title: 'Kirish', requiresAuth: false }
    },
    {
      path: '/',
      name: 'dashboard',
      component: () => import('@/views/DashboardView.vue'),
      meta: { title: 'Bosh sahifa' }
    },
    {
      path: '/tasks',
      name: 'tasks',
      component: () => import('@/views/TasksView.vue'),
      meta: { title: 'Vazifalar' }
    },
    {
      path: '/lab/:taskId?',
      name: 'lab',
      component: () => import('@/views/LabView.vue'),
      meta: { title: 'Laboratoriya' }
    },
    {
      path: '/progress',
      name: 'progress',
      component: () => import('@/views/ProgressView.vue'),
      meta: { title: 'Natijalar' }
    },
    {
      path: '/leaderboard',
      name: 'leaderboard',
      component: () => import('@/views/LeaderboardView.vue'),
      meta: { title: 'Reyting' }
    },
    {
      path: '/profile',
      name: 'profile',
      component: () => import('@/views/ProfileView.vue'),
      meta: { title: 'Profil' }
    },
    {
      path: '/admin',
      name: 'admin',
      component: () => import('@/views/AdminView.vue'),
      meta: { title: 'Admin', requiresAdmin: true }
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('@/views/NotFoundView.vue'),
      meta: { title: 'Topilmadi', requiresAuth: false }
    }
  ]
})

/** Only same-app absolute paths are accepted as post-login redirects (no open redirect). */
export function safeRedirect(value: unknown): string | null {
  if (typeof value !== 'string') return null
  if (!value.startsWith('/') || value.startsWith('//') || value.startsWith('/\\')) return null
  if (value.startsWith('/login')) return null
  return value
}

router.beforeEach(async (to): Promise<RouteLocationRaw | true> => {
  const auth = useAuthStore()
  await auth.init()

  if (to.name === 'login' && auth.isAuthenticated) {
    return safeRedirect(to.query.redirect) ?? { name: 'dashboard' }
  }

  const requiresAuth = to.meta.requiresAuth !== false
  if (requiresAuth && !auth.isAuthenticated) {
    return to.fullPath === '/' ? { name: 'login' } : { name: 'login', query: { redirect: to.fullPath } }
  }

  if (to.meta.requiresAdmin && !auth.isAdmin) {
    return { name: 'dashboard' }
  }

  return true
})

router.afterEach((to) => {
  document.title = to.meta.title ? `${to.meta.title} · ${APP_NAME}` : APP_NAME
})

export default router
