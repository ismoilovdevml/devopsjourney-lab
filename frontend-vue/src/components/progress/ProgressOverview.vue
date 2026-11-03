<!--
  ProgressOverview — conic completion ring, total XP and 4 mini stats.
  Props:
    progress?: ProgressResponse   (omit while loading)
    totalAttempts?: number        (sum of progress[].attempts)
    loading?: boolean
-->
<template>
  <section
    aria-label="Umumiy ko'rsatkichlar"
    class="flex flex-wrap items-center gap-6 rounded-panel border border-border bg-surface px-5 py-6 backdrop-blur-[6px] sm:gap-8 sm:px-7 sm:py-[26px]"
  >
    <div class="flex min-w-0 flex-[1_1_280px] items-center gap-5 sm:gap-[22px]">
      <Skeleton v-if="loading || !progress" width="116px" height="116px" rounded="rounded-full" class="shrink-0" />
      <div
        v-else
        role="img"
        :aria-label="`${pct} foiz yakunlandi`"
        class="flex h-[116px] w-[116px] shrink-0 items-center justify-center rounded-full"
        :style="{ background: `conic-gradient(#3DDC97 0 ${pct}%, #1A2332 ${pct}% 100%)` }"
      >
        <div class="flex h-[92px] w-[92px] items-center justify-center rounded-full bg-surface-solid">
          <span class="font-mono text-2xl font-bold text-text" aria-hidden="true">{{ pct }}%</span>
        </div>
      </div>

      <div class="flex min-w-0 flex-col gap-1">
        <span class="text-muted">Umumiy progress</span>
        <template v-if="loading || !progress">
          <Skeleton width="150px" height="26px" />
          <Skeleton width="120px" height="14px" />
        </template>
        <template v-else>
          <span class="font-mono text-[22px] font-bold text-accent">
            {{ progress.totalScore }} / {{ progress.totalPossible }} xp
          </span>
          <span class="text-[13px] text-muted">{{ remainingLabel }}</span>
        </template>
      </div>
    </div>

    <dl class="grid min-w-0 flex-[2_1_480px] grid-cols-2 gap-3 md:grid-cols-4">
      <div v-for="stat in stats" :key="stat.label" class="rounded-[12px] border border-border bg-surface-2 px-4 py-3.5">
        <dt class="text-meta text-muted">{{ stat.label }}</dt>
        <dd class="mt-0.5">
          <Skeleton v-if="loading || !progress" width="56px" height="28px" />
          <template v-else>
            <span class="font-mono text-[22px] font-bold" :style="{ color: stat.color }" :aria-hidden="stat.srText ? 'true' : undefined">
              {{ stat.value }}
            </span>
            <span v-if="stat.srText" class="sr-only">{{ stat.srText }}</span>
          </template>
        </dd>
      </div>
    </dl>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import type { ProgressResponse } from '@/types'
import { formatDurationLong, formatDurationShort } from './format'

const props = defineProps<{
  progress?: ProgressResponse
  totalAttempts?: number
  loading?: boolean
}>()

interface MiniStat {
  label: string
  value: string
  color: string
  srText?: string
}

const pct = computed(() => Math.min(100, Math.max(0, Math.round(props.progress?.percentage ?? 0))))

const remainingLabel = computed(() => {
  const p = props.progress
  if (!p) return ''
  if (p.totalTasks > 0 && p.completed >= p.totalTasks) return 'Barcha vazifalar yakunlandi'
  const left = Math.max(0, p.totalPossible - p.totalScore)
  return `yana ${left} xp to'plash mumkin`
})

const stats = computed<MiniStat[]>(() => {
  const p = props.progress
  return [
    {
      label: 'Yakunlangan',
      value: p ? `${p.completed}/${p.totalTasks}` : '',
      color: '#3DDC97',
      srText: p ? `${p.totalTasks} tadan ${p.completed} ta vazifa` : undefined
    },
    { label: 'Urinishlar', value: String(props.totalAttempts ?? 0), color: '#38BDF8' },
    {
      label: 'Terminalda',
      value: formatDurationShort(p?.timeInLabSeconds ?? 0),
      color: '#2DD4BF',
      srText: formatDurationLong(p?.timeInLabSeconds ?? 0)
    },
    {
      label: 'Ketma-ket kunlar',
      value: String(p?.streakDays ?? 0),
      color: '#F5B342'
    }
  ]
})
</script>
