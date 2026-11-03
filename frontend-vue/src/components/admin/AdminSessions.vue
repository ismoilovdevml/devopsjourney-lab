<!--
  AdminSessions — active lab containers with a "kill" action (ConfirmDialog -> terminate).
  Auto-refreshes every 10s while the tab is visible; pauses when the document is hidden and
  refreshes immediately when it becomes visible again. Background refresh errors keep the last data.
  Data: GET /api/admin/sessions, POST /api/admin/sessions/:id/terminate.
  Emits: count(n) (for the side nav), changed (after a session was terminated)
  Exposes: refresh()
-->
<template>
  <div class="flex flex-col gap-3">
    <div class="flex flex-wrap items-center justify-between gap-2 font-mono text-meta text-muted">
      <span class="inline-flex items-center gap-2" aria-live="polite">
        <span
          class="h-[7px] w-[7px] rounded-full"
          :class="paused ? 'bg-faint' : 'bg-accent shadow-glow-sm'"
          aria-hidden="true"
        />
        {{ paused ? "auto-refresh pauza (tab yashirin)" : `auto-refresh ${REFRESH_MS / 1000}s` }}
      </span>
      <span v-if="updatedAt">
        yangilandi {{ updatedAt }}
        <span v-if="staleError" class="text-warn" :title="staleError"> · oxirgi so'rov xato</span>
      </span>
    </div>

    <SectionError v-if="error && !loaded" :message="error" command="GET /api/admin/sessions" :retrying="loading" @retry="refresh" />

    <EmptyState
      v-else-if="loaded && sessions.length === 0"
      :icon="Container"
      command="docker ps --filter label=devops-lab → 0"
      title="Faol sessiyalar yo'q"
      description="Hozir hech bir o'quvchi laboratoriyada ishlamayapti."
    />

    <section v-else class="relative overflow-x-auto rounded-panel border border-border bg-[rgba(15,21,32,0.88)]" aria-label="Faol sessiyalar jadvali">
      <table class="w-full min-w-[800px] border-collapse">
        <caption class="sr-only">Faol laboratoriya sessiyalari: konteyner, o'quvchi, vazifa, qolgan vaqt</caption>
        <thead>
          <tr class="th text-left">
            <th scope="col" class="px-5 py-3 font-medium">Container</th>
            <th scope="col" class="px-4 py-3 font-medium">O'quvchi</th>
            <th scope="col" class="px-4 py-3 font-medium">Vazifa</th>
            <th scope="col" class="px-4 py-3 font-medium">Boshlangan</th>
            <th scope="col" class="px-4 py-3 font-medium">TTL</th>
            <th scope="col" class="py-3 pl-4 pr-5 text-right font-medium"><span class="sr-only">Amal</span></th>
          </tr>
        </thead>
        <tbody v-if="!loaded">
          <tr v-for="i in 5" :key="i" class="border-t border-divider">
            <td class="px-5 py-3"><Skeleton width="120px" /></td>
            <td class="px-4 py-3"><Skeleton width="90px" /></td>
            <td class="px-4 py-3"><Skeleton width="180px" /></td>
            <td class="px-4 py-3"><Skeleton width="80px" /></td>
            <td class="px-4 py-3"><Skeleton width="50px" /></td>
            <td class="py-3 pl-4 pr-5"><Skeleton width="44px" height="32px" class="ml-auto" /></td>
          </tr>
        </tbody>
        <tbody v-else>
          <tr v-for="s in rows" :key="s.id" class="border-t border-divider">
            <td class="px-5 py-3 font-mono text-meta">
              <span :class="s.expiring ? 'text-warn' : 'text-accent'" aria-hidden="true">●</span>
              <span class="ml-1.5" :title="s.id">{{ s.shortId }}</span>
            </td>
            <th scope="row" class="px-4 py-3 text-left font-mono font-medium text-text">{{ s.username }}</th>
            <td class="px-4 py-3 text-text-2">
              <span class="block">{{ s.taskTitle }}</span>
              <span v-if="s.taskTitle !== s.task_id" class="font-mono text-xs text-faint">{{ s.task_id }}</span>
            </td>
            <td class="px-4 py-3 text-muted">
              <time :datetime="s.created_at" :title="fullDateTime(s.created_at)">{{ relativeTime(s.created_at, now) }}</time>
            </td>
            <td class="px-4 py-3 font-mono text-[13px]" :class="s.expiring ? 'text-warn' : 'text-text'">
              <span :aria-label="`${s.ttl} qoldi`">{{ s.ttl }}</span>
            </td>
            <td class="py-3 pl-4 pr-5 text-right">
              <button
                type="button"
                class="inline-flex h-8 items-center rounded-[7px] border border-danger/45 bg-danger/[0.08] px-2.5 font-mono text-[13px] text-danger transition-colors hover:border-danger hover:bg-danger/15 disabled:cursor-not-allowed disabled:opacity-50"
                :disabled="terminating === s.id"
                :aria-label="`${s.username} sessiyasini to'xtatish (${s.shortId})`"
                @click="target = s"
              >
                kill
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </section>

    <ConfirmDialog
      :open="target !== null"
      title="Sessiyani to'xtatish"
      :message="target ? `${target.username} ning ${target.taskTitle} konteyneri (${target.shortId}) o'chiriladi. Terminaldagi ishi yo'qoladi.` : ''"
      confirm-label="kill"
      :loading="terminating !== null"
      @update:open="(v) => { if (!v && terminating === null) target = null }"
      @confirm="terminate"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { Container } from 'lucide-vue-next'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import SectionError from './SectionError.vue'
