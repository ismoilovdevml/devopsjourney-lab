<!--
  LearningPath — "./learning-path" card: every task in order with cover thumbnail, meta and status badge.
  Unlocked tasks link to the lab; locked ones link to the task description (/tasks?task=<id>).
  Props:
    tasks: TaskSummary[]          (sorted by order)
    activeTaskId?: string | null  (running session task, shown as "Jarayonda")
    totalScore?: number           (earned points, header "x / y xp")
    totalPossible?: number
    loading?: boolean
-->
<template>
  <section
    aria-labelledby="dash-path-title"
    class="min-w-0 flex-[999_1_600px] rounded-panel border border-border bg-surface backdrop-blur-[6px]"
  >
    <header class="flex flex-wrap items-center justify-between gap-3 border-b border-divider px-5 py-5 sm:px-6">
      <div class="min-w-0">
        <h2 id="dash-path-title" class="font-mono text-base font-semibold text-text">./learning-path</h2>
        <p class="mt-0.5 text-[13px] text-muted">
          Linux asoslaridan avtomatlashtirishgacha<template v-if="!loading && tasks.length"> — {{ tasks.length }} ta amaliy vazifa</template>
        </p>
      </div>
      <span v-if="!loading && totalPossible" class="font-mono text-[13px] text-accent">
        {{ totalScore ?? 0 }} / {{ totalPossible }} xp
      </span>
    </header>

    <ol v-if="loading" class="m-0 list-none p-2.5" aria-busy="true" aria-label="Yuklanmoqda">
      <li v-for="i in 6" :key="i" class="flex items-center gap-4 px-3 py-2.5">
        <Skeleton width="104px" height="46px" rounded="rounded-lg" class="hidden sm:block" />
        <Skeleton width="72px" height="40px" rounded="rounded-lg" class="sm:hidden" />
        <span class="flex flex-1 flex-col gap-1.5">
          <Skeleton width="60%" height="16px" />
          <Skeleton width="40%" height="12px" />
        </span>
        <Skeleton width="76px" height="22px" rounded="rounded-full" />
      </li>
    </ol>

    <div v-else-if="!tasks.length" class="p-5">
      <EmptyState
        :icon="ListTodo"
        command="ls ./tasks: 0 results"
        title="Vazifalar topilmadi"
        description="Hozircha o'quv yo'lida vazifa yo'q. Keyinroq qaytib ko'ring."
      />
    </div>

    <ol v-else class="m-0 list-none p-2.5">
      <li v-for="row in rows" :key="row.task.id">
        <RouterLink
          :to="row.to"
          :class="[
            'flex items-center gap-3 rounded-xl px-2 py-2.5 text-text transition-colors hover:bg-surface-3 hover:text-text sm:gap-4 sm:px-3',
            row.status === 'active' ? 'bg-info/[0.04]' : ''
          ]"
        >
          <span
            :class="[
              'block h-10 w-[72px] shrink-0 overflow-hidden rounded-lg border border-border sm:h-[46px] sm:w-[104px]',
              row.status === 'locked' ? 'opacity-50 grayscale' : ''
            ]"
          >
            <TaskCover
              :task-id="row.task.id"
              :category="row.task.category"
              aspect="auto"
              position="right"
              class="h-full"
            />
          </span>
          <span class="min-w-0 flex-1">
            <span class="block font-semibold" :class="row.status === 'locked' ? 'text-text-2' : ''">
              <span class="font-mono font-medium text-muted">{{ row.number }}</span>&nbsp; {{ row.task.title }}
            </span>
            <span class="block text-meta text-muted">{{ row.meta }}</span>
          </span>
          <Badge :variant="row.status" :label="row.badge" :icon="false" />
        </RouterLink>
      </li>
    </ol>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { RouteLocationRaw } from 'vue-router'
import { ListTodo } from 'lucide-vue-next'
import Badge from '@/components/ui/Badge.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import TaskCover from '@/components/ui/TaskCover.vue'
import {
  TASK_STATUS_LABEL,
  difficultyMeta,
  taskNumber,
  taskStatus,
  taskVisual,
  type TaskStatus
} from '@/constants/tasks'
import type { TaskSummary } from '@/types'

const props = defineProps<{
  tasks: TaskSummary[]
  activeTaskId?: string | null
  totalScore?: number
  totalPossible?: number
  loading?: boolean
}>()

interface Row {
  task: TaskSummary
  number: string
  meta: string
  status: TaskStatus
  statusLabel: string
  badge: string
  to: RouteLocationRaw
}

function badgeLabel(task: TaskSummary, status: TaskStatus): string {
  if (status === 'done') return `${task.points} / ${task.points}`
  if (status === 'open') return `${task.points} xp`
  return TASK_STATUS_LABEL[status]
}

const rows = computed<Row[]>(() =>
  props.tasks.map((task, i) => {
    const status = taskStatus(task, props.activeTaskId)
    const checks = task.checks_count > 0 ? ` · ${task.checks_count} tekshiruv` : ''
    return {
      task,
      number: taskNumber(i + 1),
      meta: `${taskVisual(task).categoryLabel} · ${difficultyMeta(task.difficulty).label}${checks}`,
      status,
      statusLabel: status === 'open' ? `${TASK_STATUS_LABEL.open}, ${task.points} xp` : TASK_STATUS_LABEL[status],
      badge: badgeLabel(task, status),
      to:
        status === 'locked'
          ? { name: 'tasks', query: { task: task.id } }
          : { name: 'lab', params: { taskId: task.id } }
    }
  })
)
</script>
