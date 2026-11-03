<!--
  AdminHeader — admin chrome: logo ("devops_journey/admin"), "sudo" badge, live system status,
  link back to the student app and the profile avatar.
  System status: GET /healthz (liveness) + GET /readyz (readiness), re-checked every 60s while the
  tab is visible. "all systems operational" is shown ONLY when /readyz answers 200 {status:"ready"}.
  Props: none.
-->
<template>
  <header class="sticky top-0 z-40 border-b border-border bg-[rgba(10,14,20,0.88)] backdrop-blur-md">
    <div class="flex min-h-[60px] flex-wrap items-center gap-x-4 gap-y-2 px-4 py-2.5 sm:px-6">
      <AppLogo suffix="/admin" />
      <Badge color="#F5B342" class="!rounded-chip font-mono">sudo</Badge>

      <div class="ml-auto flex items-center gap-2.5">
        <span
          class="hidden items-center gap-2 font-mono text-[12.5px] sm:inline-flex"
          :class="STATUS_TEXT[status.kind]"
          role="status"
          :title="status.detail"
        >
          <span class="h-[7px] w-[7px] shrink-0 rounded-full" :class="STATUS_DOT[status.kind]" aria-hidden="true" />
          {{ status.label }}
        </span>
        <span
          class="inline-flex h-[38px] w-[38px] items-center justify-center rounded-lg border border-border-strong sm:hidden"
          :title="status.label"
        >
          <span class="h-2 w-2 rounded-full" :class="STATUS_DOT[status.kind]" aria-hidden="true" />
          <span class="sr-only">Tizim holati: {{ status.label }}</span>
        </span>

        <RouterLink
          :to="{ name: 'dashboard' }"
          class="inline-flex h-[38px] items-center gap-1.5 rounded-lg border border-border-strong px-3 font-medium text-text-2 transition-colors hover:border-border-stronger hover:text-text"
        >
          <ArrowLeft class="h-4 w-4 sm:hidden" aria-hidden="true" />
          <span class="hidden sm:inline">O'quvchi ko'rinishi</span>
          <span class="sm:hidden">Ilova</span>
        </RouterLink>

        <RouterLink
          :to="{ name: 'profile' }"
          class="flex h-[38px] w-[38px] items-center justify-center rounded-full border-[1.5px] border-warn bg-surface-3 text-[13px] font-bold text-warn hover:text-warn"
          :aria-label="`Profil: ${auth.user?.username ?? ''}`"
        >
          {{ auth.initials || '?' }}
        </RouterLink>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive } from 'vue'
import { ArrowLeft } from 'lucide-vue-next'
import AppLogo from '@/components/ui/AppLogo.vue'
import Badge from '@/components/ui/Badge.vue'
import { api } from '@/services/api'
import { useAuthStore } from '@/stores/auth'

type StatusKind = 'checking' | 'ready' | 'degraded' | 'unknown' | 'offline'

interface ReadyBody {
  status?: unknown
  checks?: unknown
}

const auth = useAuthStore()

const STATUS_TEXT: Record<StatusKind, string> = {
  checking: 'text-muted',
  ready: 'text-accent',
  degraded: 'text-warn',
  unknown: 'text-muted',
  offline: 'text-danger'
}
const STATUS_DOT: Record<StatusKind, string> = {
  checking: 'bg-faint',
  ready: 'bg-accent shadow-glow-sm',
  degraded: 'bg-warn',
  unknown: 'bg-info',
  offline: 'bg-danger'
}

const status = reactive<{ kind: StatusKind; label: string; detail: string }>({
  kind: 'checking',
  label: 'tekshirilmoqda…',
  detail: ''
})

const CHECK_INTERVAL_MS = 60_000
const READY_TIMEOUT_MS = 5_000
let timer: ReturnType<typeof setInterval> | null = null

/** Names of dependencies reported as not "ok" by /readyz. */
function downChecks(body: ReadyBody): string[] {
  if (typeof body.checks !== 'object' || body.checks === null) return []
  return Object.entries(body.checks as Record<string, unknown>)
    .filter(([, v]) => v !== 'ok')
    .map(([k]) => k)
}

/** null = readiness could not be determined (endpoint unreachable or not the backend). */
async function fetchReady(): Promise<{ ready: boolean; down: string[] } | null> {
  const ctrl = new AbortController()
  const t = setTimeout(() => ctrl.abort(), READY_TIMEOUT_MS)
  try {
    const res = await fetch('/readyz', { credentials: 'same-origin', signal: ctrl.signal, headers: { Accept: 'application/json' } })
    if (!(res.headers.get('content-type') ?? '').includes('application/json')) return null
    const body = (await res.json()) as ReadyBody
    if (res.status === 200 && body.status === 'ready') return { ready: true, down: [] }
    if (res.status === 503) return { ready: false, down: downChecks(body) }
    return null
  } catch {
    return null
  } finally {
    clearTimeout(t)
  }
}

async function check() {
  let version = ''
  try {
    const h = await api.health()
    version = h.version ? ` · v${h.version.replace(/^v/, '')}` : ''
  } catch {
    Object.assign(status, { kind: 'offline', label: 'api offline', detail: "GET /healthz javob bermadi" })
    return
  }
  const ready = await fetchReady()
  if (ready === null) {
    Object.assign(status, {
      kind: 'unknown',
      label: 'api online · readyz n/a',
      detail: `API ishlayapti${version}; /readyz holatini aniqlab bo'lmadi`
    })
  } else if (ready.ready) {
    Object.assign(status, { kind: 'ready', label: 'all systems operational', detail: `DB, Redis, Docker ishlayapti${version}` })
  } else {
    const down = ready.down.length ? ready.down.join(', ') : 'dependency'
    Object.assign(status, { kind: 'degraded', label: `degraded: ${down} down`, detail: `/readyz 503${version}` })
  }
}

function onVisibility() {
  if (document.visibilityState === 'visible') void check()
}

onMounted(() => {
  void check()
  timer = setInterval(() => {
    if (document.visibilityState === 'visible') void check()
  }, CHECK_INTERVAL_MS)
  document.addEventListener('visibilitychange', onVisibility)
})

onBeforeUnmount(() => {
  if (timer) clearInterval(timer)
  document.removeEventListener('visibilitychange', onVisibility)
})
</script>
