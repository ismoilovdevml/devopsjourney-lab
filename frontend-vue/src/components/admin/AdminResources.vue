<!--
  AdminResources — container capacity meter + aggregate CPU / RAM of lab containers, and per-session
  20-cell CPU / memory meters. CPU % is docker's value (100% = one core), clamped to 100 on meters.
  Data: GET /api/admin/resources
  Exposes: refresh()
-->
<template>
  <SectionError v-if="error && !data" :message="error" command="GET /api/admin/resources" :retrying="loading" @retry="refresh" />

  <div v-else class="flex flex-col gap-5">
    <div class="grid grid-cols-1 gap-3.5 md:grid-cols-3">
      <div
        v-for="c in cards"
        :key="c.label"
        class="flex flex-col gap-2.5 rounded-card border border-border bg-[rgba(15,21,32,0.88)] px-5 py-[18px]"
      >
        <div class="flex items-baseline justify-between gap-3">
          <span class="font-mono text-muted">{{ c.label }}</span>
          <Skeleton v-if="!data" width="70px" height="22px" />
          <span v-else class="font-mono text-lg font-bold text-text">{{ c.value }}</span>
        </div>
        <Skeleton v-if="!data" height="16px" />
        <CellMeter v-else :percent="c.percent" :color="c.color" :label="c.aria" :value-text="c.value" :height="16" />
        <span class="text-meta text-muted">{{ c.note }}</span>
      </div>
    </div>

    <section aria-labelledby="adm-res-sessions" class="overflow-hidden rounded-panel border border-border bg-[rgba(15,21,32,0.88)]">
      <div class="flex items-center justify-between gap-3 border-b border-divider px-5 py-3.5">
        <h2 id="adm-res-sessions" class="font-mono text-sm font-semibold text-text">Sessiyalar bo'yicha</h2>
        <span class="font-mono text-xs text-muted">docker stats</span>
      </div>

      <div v-if="data && data.sessions.length === 0" class="p-4">
        <EmptyState :icon="Cpu" command="docker stats → 0" title="Ishlayotgan konteynerlar yo'q" description="Faol sessiya boshlanganda resurs sarfi shu yerda ko'rinadi." />
      </div>

      <div v-else class="relative overflow-x-auto">
        <table class="w-full min-w-[760px] border-collapse">
          <caption class="sr-only">Har bir sessiya konteynerining CPU va xotira sarfi</caption>
          <thead>
            <tr class="th text-left">
              <th scope="col" class="px-5 py-3 font-medium">O'quvchi</th>
              <th scope="col" class="px-4 py-3 font-medium">Vazifa</th>
              <th scope="col" class="w-[30%] px-4 py-3 font-medium">CPU</th>
              <th scope="col" class="w-[30%] py-3 pl-4 pr-5 font-medium">RAM</th>
            </tr>
          </thead>
          <tbody v-if="!data">
            <tr v-for="i in 4" :key="i" class="border-t border-divider">
              <td class="px-5 py-3"><Skeleton width="90px" /></td>
              <td class="px-4 py-3"><Skeleton width="140px" /></td>
              <td class="px-4 py-3"><Skeleton height="14px" /></td>
              <td class="py-3 pl-4 pr-5"><Skeleton height="14px" /></td>
            </tr>
          </tbody>
          <tbody v-else>
            <tr v-for="s in sessionRows" :key="s.session_id" class="border-t border-divider">
              <th scope="row" class="px-5 py-3 text-left font-mono font-medium text-text">
                {{ s.username }}
                <span class="block font-sans text-xs font-normal text-faint" :title="s.session_id">{{ s.shortId }}</span>
              </th>
              <td class="px-4 py-3 text-text-2">{{ s.taskTitle }}</td>
              <td class="px-4 py-3">
                <div class="flex items-center gap-3">
                  <CellMeter class="flex-1" :percent="s.cpu" color="#3DDC97" :label="`CPU, ${s.username}`" :value-text="s.cpuLabel" />
                  <span class="w-14 shrink-0 text-right font-mono text-[13px] text-text">{{ s.cpuLabel }}</span>
                </div>
              </td>
              <td class="py-3 pl-4 pr-5">
                <div class="flex items-center gap-3">
                  <CellMeter class="flex-1" :percent="s.memPct" color="#38BDF8" :label="`RAM, ${s.username}`" :value-text="s.memLabel" />
                  <span class="w-[92px] shrink-0 text-right font-mono text-[13px] text-text">{{ s.memLabel }}</span>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Cpu } from 'lucide-vue-next'
