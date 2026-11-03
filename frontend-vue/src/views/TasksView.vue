<!--
  TasksView (/tasks) — filterable task grid + detail panel (design/Tasks.dc.html).
  URL query: q (search), status (new|active|done), level (easy|medium|hard), task (<id> selected card),
  focus=search (focus the search input once, then dropped from the URL).
  Data: GET /api/tasks (task store), GET /api/tasks/:id (selected task), session store for start/resume.
-->
<template>
  <AppLayout>
    <PageHeader
      command="$ ls ./tasks"
      title="Vazifalar"
      subtitle="Har bir vazifa alohida Linux konteynerida bajariladi va avtomatik tekshiriladi."
    >
      <template v-if="tasks.length" #actions>
        <div class="flex items-center gap-2.5 font-mono text-[13px] text-muted">
          <div :style="{ width: `${tasks.length * 18 + (tasks.length - 1) * 3}px` }">
            <ProgressSegments
              :total="tasks.length"
              :states="segmentStates"
              :gap="3"
              :label="`Yakunlangan vazifalar: ${completedCount} / ${tasks.length}`"
            />
          </div>
          <span aria-hidden="true">{{ completedCount }}/{{ tasks.length }} done</span>
        </div>
      </template>
    </PageHeader>

    <TaskFilterBar
      ref="filterBar"
      v-model:search="searchText"
      :status="statusFilter"
      :level="levelFilter"
      @update:status="void setQuery({ status: $event === 'all' ? undefined : $event })"
      @update:level="void setQuery({ level: $event === 'all' ? undefined : $event })"
    />

    <!-- Loading -->
    <div v-if="showSkeleton" class="grid items-start gap-6 lg:grid-cols-[minmax(0,1fr)_360px]" aria-busy="true">
      <span class="sr-only" role="status">Vazifalar yuklanmoqda…</span>
      <ul class="grid gap-4 [grid-template-columns:repeat(auto-fill,minmax(260px,1fr))]" aria-hidden="true">
        <li v-for="i in 6" :key="i" class="overflow-hidden rounded-card border border-border bg-[rgba(15,21,32,0.9)]">
          <Skeleton height="auto" rounded="rounded-none" class="aspect-[320/140]" />
          <span class="flex flex-col gap-2.5 px-4 pb-4 pt-3.5">
            <Skeleton width="75%" height="16px" />
            <Skeleton width="55%" height="12px" />
          </span>
        </li>
      </ul>
      <div class="hidden overflow-hidden rounded-panel border border-border bg-[rgba(15,21,32,0.9)] lg:block" aria-hidden="true">
        <Skeleton height="auto" rounded="rounded-none" class="aspect-[320/140]" />
        <div class="flex flex-col gap-4 p-6">
          <Skeleton width="40%" height="12px" />
          <Skeleton width="80%" height="20px" />
          <Skeleton :lines="3" />
          <Skeleton height="120px" rounded="rounded-lg" />
        </div>
      </div>
    </div>

    <!-- Error (nothing cached to show) -->
    <EmptyState
      v-else-if="loadError"
      tone="danger"
      :icon="AlertTriangle"
      :command="`error: ${loadError}`"
      title="Vazifalarni yuklab bo'lmadi"
      description="Serverdan ma'lumot olib bo'lmadi. Birozdan so'ng qayta urinib ko'ring."
    >
      <template #actions>
        <Button variant="secondary" :icon="RotateCw" :loading="taskStore.isLoading" @click="reload">Qayta urinish</Button>
      </template>
    </EmptyState>

    <!-- No tasks at all -->
    <EmptyState
      v-else-if="tasks.length === 0"
      :icon="ListTodo"
      command="0 tasks"
      title="Hozircha vazifalar yo'q"
      description="Administrator vazifalarni yuklagach, ular shu yerda paydo bo'ladi."
    />

    <div v-else class="grid items-start gap-6 lg:grid-cols-[minmax(0,1fr)_360px]">
      <section aria-labelledby="tasks-list-title" class="min-w-0">
        <h2 id="tasks-list-title" class="sr-only">Vazifalar ro'yxati</h2>
        <p class="sr-only" role="status">{{ filtered.length }} ta vazifa topildi</p>
        <EmptyState
          v-if="filtered.length === 0"
          :icon="SearchX"
          command="0 results"
          title="Hech narsa topilmadi"
          description="Filtrlarni o'zgartirib ko'ring."
        >
          <template #actions>
            <Button variant="secondary" size="sm" @click="resetFilters">Filtrlarni tozalash</Button>
          </template>
        </EmptyState>
        <ul v-else class="grid gap-4 [grid-template-columns:repeat(auto-fill,minmax(260px,1fr))]">
          <li v-for="item in filtered" :key="item.task.id" class="flex">
            <TaskCard
              :task="item.task"
              :status="item.status"
              :selected="item.task.id === selectedTask?.id"
              @select="void pick(item.task.id)"
            />
          </li>
        </ul>
      </section>

      <TaskDetailPanel
        v-if="selectedTask"
        ref="detailPanel"
        class="scroll-mt-24"
        :task="selectedTask"
        :status="selectedStatus"
        :detail="detail"
        :detail-loading="detailLoading"
        :detail-error="detailError"
        :session-task="session.taskId === selectedTask.id"
        :remaining-label="session.remainingLabel"
        :starting="session.isStarting"
        @start="requestStart"
        @retry="loadDetail(selectedTask.id, true)"
      />
    </div>

    <ConfirmDialog
      v-model:open="confirmOpen"
      title="Faol sessiya to'xtatilsinmi?"
      :message="confirmMessage"
      confirm-label="To'xtatib boshlash"
      variant="danger"
      :loading="session.isStarting"
      @confirm="startSelected"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter, type LocationQueryRaw } from 'vue-router'
