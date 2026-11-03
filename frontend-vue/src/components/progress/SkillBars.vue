<!--
  SkillBars — completion percent per task category (GET /api/progress .skills), glowing bars.
  Props:
    skills: SkillProgress[]
    loading?: boolean
-->
<template>
  <section
    aria-labelledby="skills-title"
    class="flex min-w-0 flex-[1_1_340px] flex-col gap-4 rounded-panel border border-border bg-surface px-5 py-5 backdrop-blur-[6px] sm:px-6"
  >
    <h2 id="skills-title" class="font-mono text-[15px] font-semibold text-text">Ko'nikmalar</h2>

    <template v-if="loading">
      <div v-for="i in 4" :key="i" class="flex flex-col gap-2">
        <div class="flex justify-between"><Skeleton width="110px" /><Skeleton width="36px" /></div>
        <Skeleton height="8px" rounded="rounded-full" />
      </div>
    </template>

    <p v-else-if="items.length === 0" class="text-[13px] text-muted">Ko'nikmalar bo'yicha ma'lumot yo'q.</p>

    <ul v-else class="flex flex-col gap-4">
      <li v-for="s in items" :key="s.category" class="flex flex-col gap-1.5">
        <div class="flex justify-between gap-3">
          <span :id="`skill-${s.category}`" class="text-text">{{ s.label }}</span>
          <span class="font-mono text-[13px] text-muted">{{ s.percent }}%</span>
        </div>
        <div
          class="h-2 overflow-hidden rounded-full bg-divider"
          role="progressbar"
          :aria-labelledby="`skill-${s.category}`"
          aria-valuemin="0"
          aria-valuemax="100"
          :aria-valuenow="s.percent"
          :aria-valuetext="`${s.percent}%`"
        >
          <div
            class="h-full rounded-full"
            :style="{ width: `${s.percent}%`, background: s.color, boxShadow: s.percent > 0 ? `0 0 10px ${tint(s.color, 0.5)}` : 'none' }"
          />
        </div>
      </li>
    </ul>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import { tint } from '@/components/ui/utils'
import { CATEGORY_VISUALS, categoryLabel } from '@/constants/tasks'
import type { SkillProgress } from '@/types'

const props = defineProps<{
  skills: SkillProgress[]
  loading?: boolean
}>()

const items = computed(() =>
  props.skills.map((s) => ({
    category: s.category,
    label: categoryLabel(s.category),
    percent: Math.min(100, Math.max(0, Math.round(s.percent))),
    color: CATEGORY_VISUALS[s.category]?.accent ?? '#94A3B8'
  }))
)
</script>
