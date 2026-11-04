<!--
  AdminActivity — activity log rendered as a `journalctl` terminal.
  Level filter (client-side) and limit selector (100 / 250 / 500 -> ?limit=).
  `details` and `action` are user-influenced strings: they are rendered as plain text only.
  Data: GET /api/admin/activity?limit=n
  Exposes: refresh()
-->
<template>
  <div class="flex flex-col gap-3">
    <div class="flex flex-wrap items-center gap-3">
      <SegmentedControl v-model="level" :options="levelOptions" ariaLabel="Daraja bo'yicha filtr" size="sm" />
      <SegmentedControl v-model="limit" :options="LIMIT_OPTIONS" ariaLabel="Yozuvlar soni" size="sm" class="sm:ml-auto" />
    </div>

    <SectionError v-if="error && !loaded" :message="error" :command="`GET /api/admin/activity?limit=${limit}`" :retrying="loading" @retry="refresh" />

    <TerminalWindow v-else :title="`journalctl -u devops-lab -n ${limit}`" tone="terminal" :padded="false">
      <template #actions>
        <span class="whitespace-nowrap font-mono text-xs text-faint" aria-live="polite">
          <template v-if="loaded">{{ visible.length }} / {{ entries.length }}</template>
        </span>
      </template>

      <div class="relative max-h-[min(640px,70vh)] overflow-auto px-[18px] py-3">
        <div v-if="!loaded" class="flex flex-col gap-2.5 py-1">
          <Skeleton v-for="i in 10" :key="i" :width="`${55 + ((i * 17) % 40)}%`" height="13px" />
        </div>

        <p v-else-if="visible.length === 0" class="py-6 font-mono text-[13px] text-muted">
          -- No entries --
          <span class="block font-sans text-meta">
            {{ entries.length === 0 ? "Faollik jurnali hozircha bo'sh." : `${level} darajasidagi yozuvlar yo'q.` }}
          </span>
        </p>

        <ol v-else class="font-mono text-[13px] leading-[1.9]" aria-label="Faollik yozuvlari">
          <li v-for="e in visible" :key="e.id" class="break-words">
            <time class="text-faint" :datetime="e.created_at" :title="fullDateTime(e.created_at)">{{ logTime(e.created_at, today) }}</time>
            <span class="ml-3 inline-block w-[3.25rem] font-semibold" :class="LEVEL_CLASS[e.lvl]">{{ e.lvl }}</span>
            <span class="ml-1 text-info">{{ e.username || 'system' }}</span>
            <span class="ml-3 text-text-2">{{ e.action }}</span><span v-if="e.details" class="text-muted">: {{ e.details }}</span>
            <span v-if="e.ip_address" class="ml-2 hidden text-faint md:inline">[{{ e.ip_address }}]</span>
          </li>
        </ol>
      </div>
    </TerminalWindow>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import SegmentedControl from '@/components/ui/SegmentedControl.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import TerminalWindow from '@/components/ui/TerminalWindow.vue'
import SectionError from './SectionError.vue'
import { fullDateTime, logTime } from './format'
import { api, errorMessage } from '@/services/api'
import { useToast } from '@/composables/useToast'
import type { AdminActivity } from '@/types'

type Level = 'INFO' | 'WARN' | 'ERROR' | 'OTHER'
type LevelFilter = 'ALL' | 'INFO' | 'WARN' | 'ERROR'

const LIMIT_OPTIONS = [
  { value: 100, label: '100' },
  { value: 250, label: '250' },
  { value: 500, label: '500' }
]

const LEVEL_CLASS: Record<Level, string> = {
  INFO: 'text-accent',
  WARN: 'text-warn',
  ERROR: 'text-danger',
  OTHER: 'text-muted'
}

const toast = useToast()

const entries = ref<(AdminActivity & { lvl: Level })[]>([])
const loaded = ref(false)
const loading = ref(false)
const error = ref<string | null>(null)
const level = ref<LevelFilter>('ALL')
const limit = ref(100)
const today = ref(new Date())

function normalizeLevel(raw: string): Level {
  const v = (raw ?? '').toUpperCase()
  if (v === 'INFO' || v === 'WARN' || v === 'ERROR') return v
  if (v === 'WARNING') return 'WARN'
  return 'OTHER'
}

const levelCounts = computed(() => {
  const c = { INFO: 0, WARN: 0, ERROR: 0 }
  for (const e of entries.value) if (e.lvl !== 'OTHER') c[e.lvl] += 1
  return c
})

const levelOptions = computed(() => [
  { value: 'ALL' as LevelFilter, label: 'Barchasi', count: loaded.value ? entries.value.length : undefined },
  { value: 'INFO' as LevelFilter, label: 'INFO', count: loaded.value ? levelCounts.value.INFO : undefined },
  { value: 'WARN' as LevelFilter, label: 'WARN', count: loaded.value ? levelCounts.value.WARN : undefined },
  { value: 'ERROR' as LevelFilter, label: 'ERROR', count: loaded.value ? levelCounts.value.ERROR : undefined }
])

const visible = computed(() => (level.value === 'ALL' ? entries.value : entries.value.filter((e) => e.lvl === level.value)))

async function refresh(): Promise<void> {
  loading.value = true
  error.value = null
  const requested = limit.value
  try {
    const list = await api.adminActivity(requested)
    if (requested !== limit.value) return // a newer request is in flight
    entries.value = list.map((e) => ({ ...e, lvl: normalizeLevel(e.level) }))
    loaded.value = true
    today.value = new Date()
  } catch (err) {
    error.value = errorMessage(err, "Faollik jurnalini yuklab bo'lmadi.")
    if (loaded.value) toast.error(error.value)
  } finally {
    loading.value = false
  }
}

watch(limit, () => void refresh())
onMounted(() => void refresh())

defineExpose({ refresh, loading })
</script>
