<!--
  LabResultsTab — "Natija" tab: score, progress bar and one row per check (green / red border,
  mono output). When every check passes: "Keyingi vazifa →" (or a link to results on the last task).
  Props:
    check: CheckResponse | null
    checking: boolean
    attempts: number                (checks run in this session)
    checkedAt: number | null        (epoch ms of the last check)
    canCheck: boolean               (a session for this task is running)
    nextTask: TaskSummary | null
    nextLoading?: boolean
  Emits: run, next
-->
<template>
  <div v-if="!check && checking" class="flex flex-col gap-3" aria-busy="true">
    <p class="text-[13px] text-muted" role="status">Tekshirilmoqda…</p>
    <Skeleton height="28px" width="40%" />
    <Skeleton height="6px" rounded="rounded-full" />
    <Skeleton v-for="i in 3" :key="i" height="58px" rounded="rounded-btn" />
  </div>

  <EmptyState
    v-else-if="!check"
    :icon="ClipboardCheck"
    command="$ ./check.sh"
    title="Hali tekshirilmagan"
    :description="emptyDescription"
  >
    <template v-if="canCheck" #actions>
      <Button variant="primary" :icon="Play" :loading="checking" @click="emit('run')">Yechimni tekshirish</Button>
    </template>
  </EmptyState>

  <div v-else class="flex flex-col gap-3.5" :aria-busy="checking || undefined">
    <div class="flex items-baseline justify-between gap-3" role="status" aria-live="polite">
      <span class="font-mono text-2xl font-bold text-text">
        {{ check.score }}/{{ check.total }} passed
      </span>
      <span class="text-[13px] text-muted">{{ attemptLabel }}</span>
    </div>

    <div class="h-1.5 overflow-hidden rounded-full bg-divider" aria-hidden="true">
      <div
        :class="['h-full rounded-full transition-[width] duration-500', check.passed ? 'bg-accent' : 'bg-info']"
        :style="{ width: `${percent}%` }"
      />
    </div>

    <ul class="flex flex-col gap-2" :class="checking ? 'opacity-60' : ''">
      <li
        v-for="(r, i) in check.results"
        :key="i"
        :class="[
          'flex items-start gap-2.5 rounded-btn border bg-surface-solid px-3.5 py-3',
          r.ok ? 'border-accent/30' : 'border-danger/35'
        ]"
      >
        <Check v-if="r.ok" class="mt-0.5 h-[18px] w-[18px] shrink-0 text-accent" :stroke-width="2.4" aria-label="O'tdi" role="img" />
        <X v-else class="mt-0.5 h-[18px] w-[18px] shrink-0 text-danger" :stroke-width="2.4" aria-label="O'tmadi" role="img" />
        <span class="min-w-0 flex-1">
          <span class="block font-medium text-text">{{ r.name }}</span>
          <span
            v-if="r.output?.trim()"
            class="mt-0.5 block whitespace-pre-wrap break-words font-mono text-meta text-muted"
          >{{ clip(r.output) }}</span>
        </span>
      </li>
    </ul>

    <template v-if="check.passed">
      <Button v-if="nextTask" variant="primary" size="lg" block :loading="nextLoading" :disabled="checking" @click="emit('next')">
        Keyingi vazifa →
      </Button>
      <Button v-else variant="primary" size="lg" block to="/progress">Barcha vazifalar yakunlandi — natijalar</Button>
    </template>
    <p v-else class="text-[13px] text-muted">
      Xatolarni tuzating va qayta tekshiring. Qiyin bo'lsa — Maslahatlar bo'limiga qarang.
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useNow } from '@vueuse/core'
import { Check, ClipboardCheck, Play, X } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import type { CheckResponse, TaskSummary } from '@/types'

const props = defineProps<{
  check: CheckResponse | null
  checking: boolean
  attempts: number
  checkedAt: number | null
  canCheck: boolean
  nextTask: TaskSummary | null
  nextLoading?: boolean
}>()

const emit = defineEmits<{ run: []; next: [] }>()

const now = useNow({ interval: 15000 })

const MAX_OUTPUT = 600

const percent = computed(() => {
  const c = props.check
  if (!c || c.total <= 0) return 0
  return Math.round((Math.min(c.score, c.total) / c.total) * 100)
})

function ago(ts: number): string {
  const sec = Math.max(0, Math.round((now.value.getTime() - ts) / 1000))
  if (sec < 60) return 'hozirgina'
  const min = Math.floor(sec / 60)
  if (min < 60) return `${min} daqiqa oldin`
  return new Date(ts).toLocaleTimeString('uz-UZ', { hour: '2-digit', minute: '2-digit' })
}

const emptyDescription = computed(() =>
  props.canCheck
    ? "Terminalda topshiriqni bajaring, so'ng natijani ko'rish uchun yechimni tekshiring."
    : "Natijani ko'rish uchun avval muhitni ishga tushiring."
)

const attemptLabel = computed(() => {
  const parts: string[] = []
  if (props.attempts > 0) parts.push(`${props.attempts}-urinish`)
  if (props.checkedAt) parts.push(ago(props.checkedAt))
  return parts.join(' · ')
})

function clip(text: string): string {
  const t = text.trim()
  return t.length > MAX_OUTPUT ? `${t.slice(0, MAX_OUTPUT)}…` : t
}
</script>
