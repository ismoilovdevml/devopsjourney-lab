<!--
  TaskCard — selectable task tile: cover with number chip + status badge, title, category, level, +xp.
  Props:
    task: TaskSummary
    status: TaskStatus     (derived via taskStatus(); locked cards are dimmed but still selectable)
    selected?: boolean     (green border + ring; aria-pressed)
  Emits: select
-->
<template>
  <button
    type="button"
    :aria-pressed="selected"
    :class="[
      'group flex w-full flex-col items-stretch justify-start overflow-hidden rounded-card border bg-[rgba(15,21,32,0.9)] text-left transition-colors',
      selected ? 'border-accent shadow-selected' : 'border-border hover:border-border-stronger',
      status === 'locked' && !selected ? 'opacity-70 hover:opacity-90' : ''
    ]"
    @click="emit('select')"
  >
    <span class="relative block">
      <TaskCover :task-id="task.id" :category="task.category" />
      <span class="absolute left-2.5 top-2.5 rounded-chip bg-[rgba(10,14,20,0.75)] px-2 py-0.5 font-mono text-xs font-semibold text-text">
        {{ taskNumber(task.order) }}
      </span>
      <Badge :variant="status" solid class="absolute right-2.5 top-2.5" />
    </span>
    <span class="flex flex-1 flex-col gap-1.5 px-4 pb-4 pt-3.5">
      <span class="text-[15px] font-semibold text-text">{{ task.title }}</span>
      <span class="mt-auto flex flex-wrap items-center gap-2 text-xs text-muted">
        <span class="rounded-chip bg-surface-3 px-2 py-0.5">{{ visual.categoryLabel }}</span>
        <span class="rounded-chip bg-surface-3 px-2 py-0.5" :style="{ color: level.color }">{{ level.label }}</span>
        <span class="ml-auto font-mono text-accent">+{{ task.points }} xp</span>
      </span>
    </span>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import Badge from '@/components/ui/Badge.vue'
import TaskCover from '@/components/ui/TaskCover.vue'
import { difficultyMeta, taskNumber, taskVisual, type TaskStatus } from '@/constants/tasks'
import type { TaskSummary } from '@/types'

const props = defineProps<{
  task: TaskSummary
  status: TaskStatus
  selected?: boolean
}>()

const emit = defineEmits<{ select: [] }>()

const visual = computed(() => taskVisual(props.task))
const level = computed(() => difficultyMeta(props.task.difficulty))
</script>
