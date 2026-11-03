<!--
  TaskCover — cover SVG for a task (falls back by category, then to servers.svg).
  Props:
    taskId: string
    category?: string          (API task.category, used for unknown ids)
    alt?: string               (default '' = decorative)
    aspect?: string            (CSS aspect-ratio, default '320 / 140'; pass 'auto' to size via class)
    position?: 'center' | 'right'  (object-position; 'right' for small thumbnails)
    rounded?: string           (Tailwind radius class, default 'rounded-none')
  Size it with a class on the component, e.g. <TaskCover class="h-[46px] w-[104px]" aspect="auto" />.
-->
<template>
  <img
    :src="visual.cover"
    :alt="alt"
    loading="lazy"
    decoding="async"
    draggable="false"
    :class="['block w-full bg-surface-2 object-cover', rounded, position === 'right' ? 'object-right' : 'object-center']"
    :style="aspect !== 'auto' ? { aspectRatio: aspect } : undefined"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { taskVisual } from '@/constants/tasks'

const props = withDefaults(
  defineProps<{
    taskId: string
    category?: string
    alt?: string
    aspect?: string
    position?: 'center' | 'right'
    rounded?: string
  }>(),
  { alt: '', aspect: '320 / 140', position: 'center', rounded: 'rounded-none' }
)

const visual = computed(() => taskVisual({ id: props.taskId, category: props.category }))
</script>
