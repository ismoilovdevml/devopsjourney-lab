<!--
  AppHeader — sticky top bar: logo, main nav (aria-current on the active item), search button
  (Ctrl/Cmd+K -> /tasks?focus=search), live session pill (mm:ss, links to the lab; only when a
  session is active), admin link (admins only), avatar (-> /profile) and logout.
  Props: none (reads auth + session stores and the current route).
-->
<template>
  <header class="sticky top-0 z-40 border-b border-border bg-[rgba(10,14,20,0.82)] backdrop-blur-md">
    <div class="container-page flex min-h-[60px] flex-wrap items-center gap-x-5 gap-y-2 py-2.5 xl:gap-x-7">
      <AppLogo />

      <nav aria-label="Asosiy menyu" class="-mx-1 flex max-w-full gap-1 overflow-x-auto px-1 order-last w-full lg:order-none lg:w-auto">
        <RouterLink
          v-for="item in NAV"
          :key="item.name"
          :to="navTo(item)"
          :aria-current="isActive(item) ? 'page' : undefined"
          :class="[
            'flex shrink-0 items-center gap-2 rounded-lg px-3 py-2 font-medium transition-colors',
            isActive(item) ? 'bg-surface-3 text-text shadow-ring' : 'text-muted hover:text-text'
          ]"
        >
          <span v-if="isActive(item)" class="h-1.5 w-1.5 rounded-full bg-accent" aria-hidden="true" />
          {{ item.label }}
        </RouterLink>
      </nav>

      <div class="ml-auto flex items-center gap-2.5">
        <button
          type="button"
          class="hidden h-[38px] items-center gap-2 rounded-lg border border-border-strong bg-surface-solid pl-3 pr-2.5 text-muted transition-colors hover:border-border-stronger hover:text-text md:flex xl:min-w-[210px]"
          :aria-label="isMac ? 'Qidirish ⌘K' : 'Qidirish Ctrl K'"
          aria-keyshortcuts="Control+K Meta+K"
          @click="openSearch"
        >
          <Search class="h-[15px] w-[15px]" aria-hidden="true" />
          <span class="hidden flex-1 text-left xl:inline" aria-hidden="true">Qidirish…</span>
          <kbd class="hidden rounded border border-border-strong px-1.5 font-mono text-[11px] text-muted xl:inline">{{ isMac ? '⌘K' : 'Ctrl K' }}</kbd>
        </button>

        <RouterLink
          v-if="session.isActive && session.taskId"
          :to="{ name: 'lab', params: { taskId: session.taskId } }"
          class="inline-flex h-[38px] items-center gap-2 rounded-lg border border-accent/35 bg-accent/10 px-3 font-mono text-[13px] text-accent hover:text-accent"
          :aria-label="`Faol sessiya, ${session.remainingLabel} qoldi. Laboratoriyaga o'tish`"
        >
          <span class="h-[7px] w-[7px] rounded-full bg-accent shadow-glow-sm" aria-hidden="true" />
          <span aria-hidden="true">{{ session.remainingLabel }}</span>
        </RouterLink>

        <RouterLink
          v-if="auth.isAdmin"
          :to="{ name: 'admin' }"
          class="inline-flex h-[38px] items-center gap-1.5 rounded-lg border border-warn/35 bg-warn/10 px-3 font-mono text-[13px] text-warn hover:text-warn"
        >
          <ShieldCheck class="h-4 w-4" aria-hidden="true" />
          admin
        </RouterLink>

        <RouterLink
          :to="{ name: 'profile' }"
          :aria-current="route.name === 'profile' ? 'page' : undefined"
          class="flex h-[38px] w-[38px] items-center justify-center rounded-full border-[1.5px] border-accent bg-surface-3 text-[13px] font-semibold text-text hover:text-text"
          :aria-label="`${auth.initials || '?'} · Profil: ${auth.user?.username ?? ''}`"
        >
          {{ auth.initials || '?' }}
        </RouterLink>

        <button
          type="button"
          class="flex h-[38px] w-[38px] items-center justify-center rounded-lg text-muted transition-colors hover:bg-surface-3 hover:text-text"
          aria-label="Chiqish"
          title="Chiqish"
          @click="logout"
        >
          <LogOut class="h-4 w-4" aria-hidden="true" />
        </button>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted } from 'vue'
import { useRoute, useRouter, type RouteLocationRaw } from 'vue-router'
import { LogOut, Search, ShieldCheck } from 'lucide-vue-next'
import AppLogo from './AppLogo.vue'
import { useAuthStore } from '@/stores/auth'
import { useSessionStore } from '@/stores/session'

interface NavItem {
  name: string
  label: string
  to: RouteLocationRaw
}

const NAV: NavItem[] = [
  { name: 'dashboard', label: 'Bosh sahifa', to: { name: 'dashboard' } },
  { name: 'tasks', label: 'Vazifalar', to: { name: 'tasks' } },
  { name: 'lab', label: 'Laboratoriya', to: { name: 'lab' } },
  { name: 'progress', label: 'Natijalar', to: { name: 'progress' } },
  { name: 'leaderboard', label: 'Reyting', to: { name: 'leaderboard' } }
]

const auth = useAuthStore()
const session = useSessionStore()
const route = useRoute()
const router = useRouter()

const isMac = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.platform)

function isActive(item: NavItem): boolean {
  return route.name === item.name
}

/** The lab link resumes the active session's task when there is one. */
function navTo(item: NavItem): RouteLocationRaw {
  if (item.name === 'lab' && session.taskId) return { name: 'lab', params: { taskId: session.taskId } }
  return item.to
}

function openSearch() {
  router.push({ name: 'tasks', query: { focus: 'search' } })
}

function onKey(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    openSearch()
  }
}

async function logout() {
  await auth.logout()
  router.replace({ name: 'login' })
}

onMounted(() => window.addEventListener('keydown', onKey))
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))
</script>
