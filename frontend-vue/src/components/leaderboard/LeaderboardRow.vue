<!--
  LeaderboardRow — one <tr> of the leaderboard table.
  Props:
    entry: LeaderboardEntry
    isMe?: boolean              (green tint + left bar, "(siz)" suffix, aria-current)
    totalTasks?: number | null  (segment cells; null/0 = show only the completed count)
-->
<template>
  <tr
    :class="['border-t border-divider', isMe ? 'bg-accent/[0.08] shadow-[inset_3px_0_0_#3DDC97]' : '']"
    :aria-current="isMe ? 'true' : undefined"
  >
    <td class="px-6 py-[11px] font-mono text-[13px] font-semibold text-muted">
      {{ rankLabel(entry.rank) }}
    </td>
    <th scope="row" class="px-4 py-[11px] text-left font-normal">
      <span class="flex items-center gap-2.5">
        <span
          class="flex h-[30px] w-[30px] shrink-0 items-center justify-center rounded-full border border-border-strong bg-surface-3 text-[11px] font-semibold text-text-2"
          aria-hidden="true"
        >
          {{ initials(entry.username) }}
        </span>
        <span class="min-w-0 truncate font-mono font-medium text-text">
          {{ entry.username }}<span v-if="isMe" class="text-muted"> (siz)</span>
        </span>
      </span>
    </th>
    <td class="px-4 py-[11px]">
      <span class="flex items-center gap-2">
        <span v-if="cells > 0" class="shrink-0" :style="{ width: `${cells * 14 + (cells - 1) * 3}px` }">
          <ProgressSegments
            :total="cells"
            :filled="done"
            :height="8"
            :gap="3"
            :label="`Bajarilgan vazifalar: ${done} / ${cells}`"
          />
        </span>
        <span class="font-mono text-[13px] text-muted">
          <template v-if="cells > 0"><span aria-hidden="true">{{ done }}/{{ cells }}</span></template>
          <template v-else>{{ entry.completed }}</template>
        </span>
      </span>
    </td>
    <td class="py-[11px] pl-4 pr-6 text-right font-mono text-[13px] font-semibold text-accent">{{ entry.score }}</td>
  </tr>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import ProgressSegments from '@/components/ui/ProgressSegments.vue'
import type { LeaderboardEntry } from '@/types'
import { initials, rankLabel } from './leaderboard'

const props = defineProps<{
  entry: LeaderboardEntry
  isMe?: boolean
  totalTasks?: number | null
}>()

const cells = computed(() => Math.max(0, Math.floor(props.totalTasks ?? 0)))
const done = computed(() => Math.min(cells.value, Math.max(0, props.entry.completed)))
</script>
