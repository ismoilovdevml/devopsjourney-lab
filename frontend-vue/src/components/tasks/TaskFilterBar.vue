<!--
  TaskFilterBar — mono "> grep" search + status and level segmented filters.
  Props (all v-model): search: string, status: StatusFilter, level: LevelFilter
  Exposes: focusSearch()
-->
<template>
  <div class="flex flex-wrap items-center gap-3 rounded-card border border-border bg-[rgba(15,21,32,0.8)] p-3">
    <label class="relative block min-w-0 flex-[1_1_260px] sm:max-w-[380px]">
      <span class="sr-only">Vazifa qidirish</span>
      <span class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 font-mono text-accent" aria-hidden="true">&gt;</span>
      <input
        ref="input"
        type="search"
        :value="search"
        placeholder="grep vazifa yoki mavzu…"
        autocomplete="off"
        spellcheck="false"
        class="input pl-[30px] font-mono text-[13px]"
        @input="emit('update:search', ($event.target as HTMLInputElement).value)"
        @keydown.esc="onEscape"
      />
    </label>
    <SegmentedControl
      :model-value="status"
      :options="STATUS_OPTIONS"
      ariaLabel="Holat bo'yicha"
      class="max-sm:grid max-sm:w-full max-sm:grid-cols-2"
      @update:model-value="onStatus"
    />
    <SegmentedControl
      :model-value="level"
      :options="LEVEL_OPTIONS"
      ariaLabel="Daraja bo'yicha"
      class="max-sm:grid max-sm:w-full max-sm:grid-cols-[1.6fr_1fr_1fr_1fr]"
      @update:model-value="onLevel"
    />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import SegmentedControl from '@/components/ui/SegmentedControl.vue'
import { LEVEL_OPTIONS, STATUS_OPTIONS, type LevelFilter, type StatusFilter } from './filters'

const props = defineProps<{
  search: string
  status: StatusFilter
  level: LevelFilter
}>()

const emit = defineEmits<{
  'update:search': [value: string]
  'update:status': [value: StatusFilter]
  'update:level': [value: LevelFilter]
}>()

const input = ref<HTMLInputElement | null>(null)

// SegmentedControl emits the generic `string | number`; narrow it back to the filter unions.
function onStatus(value: string | number): void {
  const opt = STATUS_OPTIONS.find((o) => o.value === value)
  if (opt) emit('update:status', opt.value)
}

function onLevel(value: string | number): void {
  const opt = LEVEL_OPTIONS.find((o) => o.value === value)
  if (opt) emit('update:level', opt.value)
}

function onEscape(): void {
  if (props.search) emit('update:search', '')
}

function focusSearch(): void {
  input.value?.focus()
  input.value?.select()
}

defineExpose({ focusSearch })
</script>
