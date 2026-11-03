<!--
  QuickLinks — "Tezkor havolalar" card: rank, achievements, lab session, settings.
  Props:
    rank?: number | null              (leaderboard me.rank; null/undefined -> "—")
    achievements?: { earned: number; total: number } | null   (null -> "tez orada")
    sessionLabel?: string | null      (remaining time of the running session, e.g. "47:12")
    sessionTaskId?: string | null     (lab link resumes this task)
    loading?: boolean
-->
<template>
  <nav
    aria-labelledby="dash-quick-title"
    class="rounded-panel border border-border bg-surface p-2 backdrop-blur-[6px]"
  >
    <h2 id="dash-quick-title" class="mx-3 mb-1.5 mt-2.5 font-mono text-sm font-semibold text-text">Tezkor havolalar</h2>
    <ul class="m-0 list-none p-0">
      <li v-for="link in links" :key="link.label">
        <RouterLink
          :to="link.to"
          class="flex items-center justify-between gap-3 rounded-btn p-3 text-text transition-colors hover:bg-surface-3 hover:text-text"
        >
          <span>{{ link.label }}</span>
          <Skeleton v-if="loading && link.async" width="48px" height="14px" />
          <span v-else :class="['shrink-0', link.mono ? 'font-mono' : '', link.valueClass]">{{ link.value }}</span>
        </RouterLink>
      </li>
    </ul>
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { RouteLocationRaw } from 'vue-router'
import Skeleton from '@/components/ui/Skeleton.vue'

const props = defineProps<{
  rank?: number | null
  achievements?: { earned: number; total: number } | null
  sessionLabel?: string | null
  sessionTaskId?: string | null
  loading?: boolean
}>()

interface QuickLink {
  label: string
  value: string
  to: RouteLocationRaw
  valueClass: string
  mono: boolean
  async: boolean
}

const links = computed<QuickLink[]>(() => [
  {
    label: "Reytingdagi o'rningiz",
    value: props.rank ? `#${props.rank}` : '—',
    to: { name: 'leaderboard' },
    valueClass: 'text-warn',
    mono: true,
    async: true
  },
  {
    label: 'Yutuqlar',
    value: props.achievements ? `${props.achievements.earned} / ${props.achievements.total}` : 'tez orada',
    to: { name: 'progress' },
    valueClass: props.achievements ? 'text-violet' : 'text-[13px] text-muted',
    mono: !!props.achievements,
    async: true
  },
  {
    label: 'Laboratoriya',
    value: props.sessionLabel ?? "sessiya yo'q",
    to: props.sessionTaskId ? { name: 'lab', params: { taskId: props.sessionTaskId } } : { name: 'lab' },
    valueClass: props.sessionLabel ? 'text-accent' : 'text-[13px] text-muted',
    mono: !!props.sessionLabel,
    async: false
  },
  {
    label: 'Sozlamalar',
    value: 'Mavzu, til',
    to: { name: 'profile' },
    valueClass: 'text-muted',
    mono: false,
    async: false
  }
])
</script>
