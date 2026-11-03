<!--
  ProgressSegments — n equal segments (check progress, path indicator, leaderboard cells).
  Props:
    total: number                 (segment count)
    filled?: number               (first n segments are "done"; ignored when `states` is given)
    states?: ('done' | 'active' | 'fail' | 'empty')[]   (explicit per-segment state)
    color?: string                (hex for done segments, default #3DDC97)
    height?: number               (px, default 6)
    gap?: number                  (px, default 4)
    label?: string                (accessible name, e.g. "Tekshiruvlar: 2 / 4")
-->
<template>
  <div
    class="flex w-full"
    :style="{ gap: `${gap}px` }"
    role="progressbar"
    :aria-label="label"
    aria-valuemin="0"
    :aria-valuemax="total"
    :aria-valuenow="doneCount"
  >
    <span
      v-for="(state, i) in segmentStates"
      :key="i"
      class="flex-1 rounded-full"
      :style="{ height: `${height}px`, background: bg(state), boxShadow: state === 'active' ? `0 0 0 1px ${INFO} inset` : undefined }"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

type SegmentState = 'done' | 'active' | 'fail' | 'empty'

const props = withDefaults(
  defineProps<{
    total: number
    filled?: number
    states?: SegmentState[]
    color?: string
    height?: number
    gap?: number
    label?: string
  }>(),
  { filled: 0, color: '#3DDC97', height: 6, gap: 4 }
)

const INFO = '#38BDF8'

const segmentStates = computed<SegmentState[]>(() => {
  const n = Math.max(0, Math.floor(props.total))
  if (props.states) return Array.from({ length: n }, (_, i) => props.states?.[i] ?? 'empty')
  return Array.from({ length: n }, (_, i) => (i < props.filled ? 'done' : 'empty'))
})

const doneCount = computed(() => segmentStates.value.filter((s) => s === 'done').length)

function bg(state: SegmentState): string {
  if (state === 'done') return props.color
  if (state === 'active') return 'rgba(56,189,248,0.35)'
  if (state === 'fail') return '#F87171'
  return '#243247'
}
</script>