import EmptyState from '@/components/ui/EmptyState.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import CellMeter from './CellMeter.vue'
import SectionError from './SectionError.vue'
import { bytes, percentLabel } from './format'
import { api, errorMessage } from '@/services/api'
import { useTaskStore } from '@/stores/task'
import { useToast } from '@/composables/useToast'
import type { AdminResources } from '@/types'

const tasks = useTaskStore()
const toast = useToast()

const data = ref<AdminResources | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)

const memPercent = (used: number, limit: number) => (limit > 0 ? (used / limit) * 100 : 0)

const cards = computed(() => {
  const d = data.value
  const sessions = d?.sessions ?? []
  const running = d?.containers_running ?? 0
  const cap = d?.containers_limit ?? 0
  const cpu = sessions.reduce((sum, s) => sum + (Number.isFinite(s.cpu_percent) ? s.cpu_percent : 0), 0)
  const mem = sessions.reduce((sum, s) => sum + (s.mem_bytes || 0), 0)
  const memCap = sessions.reduce((sum, s) => sum + (s.mem_limit_bytes || 0), 0)
  const avgCpu = sessions.length ? cpu / sessions.length : 0
  return [
    {
      label: 'containers',
      value: `${running}/${cap || '—'}`,
      percent: cap > 0 ? (running / cap) * 100 : 0,
      note: cap > 0 ? `limit: ${cap} · bo'sh: ${Math.max(0, cap - running)}` : 'limit noma\'lum',
      color: '#F5B342',
      aria: 'Ishlayotgan konteynerlar limitga nisbatan'
    },
    {
      label: 'cpu · o\'rtacha',
      value: percentLabel(avgCpu),
      percent: avgCpu,
      note: sessions.length ? `${sessions.length} konteyner · jami ${percentLabel(cpu)} (100% = 1 yadro)` : 'konteynerlar yo\'q',
      color: '#3DDC97',
      aria: "Konteynerlar o'rtacha CPU yuklamasi"
    },
    {
      label: 'memory',
      value: memCap > 0 ? `${bytes(mem)}/${bytes(memCap)}` : bytes(mem),
      percent: memPercent(mem, memCap),
      note: memCap > 0 ? `konteyner limitlaridan ${percentLabel(memPercent(mem, memCap))}` : 'limit noma\'lum',
      color: '#38BDF8',
      aria: 'Konteynerlar xotira sarfi limitlarga nisbatan'
    }
  ]
})

const sessionRows = computed(() =>
  (data.value?.sessions ?? []).map((s) => ({
    ...s,
    shortId: s.session_id.length > 14 ? `${s.session_id.slice(0, 12)}…` : s.session_id,
    taskTitle: tasks.byId(s.task_id)?.title ?? s.task_id,
    cpu: s.cpu_percent,
    cpuLabel: percentLabel(s.cpu_percent),
    memPct: memPercent(s.mem_bytes, s.mem_limit_bytes),
    memLabel: s.mem_limit_bytes > 0 ? `${bytes(s.mem_bytes)} / ${bytes(s.mem_limit_bytes)}` : bytes(s.mem_bytes)
  }))
)

async function refresh(): Promise<void> {
  loading.value = true
  error.value = null
  try {
    data.value = await api.adminResources()
  } catch (err) {
    error.value = errorMessage(err, "Resurslar ma'lumotini yuklab bo'lmadi.")
    if (data.value) toast.error(error.value)
  } finally {
    loading.value = false
  }
}

onMounted(() => void refresh())

defineExpose({ refresh, loading })
</script>