import { duration, fullDateTime, parseTime, relativeTime } from './format'
import { api, errorMessage } from '@/services/api'
import { useTaskStore } from '@/stores/task'
import { useToast } from '@/composables/useToast'
import type { AdminSession } from '@/types'

interface Row extends AdminSession {
  shortId: string
  taskTitle: string
  ttl: string
  expiring: boolean
}

const emit = defineEmits<{ count: [value: number]; changed: [] }>()

const REFRESH_MS = 10_000
const EXPIRING_SECONDS = 5 * 60

const tasks = useTaskStore()
const toast = useToast()

const sessions = ref<AdminSession[]>([])
const loaded = ref(false)
const loading = ref(false)
const error = ref<string | null>(null)
const staleError = ref<string | null>(null)
const updatedAt = ref('')
const paused = ref(typeof document !== 'undefined' && document.visibilityState === 'hidden')
const now = ref(Date.now())
const target = ref<Row | null>(null)
const terminating = ref<string | null>(null)

let pollTimer: ReturnType<typeof setInterval> | null = null
let tickTimer: ReturnType<typeof setInterval> | null = null

const rows = computed<Row[]>(() =>
  sessions.value.map((s) => {
    const expires = parseTime(s.expires_at)
    const left = expires ? (expires.getTime() - now.value) / 1000 : 0
    return {
      ...s,
      shortId: s.id.length > 14 ? `${s.id.slice(0, 12)}…` : s.id,
      taskTitle: tasks.byId(s.task_id)?.title ?? s.task_id,
      ttl: expires ? duration(left) : '—',
      expiring: left < EXPIRING_SECONDS
    }
  })
)

async function refresh(): Promise<void> {
  if (loading.value) return
  loading.value = true
  if (!loaded.value) error.value = null
  try {
    // Drop sessions that are being terminated so a poll cannot resurrect them.
    const list = await api.adminSessions()
    sessions.value = terminating.value ? list.filter((s) => s.id !== terminating.value) : list
    loaded.value = true
    error.value = null
    staleError.value = null
    updatedAt.value = new Date().toLocaleTimeString('en-GB')
    emit('count', sessions.value.length)
  } catch (err) {
    const msg = errorMessage(err, "Sessiyalarni yuklab bo'lmadi.")
    if (loaded.value) staleError.value = msg
    else error.value = msg
  } finally {
    loading.value = false
    now.value = Date.now()
  }
}

async function terminate() {
  const s = target.value
  if (!s) return
  terminating.value = s.id
  try {
    await api.adminTerminateSession(s.id)
    sessions.value = sessions.value.filter((x) => x.id !== s.id)
    emit('count', sessions.value.length)
    emit('changed')
    toast.success(`${s.username} sessiyasi to'xtatildi`)
    target.value = null
  } catch (err) {
    toast.error(errorMessage(err, "Sessiyani to'xtatib bo'lmadi."))
  } finally {
    terminating.value = null
  }
}

function startPolling() {
  stopPolling()
  pollTimer = setInterval(() => void refresh(), REFRESH_MS)
}

function stopPolling() {
  if (pollTimer) clearInterval(pollTimer)
  pollTimer = null
}

function onVisibility() {
  if (document.visibilityState === 'hidden') {
    paused.value = true
    stopPolling()
  } else {
    paused.value = false
    void refresh()
    startPolling()
  }
}

onMounted(() => {
  void refresh()
  if (!paused.value) startPolling()
  tickTimer = setInterval(() => (now.value = Date.now()), 1000)
  document.addEventListener('visibilitychange', onVisibility)
})

onBeforeUnmount(() => {
  stopPolling()
  if (tickTimer) clearInterval(tickTimer)
  document.removeEventListener('visibilitychange', onVisibility)
})

defineExpose({ refresh, loading })
</script>
