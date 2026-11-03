<!--
  LeaderboardTable — ranked list: rank, avatar + username, completed tasks (segment cells), XP.
  The current user's row is highlighted; when the user is outside the returned list, their row
  is appended after a "…" separator.
  Props:
    entries: LeaderboardEntry[]
    meId?: number | null
    outsideMe?: LeaderboardEntry | null   (current user's entry when not in `entries`)
    totalTasks?: number | null            (cells to draw; null/0 = show only the count)
    loading?: boolean                     (skeleton rows)
-->
<template>
  <section
    aria-labelledby="leaderboard-table-title"
    class="overflow-hidden rounded-panel border border-border bg-surface backdrop-blur-[6px]"
  >
    <div class="overflow-x-auto">
      <table class="w-full min-w-[580px] border-collapse" :aria-busy="loading || undefined">
        <caption id="leaderboard-table-title" class="sr-only">Reyting jadvali</caption>
        <thead>
          <tr class="th text-left">
            <th scope="col" class="w-16 px-6 py-3 font-medium">O'rin</th>
            <th scope="col" class="px-4 py-3 font-medium">O'quvchi</th>
            <th scope="col" class="px-4 py-3 font-medium">Vazifalar</th>
            <th scope="col" class="py-3 pl-4 pr-6 text-right font-medium">XP</th>
          </tr>
        </thead>
        <tbody v-if="loading">
          <tr v-for="i in 8" :key="i" class="border-t border-divider" aria-hidden="true">
            <td class="px-6 py-[11px]"><Skeleton width="22px" height="14px" /></td>
            <td class="px-4 py-[11px]">
              <span class="flex items-center gap-2.5">
                <Skeleton width="30px" height="30px" rounded="rounded-full" />
                <Skeleton width="110px" height="14px" />
              </span>
            </td>
            <td class="px-4 py-[11px]"><Skeleton width="130px" height="10px" /></td>
            <td class="py-[11px] pl-4 pr-6"><Skeleton width="40px" height="14px" class="ml-auto" /></td>
          </tr>
        </tbody>
        <tbody v-else>
          <LeaderboardRow
            v-for="entry in entries"
            :key="entry.user_id"
            :entry="entry"
            :is-me="entry.user_id === meId"
            :total-tasks="totalTasks"
          />
          <template v-if="outsideMe">
            <tr class="border-t border-divider">
              <td colspan="4" class="px-6 py-1.5 text-center font-mono text-muted">
                <span aria-hidden="true">⋯</span>
                <span class="sr-only">Siz ro'yxatdagi o'rinlardan pastdasiz</span>
              </td>
            </tr>
            <LeaderboardRow :entry="outsideMe" is-me :total-tasks="totalTasks" />
          </template>
        </tbody>
      </table>
    </div>
  </section>
</template>

<script setup lang="ts">
import Skeleton from '@/components/ui/Skeleton.vue'
import type { LeaderboardEntry } from '@/types'
import LeaderboardRow from './LeaderboardRow.vue'

defineProps<{
  entries: LeaderboardEntry[]
  meId?: number | null
  outsideMe?: LeaderboardEntry | null
  totalTasks?: number | null
  loading?: boolean
}>()
</script>
