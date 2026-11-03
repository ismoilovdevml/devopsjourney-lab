<!--
  LabHintsTab — "Maslahatlar" tab. Hints unlock strictly one at a time; the unlocked count is
  remembered per task in sessionStorage (`dj.lab.hints.<taskId>`), so a reload keeps them open.
  The README solution section (if any) is the last, most detailed hint.
  Props:
    taskId: string
    hints: string[]      (TaskDetail.hints)
    solution?: string    (README "## Hints" section, markdown)
-->
<template>
  <EmptyState
    v-if="total === 0"
    :icon="Lightbulb"
    title="Maslahatlar yo'q"
    description="Bu vazifa uchun maslahatlar qo'shilmagan. Topshiriq matnini diqqat bilan o'qing."
  />
  <div v-else class="flex flex-col gap-2.5">
    <p class="mb-1 text-[13px] text-muted" aria-live="polite">
      {{ opened }} / {{ total }} ochildi · maslahatlar ballga ta'sir qilmaydi
    </p>

    <div
      v-for="(item, i) in items"
      :key="i"
      class="overflow-hidden rounded-btn border border-border bg-surface-solid"
    >
      <div class="flex items-center justify-between gap-2.5 px-3.5 py-3">
        <h3 :id="`${uidPrefix}-h-${i}`" class="font-medium text-text">{{ item.title }}</h3>
        <button
          v-if="i >= opened"
          type="button"
          :disabled="i > opened"
          :title="i > opened ? 'Avval oldingi maslahatni oching' : undefined"
          :aria-describedby="`${uidPrefix}-h-${i}`"
          :class="[
            'h-8 shrink-0 rounded-[7px] border border-border-stronger bg-surface-3 px-3 text-[13px] font-medium transition-colors',
            i > opened ? 'cursor-not-allowed text-faint' : 'text-accent hover:border-accent/50'
          ]"
          @click="unlock(i)"
        >
          <span class="inline-flex items-center gap-1.5">
            <Lock v-if="i > opened" class="h-3 w-3" aria-hidden="true" />
            Ochish
          </span>
        </button>
      </div>
      <div
        v-if="i < opened"
        :ref="(el) => setPanelRef(i, el)"
        tabindex="-1"
        :aria-labelledby="`${uidPrefix}-h-${i}`"
        role="region"
        class="border-t border-border bg-terminal px-3.5 py-3 focus-visible:outline-offset-[-2px]"
      >
        <LabMarkdown v-if="item.markdown" :source="item.body" />
        <pre v-else class="whitespace-pre-wrap font-mono text-meta leading-relaxed text-accent">{{ item.body }}</pre>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch, type ComponentPublicInstance } from 'vue'
import { Lightbulb, Lock } from 'lucide-vue-next'
import EmptyState from '@/components/ui/EmptyState.vue'
import { uid } from '@/components/ui/utils'
import LabMarkdown from './LabMarkdown.vue'

const props = defineProps<{ taskId: string; hints: string[]; solution?: string }>()

interface HintItem {
  title: string
  body: string
  markdown: boolean
}

const uidPrefix = uid('lab-hint')

const items = computed<HintItem[]>(() => {
  const list: HintItem[] = props.hints
    .filter((h) => h.trim())
    .map((h, i) => ({ title: `${i + 1}-maslahat`, body: h.trim(), markdown: false }))
  if (props.solution?.trim()) {
    list.push({ title: "Batafsil ko'rsatma (yechim)", body: props.solution, markdown: true })
  }
  return list
})

const total = computed(() => items.value.length)

const storageKey = (id: string) => `dj.lab.hints.${id}`

function readOpened(id: string): number {
  try {
    const n = Number.parseInt(sessionStorage.getItem(storageKey(id)) ?? '0', 10)
    return Number.isFinite(n) && n > 0 ? n : 0
  } catch {
    return 0
  }
}

const stored = ref(readOpened(props.taskId))
const opened = computed(() => Math.min(stored.value, total.value))

watch(
  () => props.taskId,
  (id) => {
    stored.value = readOpened(id)
  }
)

const panels = new Map<number, HTMLElement>()

function setPanelRef(i: number, el: Element | ComponentPublicInstance | null) {
  if (el instanceof HTMLElement) panels.set(i, el)
  else panels.delete(i)
}

async function unlock(i: number) {
  if (i !== opened.value) return
  stored.value = i + 1
  try {
    sessionStorage.setItem(storageKey(props.taskId), String(stored.value))
  } catch {
    // sessionStorage unavailable: hints stay open for this page view only.
  }
  await nextTick()
  panels.get(i)?.focus()
}
</script>
