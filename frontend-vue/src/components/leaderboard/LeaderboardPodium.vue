<!--
  LeaderboardPodium — top-3 podium (2nd · 1st · 3rd visually; DOM order is 1st, 2nd, 3rd so
  screen readers read it in rank order). Missing places (fewer than 3 entries) render as an
  empty dashed slot.
  Props:
    entries: LeaderboardEntry[]   (already ranked; only the first 3 are used)
    meId?: number | null          (current user id, marked "(siz)")
    loading?: boolean             (skeleton cards)
-->
<template>
  <section aria-labelledby="podium-title" :aria-busy="loading || undefined">
    <h2 id="podium-title" class="sr-only">Eng yaxshi uchlik</h2>
    <ol class="grid grid-cols-3 items-end gap-2.5 sm:gap-3.5">
      <li
        v-for="slot in slots"
        :key="slot.place"
        :class="['row-start-1 min-w-0', COLUMN[slot.place]]"
      >
        <div
          v-if="loading"
          :class="['flex flex-col items-center gap-2 rounded-t-panel border border-b-0 border-border bg-[rgba(15,21,32,0.88)] px-3', slot.place === 1 ? 'min-h-[200px] pt-[26px]' : 'min-h-[168px] pt-[18px]']"
          aria-hidden="true"
        >
          <Skeleton :width="slot.place === 1 ? '60px' : '50px'" :height="slot.place === 1 ? '60px' : '50px'" rounded="rounded-full" />
          <Skeleton width="70%" height="14px" />
          <Skeleton width="50%" height="22px" />
        </div>

        <div
          v-else-if="slot.entry"
          :class="[
            'flex flex-col items-center gap-1.5 rounded-t-panel border border-b-0 border-border bg-[rgba(15,21,32,0.88)] px-2 text-center sm:px-3',
            slot.entry.rank === 1 ? 'min-h-[200px] pt-[26px]' : 'min-h-[168px] pt-[18px]'
          ]"
        >
          <span
            :class="[
              'flex shrink-0 items-center justify-center rounded-full border-2 bg-surface-3 text-[15px] font-bold',
              slot.entry.rank === 1 ? 'h-[60px] w-[60px]' : 'h-[50px] w-[50px]'
            ]"
            :style="{ borderColor: color(slot.entry), color: color(slot.entry), boxShadow: `0 0 18px ${tint(color(slot.entry), 0.33)}` }"
            aria-hidden="true"
          >
            {{ initials(slot.entry.username) }}
          </span>
          <span class="w-full truncate font-semibold" :title="slot.entry.username">
            <span class="sr-only">{{ slot.entry.rank }}-o'rin: </span>
            {{ slot.entry.username }}
            <span v-if="slot.entry.user_id === meId" class="font-normal text-muted">(siz)</span>
          </span>
          <span class="font-mono text-lg font-bold text-text sm:text-xl">
            {{ slot.entry.score }} <span class="text-xs font-normal text-muted">xp</span>
          </span>
          <span
            class="mt-auto self-stretch rounded-t-lg py-2 font-mono text-[15px] font-bold text-bg"
            :style="{ background: color(slot.entry) }"
            aria-hidden="true"
          >#{{ slot.entry.rank }}</span>
        </div>

        <div
          v-else
          class="flex min-h-[168px] flex-col items-center justify-center gap-1 rounded-t-panel border border-b-0 border-dashed border-border-strong px-3 text-center"
        >
          <span class="font-mono text-[15px] font-bold text-faint" aria-hidden="true">#{{ slot.place }}</span>
          <span class="text-[13px] text-muted">
            <span class="sr-only">{{ slot.place }}-o'rin: </span>Bo'sh
          </span>
        </div>
      </li>
    </ol>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import { tint } from '@/components/ui/utils'
import type { LeaderboardEntry } from '@/types'
import { initials, medalColor } from './leaderboard'

const props = defineProps<{
  entries: LeaderboardEntry[]
  meId?: number | null
  loading?: boolean
}>()

type Place = 1 | 2 | 3

/** Visual columns: 2nd left, 1st center, 3rd right. */
const COLUMN: Record<Place, string> = {
  1: 'col-start-2',
  2: 'col-start-1',
  3: 'col-start-3'
}

const slots = computed(() =>
  ([1, 2, 3] as const).map((place) => ({ place, entry: props.entries[place - 1] as LeaderboardEntry | undefined }))
)

function color(entry: LeaderboardEntry): string {
  return medalColor(entry.rank)
}
</script>
