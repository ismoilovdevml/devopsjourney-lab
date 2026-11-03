<!--
  ProgressTaskTable — per-task results: cover thumbnail + title, status badge, best check score,
  attempts, time spent and completion date.
  Props:
    tasks: TaskSummary[]            (sorted by order)
    entries: ProgressEntry[]        (GET /api/progress .progress)
    activeTaskId?: string | null    (task with a running lab session)
    loading?: boolean
-->
<template>
  <section aria-labelledby="progress-table-title" class="overflow-hidden rounded-panel border border-border bg-surface backdrop-blur-[6px]">
    <div class="border-b border-divider px-5 py-[18px] sm:px-6">
      <h2 id="progress-table-title" class="font-mono text-[15px] font-semibold text-text">Vazifalar bo'yicha</h2>
    </div>

    <div v-if="!loading && rows.length === 0" class="p-5">
      <EmptyState :icon="ListX" title="Vazifalar topilmadi" description="Hozircha hech qanday vazifa yuklanmagan." />
    </div>

    <div v-else class="overflow-x-auto">
      <table class="w-full min-w-[820px] border-collapse">
        <caption class="sr-only">Har bir vazifa bo'yicha natijalar</caption>
        <thead>
          <tr class="th text-left">
            <th scope="col" class="px-6 py-3 font-medium">Vazifa</th>
            <th scope="col" class="px-4 py-3 font-medium">Holat</th>
            <th scope="col" class="px-4 py-3 font-medium">Ball</th>
            <th scope="col" class="px-4 py-3 font-medium">Urinish</th>
            <th scope="col" class="px-4 py-3 font-medium">Vaqt</th>
            <th scope="col" class="py-3 pl-4 pr-6 font-medium">Sana</th>
          </tr>
        </thead>
        <tbody>
          <template v-if="loading">
            <tr v-for="i in 6" :key="i" class="border-t border-divider">
              <td class="px-6 py-3">
                <span class="flex items-center gap-3">
                  <Skeleton width="64px" height="30px" />
                  <Skeleton width="220px" height="14px" />
                </span>
              </td>
              <td class="px-4 py-3"><Skeleton width="90px" height="22px" rounded="rounded-full" /></td>
              <td class="px-4 py-3"><Skeleton width="40px" /></td>
              <td class="px-4 py-3"><Skeleton width="24px" /></td>
              <td class="px-4 py-3"><Skeleton width="56px" /></td>
              <td class="py-3 pl-4 pr-6"><Skeleton width="70px" /></td>
            </tr>
          </template>
          <tr v-for="row in rows" v-else :key="row.task.id" class="border-t border-divider">
            <th scope="row" class="px-6 py-3 text-left font-medium">
              <RouterLink
                v-if="row.status !== 'locked'"
                :to="{ name: 'lab', params: { taskId: row.task.id } }"
                class="group flex items-center gap-3 rounded-md text-text hover:text-text"
              >
                <TaskCover
                  :task-id="row.task.id"
                  :category="row.task.category"
                  aspect="auto"
                  position="right"
                  rounded="rounded-chip"
                  class="h-[30px] !w-16 shrink-0 border border-border"
                />
                <span class="group-hover:underline group-hover:underline-offset-2">{{ row.task.title }}</span>
              </RouterLink>
              <span v-else class="flex items-center gap-3 text-text-2">
                <TaskCover
                  :task-id="row.task.id"
                  :category="row.task.category"
                  aspect="auto"
                  position="right"
                  rounded="rounded-chip"
                  class="h-[30px] !w-16 shrink-0 border border-border opacity-60"
                />
                {{ row.task.title }}
              </span>
            </th>
            <td class="px-4 py-3"><Badge :variant="row.status" /></td>
            <td class="px-4 py-3 font-mono text-[13px] text-text">
              <span :aria-label="row.scoreLabel">{{ row.score }}</span>
            </td>
            <td class="px-4 py-3 font-mono text-[13px] text-text">{{ row.attempts }}</td>
            <td class="px-4 py-3 font-mono text-[13px] text-text">
              <template v-if="row.time === null">—</template>
              <span v-else :title="row.timeLong">{{ row.time }}</span>
            </td>
            <td class="whitespace-nowrap py-3 pl-4 pr-6 text-muted">{{ row.date ?? '—' }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ListX } from 'lucide-vue-next'
import Badge from '@/components/ui/Badge.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import TaskCover from '@/components/ui/TaskCover.vue'
import { taskStatus, type TaskStatus } from '@/constants/tasks'
import type { ProgressEntry, TaskSummary } from '@/types'
import { formatDayMonth, formatDurationLong, formatDurationShort } from './format'

const props = defineProps<{
  tasks: TaskSummary[]
  entries: ProgressEntry[]
  activeTaskId?: string | null
  loading?: boolean
}>()

interface Row {
  task: TaskSummary
  status: TaskStatus
  score: string
  scoreLabel: string
  attempts: number
  time: string | null
  timeLong: string
  date: string | null
}

const rows = computed<Row[]>(() => {
  const byTask = new Map(props.entries.map((e) => [e.task_id, e]))
  return props.tasks.map((task) => {
    const e = byTask.get(task.id)
    const total = e && e.total > 0 ? e.total : task.checks_count
    const hasScore = !!e && e.attempts > 0
    return {
      task,
      status: taskStatus(task, props.activeTaskId),
      score: `${hasScore ? e.score : '—'}/${total}`,
      scoreLabel: hasScore
        ? `${total} ta tekshiruvdan ${e.score} tasi o'tdi`
        : `Hali tekshirilmagan, ${total} ta tekshiruv`,
      attempts: e?.attempts ?? 0,
      time: e && e.time_spent > 0 ? formatDurationShort(e.time_spent) : null,
      timeLong: formatDurationLong(e?.time_spent ?? 0),
      date: e?.passed ? formatDayMonth(e.completed_at) : null
    }
  })
})
</script>
