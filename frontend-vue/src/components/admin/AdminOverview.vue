<!--
  AdminOverview — KPI tiles, 24h sessions bar chart and per-task completion bars.
  Data comes from the parent (GET /api/admin/stats is shared with the side-nav counts).
  Props:
    stats: AdminStats | null
    loading: boolean
    error: string | null
  Emits: retry
-->
<template>
  <SectionError v-if="error && !stats" :message="error" command="GET /api/admin/stats" :retrying="loading" @retry="emit('retry')" />

  <div v-else class="flex flex-col gap-5">
    <!-- KPI tiles -->
    <div class="grid grid-cols-1 gap-3.5 min-[480px]:grid-cols-2 xl:grid-cols-4">
      <div
        v-for="k in kpis"
        :key="k.label"
        class="flex flex-col gap-1 rounded-card border border-border bg-[rgba(15,21,32,0.88)] px-[18px] py-4"
      >
        <span class="text-[13px] text-muted">{{ k.label }}</span>
        <Skeleton v-if="!stats" width="90px" height="32px" class="my-0.5" />
        <span v-else class="font-mono text-[26px] font-bold leading-tight" :style="{ color: k.color }">{{ k.value }}</span>
        <span class="text-meta text-muted">{{ k.note }}</span>
      </div>
    </div>

    <div class="flex flex-wrap items-stretch gap-5">
      <!-- Sessions over the last 24 hours -->
      <section
        aria-labelledby="adm-load-title"
        class="flex min-w-0 flex-[2_1_420px] flex-col gap-3 rounded-panel border border-border bg-[rgba(15,21,32,0.88)] px-[22px] py-[18px]"
      >
        <div class="flex items-center justify-between gap-3">
          <h2 id="adm-load-title" class="font-mono text-sm font-semibold text-text">Sessiyalar · 24 soat</h2>
          <span class="text-xs text-muted">soatbay</span>
        </div>

        <template v-if="!stats">
          <Skeleton height="140px" />
          <Skeleton height="12px" />
        </template>
        <template v-else>
          <div
            role="img"
            :aria-label="chartSummary"
            class="flex h-[140px] items-end gap-[3px] border-b border-border-strong pt-2 sm:gap-1"
          >
            <span
              v-for="b in bars"
              :key="b.index"
              class="block flex-1 rounded-t-[3px]"
              :style="{ height: b.height, background: b.current ? '#3DDC97' : 'rgba(61,220,151,.45)' }"
              :title="`${b.hour} — ${b.value} ta sessiya`"
            />
          </div>
          <div class="flex justify-between font-mono text-[11.5px] text-muted" aria-hidden="true">
            <span v-for="l in axisLabels" :key="l">{{ l }}</span>
          </div>
          <p class="text-meta text-muted">
            Jami <span class="font-mono text-text-2">{{ total24h }}</span> ta sessiya<template v-if="peak.value > 0">,
              eng ko'p <span class="font-mono text-text-2">{{ peak.value }}</span> ta ({{ peak.hour }})</template>.
          </p>
          <table class="sr-only">
            <caption>Soatlar bo'yicha yaratilgan sessiyalar</caption>
            <thead><tr><th scope="col">Soat</th><th scope="col">Sessiyalar</th></tr></thead>
            <tbody>
              <tr v-for="b in bars" :key="b.index"><th scope="row">{{ b.hour }}</th><td>{{ b.value }}</td></tr>
            </tbody>
          </table>
        </template>
      </section>

      <!-- Completion per task -->
      <section
        aria-labelledby="adm-funnel-title"
        class="flex min-w-0 flex-[1_1_320px] flex-col gap-3 rounded-panel border border-border bg-[rgba(15,21,32,0.88)] px-[22px] py-[18px]"
      >
        <div class="flex items-center justify-between gap-3">
          <h2 id="adm-funnel-title" class="font-mono text-sm font-semibold text-text">Yakunlash darajasi</h2>
          <span class="text-xs text-muted">o'quvchilar ulushi</span>
        </div>

        <template v-if="!stats">
          <Skeleton v-for="i in 6" :key="i" height="34px" />
        </template>
        <p v-else-if="completion.length === 0" class="py-6 text-center text-[13px] text-muted">Vazifalar topilmadi.</p>
        <ul v-else class="flex flex-col gap-3">
          <li v-for="c in completion" :key="c.id" class="flex items-center gap-3">
            <TaskCover :task-id="c.id" :category="c.category" aspect="auto" position="right" rounded="rounded-md" class="h-7 !w-14 shrink-0 border border-border" />
            <div class="flex min-w-0 flex-1 flex-col gap-1">
              <div class="flex items-baseline justify-between gap-2 text-[13px]">
                <span class="truncate text-text" :title="c.title">{{ c.title }}</span>
                <span class="shrink-0 font-mono text-muted">
                  <span class="text-faint">{{ c.passed }}/{{ totalUsers }} ·</span> {{ c.pctLabel }}
                </span>
              </div>
              <div
                class="block h-1.5 overflow-hidden rounded-full bg-divider"
                role="progressbar"
                :aria-label="`${c.title}: ${c.passed} / ${totalUsers} o'quvchi yakunlagan, ${c.attempts} ta urinish`"
                aria-valuemin="0"
                aria-valuemax="100"
                :aria-valuenow="Math.round(c.pct)"
              >
                <span class="block h-full rounded-full bg-info" :style="{ width: `${c.pct}%` }" />
              </div>
            </div>
          </li>
        </ul>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import TaskCover from '@/components/ui/TaskCover.vue'
