<!--
  AdminView — /admin (admins only; enforced by the router guard and the backend).
  Own chrome per design/Admin.dc.html: AdminHeader + side nav (horizontal tabs on narrow screens).
  Section is kept in the URL: ?section=overview|users|sessions|activity|resources|tasks.
  GET /api/admin/stats is loaded here (overview + nav counts); every other section loads its own data.
-->
<template>
  <div class="flex min-h-screen flex-col">
    <a href="#main" class="sr-only-focusable fixed left-4 top-2 z-50 rounded-ctl bg-accent px-3 py-2 font-medium text-accent-ink">
      Asosiy kontentga o'tish
    </a>
    <AdminHeader />

    <div class="flex flex-1 flex-col lg:flex-row lg:items-start">
      <nav
        ref="navRef"
        aria-label="Admin bo'limlari"
        class="relative mx-4 mt-4 flex gap-1 overflow-x-auto rounded-card border border-border bg-surface p-1.5 sm:mx-6 lg:sticky lg:top-[84px] lg:my-6 lg:ml-6 lg:mr-0 lg:w-[250px] lg:shrink-0 lg:flex-col lg:overflow-visible lg:p-2"
      >
        <RouterLink
          v-for="item in navItems"
          :key="item.key"
          :to="{ name: 'admin', query: item.key === 'overview' ? {} : { section: item.key } }"
          :aria-current="item.key === section ? 'page' : undefined"
          :class="[
            'flex h-10 shrink-0 items-center justify-between gap-3 whitespace-nowrap rounded-lg px-3 font-medium transition-colors',
            item.key === section ? 'bg-raised text-text shadow-[inset_2px_0_0_#3DDC97]' : 'text-muted hover:bg-surface-3 hover:text-text'
          ]"
        >
          <span class="inline-flex items-center gap-2.5">
            <component :is="item.icon" class="h-4 w-4 shrink-0" aria-hidden="true" />
            {{ item.label }}
          </span>
          <span v-if="item.count !== null" class="font-mono text-xs text-muted">
            <span class="sr-only">(</span>{{ item.count }}<span class="sr-only">)</span>
          </span>
        </RouterLink>
      </nav>

      <main id="main" tabindex="-1" class="flex min-w-0 flex-1 flex-col gap-5 px-4 pb-16 pt-5 outline-none sm:px-6 lg:pt-6">
        <div class="flex flex-wrap items-end justify-between gap-4">
          <div class="flex min-w-0 flex-col gap-1">
            <span class="cmd-line">{{ current.command }}</span>
            <h1 class="font-mono text-[26px] font-bold leading-tight tracking-[-0.02em] text-text">{{ current.title }}</h1>
          </div>
          <Button size="sm" variant="ghost" :icon="RefreshCw" :loading="refreshing" @click="refresh">Yangilash</Button>
        </div>

        <AdminOverview v-if="section === 'overview'" :stats="stats" :loading="statsLoading" :error="statsError" @retry="loadStats" />
        <AdminUsers v-else-if="section === 'users'" ref="sectionRef" @count="(n) => (counts.users = n)" @changed="loadStats" />
        <AdminSessions v-else-if="section === 'sessions'" ref="sectionRef" @count="(n) => (counts.sessions = n)" @changed="loadStats" />
        <AdminActivity v-else-if="section === 'activity'" ref="sectionRef" />
        <AdminResources v-else-if="section === 'resources'" ref="sectionRef" />
        <AdminTasks v-else ref="sectionRef" @changed="loadStats" />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref, watch, type Component } from 'vue'
import { useRoute } from 'vue-router'
import { Activity, BookOpen, Cpu, LayoutDashboard, RefreshCw, TerminalSquare, Users } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import AdminHeader from '@/components/admin/AdminHeader.vue'
import AdminOverview from '@/components/admin/AdminOverview.vue'
import AdminUsers from '@/components/admin/AdminUsers.vue'
import AdminSessions from '@/components/admin/AdminSessions.vue'
import AdminActivity from '@/components/admin/AdminActivity.vue'
import AdminResources from '@/components/admin/AdminResources.vue'
import AdminTasks from '@/components/admin/AdminTasks.vue'
import { api, errorMessage } from '@/services/api'
import { useTaskStore } from '@/stores/task'
import type { AdminStats } from '@/types'

