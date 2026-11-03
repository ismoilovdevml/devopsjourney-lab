<!--
  AchievementGrid — computed badges (GET /api/progress .achievements) with a Hammasi / Olingan filter.
  Earned = colored tile with the date it was earned; locked = dashed outline.
  Props:
    achievements: Achievement[]
    loading?: boolean
-->
<template>
  <section
    aria-labelledby="achievements-title"
    class="flex min-w-0 flex-[999_1_560px] flex-col gap-4 rounded-panel border border-border bg-surface px-5 py-5 backdrop-blur-[6px] sm:px-6"
  >
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h2 id="achievements-title" class="font-mono text-[15px] font-semibold text-text">
        Yutuqlar
        <span v-if="!loading" class="font-normal text-muted">
          <span aria-hidden="true">{{ earnedCount }}/{{ achievements.length }}</span>
          <span class="sr-only">, {{ achievements.length }} tadan {{ earnedCount }} tasi olingan</span>
        </span>
      </h2>
      <SegmentedControl v-model="filter" :options="FILTERS" aria-label="Yutuqlar filtri" size="sm" />
    </div>

    <ul v-if="loading" class="grid grid-cols-[repeat(auto-fill,minmax(210px,1fr))] gap-2.5" aria-hidden="true">
      <li v-for="i in 6" :key="i" class="flex items-center gap-3 rounded-[12px] border border-border px-3.5 py-3">
        <Skeleton width="38px" height="38px" rounded="rounded-btn" />
        <span class="flex flex-1 flex-col gap-1.5"><Skeleton width="70%" /><Skeleton width="90%" height="12px" /></span>
      </li>
    </ul>

    <EmptyState
      v-else-if="visible.length === 0"
      :icon="Award"
      :title="filter === 'earned' ? 'Hali yutuq olinmagan' : 'Yutuqlar mavjud emas'"
      :description="filter === 'earned' ? 'Vazifalarni yakunlab, birinchi yutug\'ingizni oling.' : undefined"
    />

    <ul v-else class="grid grid-cols-[repeat(auto-fill,minmax(210px,1fr))] gap-2.5">
      <li
        v-for="a in visible"
        :key="a.id"
        :class="[
          'flex items-center gap-3 rounded-[12px] px-3.5 py-3',
          a.earned ? 'border bg-surface-2' : 'border border-dashed border-border-stronger'
        ]"
        :style="a.earned ? { borderColor: tint(a.meta.color, 0.33) } : undefined"
      >
        <span
          :class="['flex h-[38px] w-[38px] shrink-0 items-center justify-center rounded-btn', a.earned ? '' : 'bg-surface-3 text-muted']"
          :style="a.earned ? { color: a.meta.color, background: tint(a.meta.color, 0.12), boxShadow: `0 0 14px ${tint(a.meta.color, 0.25)}` } : undefined"
          aria-hidden="true"
        >
          <component :is="a.meta.icon" class="h-[18px] w-[18px]" />
        </span>
        <span class="min-w-0 flex-1">
          <span class="block font-semibold" :class="a.earned ? 'text-text' : 'text-text-2'">{{ a.meta.title }}</span>
          <span class="block text-meta text-muted">{{ a.meta.description }}</span>
          <span v-if="a.earned" class="mt-0.5 block font-mono text-xs" :style="{ color: a.meta.color }">
            <span class="sr-only">Olingan</span>
            <template v-if="a.date"><span aria-hidden="true">✓ </span>{{ a.date }}</template>
            <span v-else aria-hidden="true">✓ olingan</span>
          </span>
          <span v-else class="sr-only">Hali olinmagan</span>
        </span>
      </li>
    </ul>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Award } from 'lucide-vue-next'
import EmptyState from '@/components/ui/EmptyState.vue'
import SegmentedControl from '@/components/ui/SegmentedControl.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import { tint } from '@/components/ui/utils'
import type { Achievement } from '@/types'
import { achievementMeta, formatDayMonth } from './format'

const props = defineProps<{
  achievements: Achievement[]
  loading?: boolean
}>()

type Filter = 'all' | 'earned'

const FILTERS: { value: Filter; label: string }[] = [
  { value: 'all', label: 'Hammasi' },
  { value: 'earned', label: 'Olingan' }
]

const filter = ref<Filter>('all')

const earnedCount = computed(() => props.achievements.filter((a) => a.earned).length)

const visible = computed(() =>
  props.achievements
    .map((a, i) => ({
      id: a.id,
      earned: a.earned,
      meta: achievementMeta(a, i),
      date: a.earned ? formatDayMonth(a.earned_at) : null
    }))
    .filter((a) => filter.value === 'all' || a.earned)
)
</script>
