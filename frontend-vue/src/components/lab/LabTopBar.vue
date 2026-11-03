<!--
  LabTopBar — workspace header (replaces the main nav on /lab).
  Logo, breadcrumb `tasks / <id>`, learning-path indicator (one segment per task: done = green,
  current = blue), session timer pill, "Qayta yaratish" and "Tugatish".
  Props:
    taskId: string | null
    tasks: TaskSummary[]            (ordered; drives the path indicator)
    sessionActive: boolean          (session for this task is running -> pill + actions enabled)
    remainingSeconds: number
    remainingLabel: string          ("mm:ss")
    recreating?: boolean
    stopping?: boolean
    checking?: boolean              (a check is running -> session actions are blocked)
  Emits: recreate, finish
-->
<template>
  <header
    class="flex flex-wrap items-center gap-x-3.5 gap-y-2 border-b border-border bg-[rgba(10,14,20,0.9)] px-4 py-2 backdrop-blur-md"
  >
    <RouterLink
      to="/"
      aria-label="Bosh sahifa"
      class="flex h-[30px] w-[30px] shrink-0 items-center justify-center rounded-lg bg-accent shadow-[0_0_18px_rgba(61,220,151,0.45)]"
    >
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#04130C" stroke-width="2.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <path d="M5 7l5 5-5 5" />
        <path d="M13 17h6" />
      </svg>
    </RouterLink>

    <nav aria-label="Yo'l" class="flex min-w-0 items-center gap-2 font-mono text-[13px] text-muted">
      <RouterLink to="/tasks" class="rounded hover:text-text">tasks</RouterLink>
      <span aria-hidden="true">/</span>
      <span class="truncate text-text" aria-current="page">{{ taskId ?? '—' }}</span>
    </nav>

    <div
      v-if="tasks.length"
      class="ml-1.5 flex gap-[3px]"
      role="img"
      :aria-label="pathLabel"
      :title="pathLabel"
    >
      <span
        v-for="t in tasks"
        :key="t.id"
        class="h-[5px] w-4 rounded-[2px]"
        :class="t.id === taskId ? 'bg-info' : t.completed ? 'bg-accent' : 'bg-border-strong'"
      />
    </div>

    <div class="ml-auto flex flex-wrap items-center gap-2">
      <span
        v-if="sessionActive"
        role="timer"
        :aria-label="`Sessiya tugashiga ${remainingLabel} qoldi`"
        :class="[
          'inline-flex h-9 items-center gap-2 rounded-lg border px-3 font-mono text-[13px] tabular-nums',
          warn ? 'border-warn/40 bg-warn/10 text-warn' : 'border-accent/35 bg-accent/10 text-accent'
        ]"
      >
        <span
          :class="['h-[7px] w-[7px] rounded-full', warn ? 'bg-warn' : 'bg-accent shadow-glow-sm']"
          aria-hidden="true"
        />
        sessiya {{ remainingLabel }}
      </span>

      <Button
        size="sm"
        :icon="RefreshCw"
        :disabled="!sessionActive || stopping || checking"
        :loading="recreating"
        @click="emit('recreate')"
      >
        <span class="max-sm:sr-only">Qayta yaratish</span>
      </Button>
      <Button
        size="sm"
        variant="danger"
        :disabled="!sessionActive || recreating || checking"
        :loading="stopping"
        @click="emit('finish')"
      >
        Tugatish
      </Button>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RefreshCw } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import type { TaskSummary } from '@/types'

const props = defineProps<{
  taskId: string | null
  tasks: TaskSummary[]
  sessionActive: boolean
  remainingSeconds: number
  remainingLabel: string
  recreating?: boolean
  stopping?: boolean
  checking?: boolean
}>()

const emit = defineEmits<{ recreate: []; finish: [] }>()

/** Below 5 minutes the timer turns amber. */
const WARN_SECONDS = 5 * 60

const warn = computed(() => props.sessionActive && props.remainingSeconds < WARN_SECONDS)

const pathLabel = computed(() => {
  const idx = props.tasks.findIndex((t) => t.id === props.taskId)
  const done = props.tasks.filter((t) => t.completed).length
  const position = idx >= 0 ? `Vazifa ${idx + 1} / ${props.tasks.length}` : `${props.tasks.length} ta vazifa`
  return `${position} · ${done} ta yakunlangan`
})
</script>