import SectionError from './SectionError.vue'
import { clampPercent, count, percentLabel } from './format'
import { useTaskStore } from '@/stores/task'
import type { AdminStats } from '@/types'

const props = defineProps<{
  stats: AdminStats | null
  loading: boolean
  error: string | null
}>()
const emit = defineEmits<{ retry: [] }>()

const tasks = useTaskStore()

const totalUsers = computed(() => props.stats?.total_users ?? 0)

const kpis = computed(() => {
  const s = props.stats
  return [
    { label: "O'quvchilar", value: count(s?.total_users), note: "ro'yxatdan o'tgan", color: '#E6EDF3' },
    { label: 'Faol foydalanuvchilar', value: count(s?.active_users), note: "so'nggi 24 soatda", color: '#3DDC97' },
    { label: 'Faol sessiyalar', value: count(s?.active_sessions), note: 'ishlayotgan konteynerlar', color: '#38BDF8' },
    {
      label: 'Yakunlash darajasi',
      value: s ? percentLabel(s.completion_rate) : '—',
      note: `${count(s?.total_attempts)} ta tekshiruv`,
      color: '#F5B342'
    }
  ]
})

const pad = (n: number) => String(n).padStart(2, '0')

/** 24 buckets, index 23 = the current hour (see backend bucketSessions). */
const bars = computed(() => {
  const raw = props.stats?.sessions_24h ?? []
  const values = Array.from({ length: 24 }, (_, i) => {
    const v = Number(raw[raw.length - 24 + i] ?? 0)
    return Number.isFinite(v) && v > 0 ? v : 0
  })
  const max = Math.max(1, ...values)
  const nowHour = new Date().getHours()
  return values.map((value, index) => ({
    index,
    value,
    hour: `${pad((nowHour - (23 - index) + 48) % 24)}:00`,
    current: index === 23,
    height: value > 0 ? `${Math.max(4, Math.round((value / max) * 100))}%` : '2px'
  }))
})

const axisLabels = computed(() => [bars.value[0].hour, bars.value[6].hour, bars.value[12].hour, bars.value[18].hour, 'hozir'])

const total24h = computed(() => bars.value.reduce((sum, b) => sum + b.value, 0))

const peak = computed(() =>
  bars.value.reduce((best, b) => (b.value > best.value ? { value: b.value, hour: b.hour } : best), { value: 0, hour: '' })
)

const chartSummary = computed(() => {
  if (total24h.value === 0) return "Oxirgi 24 soatda sessiyalar yaratilmagan"
  return `Oxirgi 24 soatda jami ${total24h.value} ta sessiya, eng ko'p ${peak.value.value} ta soat ${peak.value.hour} da. Batafsil jadval quyida.`
})

const completion = computed(() => {
  const list = props.stats?.task_completion ?? []
  const users = totalUsers.value
  return list.map((tc) => {
    const task = tasks.byId(tc.task_id)
    const pct = users > 0 ? clampPercent((tc.passed / users) * 100) : 0
    return {
      id: tc.task_id,
      title: task?.title ?? tc.task_id,
      category: task?.category,
      passed: tc.passed,
      attempts: tc.attempts,
      pct,
      pctLabel: percentLabel(pct)
    }
  })
})
</script>
