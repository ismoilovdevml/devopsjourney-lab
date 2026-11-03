<!--
  CellMeter — 20-cell horizontal meter (admin resources).
  Props:
    percent: number     (0..100; clamped)
    color?: string      (hex for filled cells, default #3DDC97)
    label: string       (accessible name, e.g. "CPU, laylo.k")
    valueText?: string  (aria-valuetext, e.g. "92M / 256M")
    cells?: number      (default 20)
    height?: number     (cell height px, default 14)
-->
<template>
  <div
    class="flex w-full gap-[3px]"
    role="meter"
    :aria-label="label"
    aria-valuemin="0"
    aria-valuemax="100"
    :aria-valuenow="Math.round(value)"
    :aria-valuetext="valueText ?? `${Math.round(value)}%`"
  >
    <span
      v-for="i in cells"
      :key="i"
      class="block flex-1 rounded-[2px]"
      :style="{ height: `${height}px`, background: i <= filled ? color : '#1A2332' }"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { clampPercent } from './format'

const props = withDefaults(
  defineProps<{
    percent: number
    color?: string
    label: string
    valueText?: string
    cells?: number
    height?: number
  }>(),
  { color: '#3DDC97', cells: 20, height: 14 }
)

const value = computed(() => clampPercent(props.percent))
/** Any non-zero load lights at least one cell so tiny values stay visible. */
const filled = computed(() => {
  const n = Math.round((value.value / 100) * props.cells)
  return value.value > 0 && n === 0 ? 1 : n
})
</script>
