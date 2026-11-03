<!--
  LabTaskPanel — left workspace panel: cover header with the task title and three tabs
  (Topshiriq / Maslahatlar / Natija) following the WAI-ARIA tabs pattern (arrow keys, Home/End).
  Props:
    task: TaskDetail | null        (null while loading -> skeleton)
    tab: LabTab                    (v-model:tab)
    check: CheckResponse | null
    checking: boolean
    attempts: number
    checkedAt: number | null
    canCheck: boolean
    nextTask: TaskSummary | null
    nextLoading?: boolean
  Emits: update:tab, run-check, next
-->
<template>
  <section
    aria-label="Topshiriq"
    class="flex min-h-0 min-w-0 flex-col border-border bg-[rgba(13,18,27,0.94)] lg:w-[clamp(380px,28vw,470px)] lg:shrink-0 lg:border-r"
  >
    <div class="relative">
      <TaskCover
        v-if="task"
        :task-id="task.id"
        :category="task.category"
        aspect="auto"
        class="h-[104px]"
      />
      <Skeleton v-else height="104px" rounded="rounded-none" />
      <!-- Left-to-right fade keeps the title readable over the cover artwork. -->
      <div class="absolute inset-0 flex flex-col justify-end bg-gradient-to-r from-[#0D121B] via-[#0D121B]/90 to-[#0D121B]/20 px-5 pb-2.5">
        <template v-if="task">
          <p class="font-mono text-xs text-text-2">
            {{ taskNumber(task.order) }} · {{ visual.categoryLabel }} ·
            <span :style="{ color: difficulty.color }">{{ difficulty.label }}</span> · +{{ task.points }} xp
          </p>
          <h1 class="text-lg font-semibold leading-snug text-text">{{ task.title }}</h1>
        </template>
        <template v-else>
          <Skeleton width="45%" height="12px" />
          <Skeleton width="70%" height="20px" class="mt-1.5" />
        </template>
      </div>
    </div>

    <div role="tablist" aria-label="Panel" class="flex gap-0.5 border-b border-border px-3" @keydown="onTabKeydown">
      <button
        v-for="t in TABS"
        :id="tabId(t.key)"
        :key="t.key"
        :ref="(el) => setTabRef(t.key, el)"
        type="button"
        role="tab"
        :aria-selected="tab === t.key"
        :aria-controls="panelId(t.key)"
        :tabindex="tab === t.key ? 0 : -1"
        :class="[
          '-mb-px inline-flex h-[42px] items-center gap-1.5 border-b-2 px-3 font-medium transition-colors focus-visible:outline-offset-[-2px]',
          tab === t.key ? 'border-accent text-text' : 'border-transparent text-muted hover:text-text-2'
        ]"
        @click="select(t.key)"
      >
        {{ t.label }}
        <span
          v-if="t.key === 'results' && check"
          :class="[
            'rounded-full px-1.5 font-mono text-[11px] leading-[18px]',
            check.passed ? 'bg-accent/15 text-accent' : 'bg-danger/15 text-danger'
          ]"
        >{{ check.score }}/{{ check.total }}</span>
      </button>
    </div>

    <div
      :id="panelId(tab)"
      ref="body"
      role="tabpanel"
      :aria-labelledby="tabId(tab)"
      tabindex="0"
      class="min-h-0 flex-1 overflow-y-auto px-5 pb-7 pt-5 focus-visible:outline-offset-[-2px] max-lg:max-h-[55vh]"
    >
      <div v-if="!task" class="flex flex-col gap-3" aria-busy="true">
        <Skeleton :lines="3" />
        <Skeleton height="120px" rounded="rounded-btn" />
        <Skeleton :lines="4" />
      </div>
      <LabTaskTab
        v-else-if="tab === 'task'"
        :task="task"
        :readme="readme.body"
        :check="check"
        :has-hints="task.hints.length > 0 || !!readme.hints"
        @open-hints="select('hints', true)"
      />
      <LabHintsTab v-else-if="tab === 'hints'" :task-id="task.id" :hints="task.hints" :solution="readme.hints" />
      <LabResultsTab
        v-else
        :check="check"
        :checking="checking"
        :attempts="attempts"
        :checked-at="checkedAt"
        :can-check="canCheck"
        :next-task="nextTask"
        :next-loading="nextLoading"
        @run="emit('run-check')"
        @next="emit('next')"
      />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch, type ComponentPublicInstance } from 'vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import TaskCover from '@/components/ui/TaskCover.vue'
import { uid } from '@/components/ui/utils'
import { difficultyMeta, taskNumber, taskVisual } from '@/constants/tasks'
import type { CheckResponse, TaskDetail, TaskSummary } from '@/types'
import LabHintsTab from './LabHintsTab.vue'
import LabResultsTab from './LabResultsTab.vue'
import LabTaskTab from './LabTaskTab.vue'
import { splitReadme } from './readme'
import type { LabTab } from './types'

const props = defineProps<{
  task: TaskDetail | null
  tab: LabTab
  check: CheckResponse | null
  checking: boolean
  attempts: number
  checkedAt: number | null
  canCheck: boolean
  nextTask: TaskSummary | null
  nextLoading?: boolean
}>()

const emit = defineEmits<{ 'update:tab': [tab: LabTab]; 'run-check': []; next: [] }>()

const TABS: { key: LabTab; label: string }[] = [
  { key: 'task', label: 'Topshiriq' },
  { key: 'hints', label: 'Maslahatlar' },
  { key: 'results', label: 'Natija' }
]

const prefix = uid('lab-tabs')
const tabId = (key: LabTab) => `${prefix}-tab-${key}`
const panelId = (key: LabTab) => `${prefix}-panel-${key}`

const visual = computed(() => taskVisual({ id: props.task?.id ?? '', category: props.task?.category }))
const difficulty = computed(() => difficultyMeta(props.task?.difficulty ?? 1))
const readme = computed(() => splitReadme(props.task?.readme))

const body = ref<HTMLElement | null>(null)
const tabRefs = new Map<LabTab, HTMLElement>()

function setTabRef(key: LabTab, el: Element | ComponentPublicInstance | null) {
  if (el instanceof HTMLElement) tabRefs.set(key, el)
  else tabRefs.delete(key)
}

function select(key: LabTab, focusTab = false) {
  emit('update:tab', key)
  if (focusTab) nextTick(() => tabRefs.get(key)?.focus())
}

function onTabKeydown(e: KeyboardEvent) {
  const idx = TABS.findIndex((t) => t.key === props.tab)
  let next = -1
  if (e.key === 'ArrowRight') next = (idx + 1) % TABS.length
  else if (e.key === 'ArrowLeft') next = (idx - 1 + TABS.length) % TABS.length
  else if (e.key === 'Home') next = 0
  else if (e.key === 'End') next = TABS.length - 1
  if (next < 0) return
  e.preventDefault()
  select(TABS[next].key, true)
}

// Each tab starts at the top of its content.
watch(
  () => props.tab,
  () => nextTick(() => body.value?.scrollTo({ top: 0 }))
)
</script>
