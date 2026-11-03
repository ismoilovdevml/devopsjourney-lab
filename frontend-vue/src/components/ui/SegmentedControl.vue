<!--
  SegmentedControl — group of toggle buttons (filters, periods, sizes). Uses aria-pressed.
  Props:
    modelValue: T                 (v-model; T = string | number)
    options: { value: T; label: string; count?: number }[]
    ariaLabel: string             (group label, e.g. "Holat bo'yicha")
    size?: 'sm' | 'md'            (button height 30 / 34px; default 'md')
    block?: boolean               (stretch buttons to fill the width)
  Emits: update:modelValue(T)
-->
<template>
  <div
    role="group"
    :aria-label="ariaLabel"
    :class="[
      'flex flex-wrap gap-1 rounded-btn border border-border bg-surface-2 p-1',
      block ? 'w-full' : 'inline-flex'
    ]"
  >
    <button
      v-for="opt in options"
      :key="String(opt.value)"
      type="button"
      :aria-pressed="opt.value === modelValue"
      :class="[
        'inline-flex items-center justify-center gap-1.5 rounded-[7px] px-3 font-medium transition-colors',
        size === 'sm' ? 'h-[30px] text-[13px]' : 'h-[34px] text-sm',
        block ? 'flex-1' : '',
        opt.value === modelValue
          ? 'bg-raised text-text shadow-ring-strong'
          : 'bg-transparent text-muted hover:text-text'
      ]"
      @click="emit('update:modelValue', opt.value)"
    >
      {{ opt.label }}
      <span v-if="opt.count !== undefined" class="font-mono text-xs text-muted">{{ opt.count }}</span>
    </button>
  </div>
</template>

<script setup lang="ts" generic="T extends string | number">
withDefaults(
  defineProps<{
    modelValue: T
    options: { value: T; label: string; count?: number }[]
    ariaLabel: string
    size?: 'sm' | 'md'
    block?: boolean
  }>(),
  { size: 'md' }
)

const emit = defineEmits<{ 'update:modelValue': [value: T] }>()
</script>
