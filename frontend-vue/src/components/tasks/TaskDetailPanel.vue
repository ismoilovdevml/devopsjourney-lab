<!--
  TaskDetailPanel — right-hand detail for the selected task: cover, id, description, mini stats,
  "# nima tekshiriladi" list, README and the call to action.
  Summary fields render immediately; detail fields (image, checks, readme) come from GET /api/tasks/:id.
  Props:
    task: TaskSummary
    status: TaskStatus
    detail: TaskDetail | null
    detailLoading?: boolean
    detailError?: string | null
    sessionTask?: boolean      (the running lab session belongs to this task -> "Terminalga qaytish")
    remainingLabel?: string    (mm:ss left in that session)
    starting?: boolean         (start request in flight)
  Emits: start, retry
  Exposes: focusHeading()
-->
<template>
  <aside
    :aria-labelledby="headingId"
    class="flex flex-col overflow-hidden rounded-panel border border-border bg-[rgba(15,21,32,0.9)]"
  >
    <TaskCover :task-id="task.id" :category="task.category" class="max-h-[220px] lg:max-h-none" />
    <div class="flex flex-col gap-[18px] px-5 pb-6 pt-[22px] sm:px-6">
      <div>
        <p class="mb-1.5 font-mono text-meta text-muted">tasks/{{ task.id }}</p>
        <h2
          :id="headingId"
          ref="heading"
          tabindex="-1"
          class="text-[21px] font-semibold leading-tight tracking-[-0.01em] text-text outline-none"
        >
          {{ task.title }}
        </h2>
      </div>

      <p class="text-text-2">{{ task.description }}</p>

      <dl class="grid grid-cols-3 gap-2">
        <div class="min-w-0 rounded-btn border border-border bg-surface-2 px-3 py-2.5">
          <dt class="text-xs text-muted">Ball</dt>
          <dd class="font-mono font-semibold text-accent">{{ task.points }} xp</dd>
        </div>
        <div class="min-w-0 rounded-btn border border-border bg-surface-2 px-3 py-2.5">
          <dt class="text-xs text-muted">Tekshiruv</dt>
          <dd class="font-mono font-semibold text-text">{{ checksCount }}</dd>
        </div>
        <div class="min-w-0 rounded-btn border border-border bg-surface-2 px-3 py-2.5">
          <dt class="text-xs text-muted">Image</dt>
          <dd class="min-h-[21px] truncate font-mono text-xs font-semibold leading-[21px] text-text" :title="imageLabel">
            <Skeleton v-if="detailLoading && !detail" width="70%" height="12px" class="mt-[5px]" />
            <template v-else>{{ imageLabel }}</template>
          </dd>
        </div>
      </dl>

      <section :aria-labelledby="checksHeadingId">
        <h3 :id="checksHeadingId" class="mb-2.5 font-mono text-[13px] font-semibold text-muted"># nima tekshiriladi</h3>
        <div v-if="detailLoading && !detail" class="flex flex-col gap-1.5" aria-busy="true">
          <Skeleton v-for="i in Math.min(task.checks_count || 4, 6)" :key="i" height="40px" rounded="rounded-lg" />
          <span class="sr-only">Yuklanmoqda…</span>
        </div>
        <div
          v-else-if="detailError && !detail"
          class="flex flex-wrap items-center gap-3 rounded-lg border border-dashed border-danger/40 px-3 py-2.5 text-[13px] text-text-2"
          role="alert"
        >
          <AlertTriangle class="h-4 w-4 shrink-0 text-danger" aria-hidden="true" />
          <span class="min-w-0 flex-1">{{ detailError }}</span>
          <Button size="sm" variant="secondary" :icon="RotateCw" @click="emit('retry')">Qayta urinish</Button>
        </div>
        <p v-else-if="checks.length === 0" class="text-[13px] text-muted">Bu vazifa uchun tekshiruvlar ro'yxati yo'q.</p>
        <ol v-else class="flex flex-col gap-1.5">
          <li
            v-for="(check, i) in checks"
            :key="`${i}-${check.name}`"
            class="flex items-center gap-2.5 rounded-lg border border-divider bg-surface-2 px-3 py-[9px] text-[13.5px] text-text"
          >
            <span class="shrink-0 font-mono text-accent" aria-hidden="true">{{ status === 'done' ? '[x]' : '[ ]' }}</span>
            <span class="min-w-0 break-words">{{ check.name }}</span>
          </li>
        </ol>
      </section>

      <TaskReadme v-if="detail?.readme" :key="task.id" :markdown="detail.readme" />

      <div
        v-if="status === 'locked'"
        class="flex items-center gap-2.5 rounded-btn border border-dashed border-border-stronger px-3.5 py-3 text-text-2"
      >
        <Lock class="h-4 w-4 shrink-0 text-warn" aria-hidden="true" />
        Ochish uchun avvalgi vazifalarni yakunlang
      </div>
      <Button
        v-else-if="sessionTask"
        variant="primary"
        size="lg"
        block
        :icon="TerminalIcon"
        :to="{ name: 'lab', params: { taskId: task.id } }"
      >
        Terminalga qaytish
      </Button>
      <Button
        v-else
        variant="primary"
        size="lg"
        block
        :icon="status === 'done' ? RotateCcw : TerminalIcon"
        :loading="starting"
        @click="emit('start')"
      >
        {{ status === 'done' ? 'Qayta ishlash' : 'Laboratoriyani boshlash' }}
      </Button>

      <p class="text-meta text-muted">
        <template v-if="sessionTask && remainingLabel">
          Sessiya tugashiga <span class="font-mono text-text-2">{{ remainingLabel }}</span> qoldi.
        </template>
        <template v-else>Sessiya vaqti cheklangan.</template>
        Konteyner tugagach o'chiriladi — natijalar saqlanadi.
      </p>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { AlertTriangle, Lock, RotateCcw, RotateCw, SquareTerminal as TerminalIcon } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import TaskCover from '@/components/ui/TaskCover.vue'
import { uid } from '@/components/ui/utils'
import type { TaskStatus } from '@/constants/tasks'
import type { TaskDetail, TaskSummary } from '@/types'
import TaskReadme from './TaskReadme.vue'

const props = defineProps<{
  task: TaskSummary
  status: TaskStatus
  detail: TaskDetail | null
  detailLoading?: boolean
  detailError?: string | null
  sessionTask?: boolean
  remainingLabel?: string
  starting?: boolean
}>()

const emit = defineEmits<{ start: []; retry: [] }>()

const headingId = uid('task-detail')
const checksHeadingId = uid('task-checks')
const heading = ref<HTMLElement | null>(null)

const checks = computed(() => props.detail?.checks ?? [])
const checksCount = computed(() => props.detail?.checks.length ?? props.task.checks_count)
/** Tasks without an explicit image run on the backend's default lab image. */
const imageLabel = computed(() => (props.detail ? props.detail.image || 'standart' : '—'))

function focusHeading(): void {
  heading.value?.focus({ preventScroll: true })
}

defineExpose({ focusHeading })
</script>