import { useMediaQuery } from '@vueuse/core'
import { AlertTriangle, ListTodo, RotateCw, SearchX } from 'lucide-vue-next'
import AppLayout from '@/components/ui/AppLayout.vue'
import Button from '@/components/ui/Button.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import PageHeader from '@/components/ui/PageHeader.vue'
import ProgressSegments from '@/components/ui/ProgressSegments.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import TaskCard from '@/components/tasks/TaskCard.vue'
import TaskDetailPanel from '@/components/tasks/TaskDetailPanel.vue'
import TaskFilterBar from '@/components/tasks/TaskFilterBar.vue'
import {
  matchesLevel,
  matchesSearch,
  matchesStatus,
  parseLevel,
  parseStatus,
  queryString
} from '@/components/tasks/filters'
import { useToast } from '@/composables/useToast'
import { taskStatus, type TaskStatus } from '@/constants/tasks'
import { errorMessage, isApiStatus } from '@/services/api'
import { useSessionStore } from '@/stores/session'
import { useTaskStore } from '@/stores/task'
import type { TaskDetail, TaskSummary } from '@/types'

const SEARCH_DEBOUNCE_MS = 300

const route = useRoute()
const router = useRouter()
const taskStore = useTaskStore()
const session = useSessionStore()
const toast = useToast()
const isDesktop = useMediaQuery('(min-width: 1024px)')

const filterBar = ref<InstanceType<typeof TaskFilterBar> | null>(null)
const detailPanel = ref<InstanceType<typeof TaskDetailPanel> | null>(null)

// ---------- Task list ----------

const tasks = computed(() => taskStore.tasks)
const showSkeleton = computed(
  () => taskStore.tasks.length === 0 && !taskStore.error && (taskStore.isLoading || !taskStore.loaded)
)
const loadError = computed(() => (taskStore.tasks.length === 0 ? taskStore.error : null))

function statusOf(task: TaskSummary): TaskStatus {
  return taskStatus(task, session.taskId)
}

const completedCount = computed(() => tasks.value.filter((t) => t.completed).length)
const segmentStates = computed(() =>
  tasks.value.map((t) => {
    const s = statusOf(t)
    return s === 'done' ? 'done' : s === 'active' ? 'active' : 'empty'
  })
)

function reload(): void {
  void taskStore.loadTasks(true)
}

// ---------- Filters (URL is the source of truth) ----------

const statusFilter = computed(() => parseStatus(route.query))
const levelFilter = computed(() => parseLevel(route.query))

/** Search text updates the list immediately; the URL follows after a short debounce. */
const searchText = ref(queryString(route.query.q))
let searchTimer: ReturnType<typeof setTimeout> | null = null

watch(searchText, (value) => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => void setQuery({ q: value.trim() ? value : undefined }), SEARCH_DEBOUNCE_MS)
})

// Back/forward navigation changes the query: reflect it in the input.
watch(
  () => queryString(route.query.q),
  (q) => {
    if (q !== searchText.value && !searchTimer) searchText.value = q
  }
)

/**
 * Patch the URL query (replace, no history entry). The shared router scrollBehavior scrolls to the top
 * on every navigation, so the scroll position is restored for these query-only updates.
 */
async function setQuery(patch: Record<string, string | undefined>): Promise<void> {
  if (Object.prototype.hasOwnProperty.call(patch, 'q') && searchTimer) {
    clearTimeout(searchTimer)
    searchTimer = null
  }
  const next: LocationQueryRaw = { ...route.query }
  for (const [key, value] of Object.entries(patch)) {
    if (value === undefined || value === '') delete next[key]
    else next[key] = value
  }
  const scrollY = window.scrollY
  await router.replace({ query: next })
  await new Promise<void>((resolve) => requestAnimationFrame(() => resolve()))
  if (window.scrollY !== scrollY) window.scrollTo({ top: scrollY })
}