type Section = 'overview' | 'users' | 'sessions' | 'activity' | 'resources' | 'tasks'

interface SectionMeta {
  label: string
  title: string
  command: string
  icon: Component
}

/** Public surface every section component exposes via defineExpose. */
interface SectionHandle {
  refresh: () => Promise<void>
  loading: boolean
}

const SECTIONS: Record<Section, SectionMeta> = {
  overview: { label: 'Umumiy', title: "Umumiy ko'rinish", command: '$ sudo systemctl status lab', icon: LayoutDashboard },
  users: { label: 'Foydalanuvchilar', title: 'Foydalanuvchilar', command: '$ getent passwd', icon: Users },
  sessions: { label: 'Sessiyalar', title: 'Faol sessiyalar', command: '$ docker ps --filter label=devops-lab', icon: TerminalSquare },
  activity: { label: 'Faollik jurnali', title: 'Faollik jurnali', command: '$ journalctl -u devops-lab', icon: Activity },
  resources: { label: 'Resurslar', title: 'Server resurslari', command: '$ docker stats --no-stream', icon: Cpu },
  tasks: { label: 'Vazifalar', title: 'Vazifalar', command: '$ ls backend-go/tasks', icon: BookOpen }
}
const ORDER = Object.keys(SECTIONS) as Section[]

const route = useRoute()
const tasks = useTaskStore()

const section = computed<Section>(() => {
  const q = route.query.section
  return typeof q === 'string' && (ORDER as string[]).includes(q) ? (q as Section) : 'overview'
})
const current = computed(() => SECTIONS[section.value])

const stats = ref<AdminStats | null>(null)
const statsLoading = ref(false)
const statsError = ref<string | null>(null)
/** Live counts reported by sections; they take precedence over the stats snapshot. */
const counts = reactive<{ users: number | null; sessions: number | null }>({ users: null, sessions: null })
const sectionRef = ref<SectionHandle | null>(null)

const navItems = computed(() =>
  ORDER.map((key) => {
    let count: number | null = null
    if (key === 'users') count = counts.users ?? stats.value?.total_users ?? null
    if (key === 'sessions') count = counts.sessions ?? stats.value?.active_sessions ?? null
    if (key === 'tasks') count = tasks.loaded ? tasks.tasks.length : null
    return { key, ...SECTIONS[key], count }
  })
)

async function loadStats(): Promise<void> {
  statsLoading.value = true
  statsError.value = null
  try {
    stats.value = await api.adminStats()
  } catch (err) {
    statsError.value = errorMessage(err, "Statistikani yuklab bo'lmadi.")
  } finally {
    statsLoading.value = false
  }
}

const refreshing = computed(() => (section.value === 'overview' ? statsLoading.value : sectionRef.value?.loading ?? false))

async function refresh() {
  if (section.value === 'overview') {
    await Promise.all([loadStats(), tasks.loadTasks(true)])
  } else {
    await sectionRef.value?.refresh()
  }
}

const navRef = ref<HTMLElement | null>(null)

/** On narrow screens the nav is a horizontal scroller: keep the active tab visible. */
async function revealActiveTab() {
  await nextTick()
  const nav = navRef.value
  const active = nav?.querySelector<HTMLElement>('[aria-current="page"]')
  if (!nav || !active || nav.scrollWidth <= nav.clientWidth) return
  const left = active.offsetLeft - (nav.clientWidth - active.offsetWidth) / 2
  nav.scrollLeft = Math.max(0, left)
}

watch(section, () => void revealActiveTab())

onMounted(() => {
  void loadStats()
  void tasks.loadTasks()
  void revealActiveTab()
})
</script>
