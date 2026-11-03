<!--
  LabTaskTab — "Topshiriq" tab: description, automatic checks (with the last check's status) and the
  task README (title and solution section removed, see readme.ts).
  Props:
    task: TaskDetail
    readme: string                 (README body markdown)
    check: CheckResponse | null    (last check of the current session)
    hasHints: boolean
  Emits: open-hints
-->
<template>
  <div class="flex flex-col gap-5">
    <p v-if="task.description" class="text-text-2">{{ task.description }}</p>

    <div v-if="task.checks.length">
      <h2 class="mb-2.5 font-mono text-meta font-semibold text-muted"># avtomatik tekshiruvlar</h2>
      <ul class="flex flex-col gap-1.5 font-mono text-meta text-text-2">
        <li v-for="(c, i) in task.checks" :key="i" class="flex gap-2">
          <span :class="['shrink-0', STATE_CLASS[stateOf(c.name, i)]]" aria-hidden="true">{{ STATE_GLYPH[stateOf(c.name, i)] }}</span>
          <span class="sr-only">{{ STATE_SR[stateOf(c.name, i)] }}:</span>
          <span>{{ c.name }}</span>
        </li>
      </ul>
    </div>

    <LabMarkdown v-if="readme" :source="readme" />

    <p
      v-if="hasHints"
      class="rounded-btn border border-warn/30 bg-warn/[0.08] px-3.5 py-3 text-[13px] text-[#F5D9A3]"
    >
      Qiyin bo'lsa,
      <button type="button" class="font-semibold text-warn underline-offset-2 hover:underline" @click="emit('open-hints')">
        Maslahatlar
      </button>ni oching — ular bittadan ochiladi, o'zingiz o'ylashga imkon qoladi.
    </p>
  </div>
</template>

<script setup lang="ts">
import LabMarkdown from './LabMarkdown.vue'
import type { CheckResponse, TaskDetail } from '@/types'

const props = defineProps<{
  task: TaskDetail
  readme: string
  check: CheckResponse | null
  hasHints: boolean
}>()

const emit = defineEmits<{ 'open-hints': [] }>()

type CheckState = 'pass' | 'fail' | 'unknown'

const STATE_GLYPH: Record<CheckState, string> = { pass: '[✓]', fail: '[✗]', unknown: '[ ]' }
const STATE_CLASS: Record<CheckState, string> = { pass: 'text-accent', fail: 'text-danger', unknown: 'text-muted' }
const STATE_SR: Record<CheckState, string> = { pass: "O'tdi", fail: "O'tmadi", unknown: 'Tekshirilmagan' }

/** Match a task check to the last check result by name, falling back to position. */
function stateOf(name: string, index: number): CheckState {
  const results = props.check?.results
  if (!results?.length) return 'unknown'
  const r = results.find((x) => x.name === name) ?? results[index]
  if (!r) return 'unknown'
  return r.ok ? 'pass' : 'fail'
}
</script>