function resetFilters(): void {
  searchText.value = ''
  void setQuery({ q: undefined, status: undefined, level: undefined })
}

const filtered = computed(() =>
  tasks.value
    .map((task) => ({ task, status: statusOf(task) }))
    .filter(
      ({ task, status }) =>
        matchesStatus(status, statusFilter.value) &&
        matchesLevel(task.difficulty, levelFilter.value) &&
        matchesSearch(task, searchText.value)
    )
)

// ?focus=search (from the header search button / Ctrl+K): focus once, then drop it from the URL.
watch(
  () => route.query.focus,
  async (focus) => {
    if (focus !== 'search') return
    await nextTick()
    filterBar.value?.focusSearch()
    void setQuery({ focus: undefined })
  },
  { immediate: true, flush: 'post' }
)

// ---------- Selection ----------

/** Default selection: running session's task, then an in-progress one, then the next open one. */
const defaultTask = computed<TaskSummary | undefined>(
  () =>
    taskStore.byId(session.taskId) ??
    tasks.value.find((t) => t.in_progress) ??
    tasks.value.find((t) => t.unlocked && !t.completed) ??
    tasks.value[0]
)

const selectedTask = computed<TaskSummary | undefined>(
  () => taskStore.byId(queryString(route.query.task)) ?? defaultTask.value
)
const selectedStatus = computed<TaskStatus>(() => (selectedTask.value ? statusOf(selectedTask.value) : 'open'))

async function pick(id: string): Promise<void> {
  await setQuery({ task: id })
  if (isDesktop.value) return
  // Stacked layout: the detail section sits under the grid, bring it into view.
  const el = detailPanel.value?.$el as HTMLElement | undefined
  const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  el?.scrollIntoView({ behavior: reduce ? 'auto' : 'smooth', block: 'start' })
  detailPanel.value?.focusHeading()
}

// ---------- Detail ----------

const detail = ref<TaskDetail | null>(null)
const detailLoading = ref(false)
const detailError = ref<string | null>(null)
let detailRequest = 0

async function loadDetail(id: string, force = false): Promise<void> {
  const req = ++detailRequest
  detail.value = taskStore.details[id] ?? null
  detailError.value = null
  if (detail.value && !force) return
  detailLoading.value = true
  try {
    const result = await taskStore.loadTask(id, force)
    if (req === detailRequest) detail.value = result
  } catch (err) {
    if (req === detailRequest) {
      detailError.value = isApiStatus(err, 404) ? 'Vazifa topilmadi.' : "Vazifa tafsilotlarini yuklab bo'lmadi."
    }
  } finally {
    if (req === detailRequest) detailLoading.value = false
  }
}

watch(
  () => selectedTask.value?.id,
  (id) => {
    if (id) void loadDetail(id)
  },
  { immediate: true }
)

// ---------- Start / resume ----------

const confirmOpen = ref(false)

const confirmMessage = computed(() => {
  const running = taskStore.byId(session.taskId)?.title ?? session.taskId ?? ''
  const next = selectedTask.value?.title ?? ''
  return `"${running}" uchun ishlayotgan konteyner to'xtatiladi va "${next}" uchun yangi laboratoriya ochiladi. Natijalaringiz saqlanadi.`
})

function requestStart(): void {
  const task = selectedTask.value
  if (!task || session.isStarting) return
  if (session.isActive && session.taskId !== task.id) {
    confirmOpen.value = true
    return
  }
  void startSelected()
}

async function startSelected(): Promise<void> {
  const task = selectedTask.value
  if (!task) return
  try {
    await session.start(task.id)
    confirmOpen.value = false
    taskStore.invalidate()
    await router.push({ name: 'lab', params: { taskId: task.id } })
  } catch (err) {
    confirmOpen.value = false
    if (isApiStatus(err, 409)) {
      toast.warning('Bu vazifa hali qulflangan. Avval oldingi vazifani yakunlang.')
      reload()
    } else if (isApiStatus(err, 503)) {
      toast.error("Laboratoriyada bo'sh joy qolmadi. Bir necha daqiqadan so'ng qayta urinib ko'ring.")
    } else {
      toast.error(errorMessage(err, "Laboratoriyani ishga tushirib bo'lmadi."))
    }
  }
}

// ---------- Lifecycle ----------

onMounted(() => {
  // Show cached tasks instantly, but refresh status flags in the background.
  void taskStore.loadTasks(taskStore.loaded)
})

onBeforeUnmount(() => {
  if (searchTimer) clearTimeout(searchTimer)
})
</script>
