<!--
  Card — translucent panel over the wallpaper (radius 16px, border #1F2A3A).
  Props:
    as?: string            (root tag, default 'section')
    title?: string         (renders a header row with a mono 16px title)
    subtitle?: string      (13px muted line under the title)
    padding?: 'none' | 'sm' | 'md' | 'lg'  (body padding: 0 / 12 / 20 / 24px; default 'md')
    variant?: 'default' | 'inset'  (inset = solid #0B1018, for terminal-like blocks)
    interactive?: boolean  (hover border highlight)
    selected?: boolean     (green border + ring)
  Slots: default (body), actions (right side of the header), header (replaces the whole header).
-->
<template>
  <component
    :is="as"
    :class="[
      'overflow-hidden rounded-panel border transition-colors',
      variant === 'inset' ? 'bg-surface-2' : 'bg-surface backdrop-blur-[6px]',
      selected ? 'border-accent shadow-selected' : 'border-border',
      interactive && !selected ? 'hover:border-border-stronger' : ''
    ]"
    :aria-labelledby="title ? headingId : undefined"
  >
    <slot name="header">
      <header
        v-if="title || $slots.actions"
        class="flex flex-wrap items-center justify-between gap-3 border-b border-divider px-6 py-5"
      >
        <div class="min-w-0">
          <h2 v-if="title" :id="headingId" class="font-mono text-base font-semibold text-text">{{ title }}</h2>
          <p v-if="subtitle" class="mt-0.5 text-[13px] text-muted">{{ subtitle }}</p>
        </div>
        <div v-if="$slots.actions" class="flex items-center gap-2">
          <slot name="actions" />
        </div>
      </header>
    </slot>
    <div :class="PADDING[padding]">
      <slot />
    </div>
  </component>
</template>

<script setup lang="ts">
import { uid } from './utils'

withDefaults(
  defineProps<{
    as?: string
    title?: string
    subtitle?: string
    padding?: 'none' | 'sm' | 'md' | 'lg'
    variant?: 'default' | 'inset'
    interactive?: boolean
    selected?: boolean
  }>(),
  { as: 'section', padding: 'md', variant: 'default' }
)

const headingId = uid('card')

const PADDING = { none: '', sm: 'p-3', md: 'p-5', lg: 'p-6' } as const
</script>
