<!--
  LeaderboardMeBanner — "Sizning o'rningiz": the current user's rank, score and the XP needed for
  the next rank. Without points (me null or score 0) it invites the user to solve a first task.
  Props:
    me: LeaderboardEntry | null
    gap?: number | null     (XP to the next rank; hidden when null)
    loading?: boolean       (skeleton)
-->
<template>
  <section
    aria-labelledby="me-banner-title"
    class="flex flex-wrap items-center gap-x-5 gap-y-3 rounded-card border border-accent/35 bg-accent/[0.08] px-[22px] py-[18px]"
    :aria-busy="loading || undefined"
  >
    <template v-if="loading">
      <h2 id="me-banner-title" class="sr-only">Sizning o'rningiz</h2>
      <Skeleton width="64px" height="36px" />
      <div class="flex min-w-[200px] flex-1 flex-col gap-2">
        <Skeleton width="140px" height="14px" />
        <Skeleton width="200px" height="12px" />
      </div>
    </template>

    <template v-else-if="hasPoints && me">
      <span class="font-mono text-[30px] font-bold leading-none text-accent" aria-hidden="true">#{{ me.rank }}</span>
      <div class="min-w-[200px] flex-1">
        <h2 id="me-banner-title" class="font-semibold text-text">
          Sizning o'rningiz<span class="sr-only">: {{ me.rank }}</span>
        </h2>
        <p class="text-[13px] text-text-2">
          <span class="font-mono">{{ me.score }}</span> xp
          <template v-if="me.rank === 1"> · siz birinchi o'rindasiz</template>
          <template v-else-if="gap !== null && gap !== undefined">
            · keyingi o'ringacha <span class="font-mono">{{ gap }}</span> xp
          </template>
        </p>
      </div>
      <Button variant="primary" to="/tasks">Ball yig'ish</Button>
    </template>

    <template v-else>
      <span class="font-mono text-[30px] font-bold leading-none text-muted" aria-hidden="true">#—</span>
      <div class="min-w-[200px] flex-1">
        <h2 id="me-banner-title" class="font-semibold text-text">Sizning o'rningiz</h2>
        <p class="text-[13px] text-text-2">Hali ball yo'q — birinchi vazifani bajaring</p>
      </div>
      <Button variant="primary" to="/tasks">Vazifalarga o'tish</Button>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import Button from '@/components/ui/Button.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import type { LeaderboardEntry } from '@/types'

const props = defineProps<{
  me: LeaderboardEntry | null
  gap?: number | null
  loading?: boolean
}>()

const hasPoints = computed(() => props.me !== null && props.me.score > 0)
</script>
