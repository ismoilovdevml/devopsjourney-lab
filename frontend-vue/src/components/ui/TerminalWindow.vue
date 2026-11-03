<!--
  TerminalWindow — macOS-style window: traffic-light title bar + mono body.
  Props:
    title?: string               (mono text in the title bar, e.g. "tail -f activity.log")
    tone?: 'panel' | 'terminal'  (body bg #0B1018 or #070A0F; default 'panel')
    padded?: boolean             (body padding 14x16; default true — set false for xterm)
    fill?: boolean               (flex column that fills the parent height; body scrolls)
  Slots: default (body), actions (right side of the title bar), tabs (after the title).
-->
<template>
  <section
    :class="[
      'overflow-hidden rounded-panel border border-border',
      tone === 'terminal' ? 'bg-terminal' : 'bg-surface-2',
      fill ? 'flex h-full min-h-0 flex-col' : ''
    ]"
    :aria-labelledby="title ? titleId : undefined"
  >
    <div class="flex shrink-0 items-center gap-2 border-b border-border bg-titlebar px-4 py-3">
      <span class="h-2.5 w-2.5 rounded-full bg-danger" aria-hidden="true" />
      <span class="h-2.5 w-2.5 rounded-full bg-warn" aria-hidden="true" />
      <span class="h-2.5 w-2.5 rounded-full bg-accent" aria-hidden="true" />
      <h2 v-if="title" :id="titleId" class="ml-2 truncate font-mono text-[13px] font-medium text-muted">
        {{ title }}
      </h2>
      <slot name="tabs" />
      <div v-if="$slots.actions" class="ml-auto flex items-center gap-2">
        <slot name="actions" />
      </div>
    </div>
    <div
      :class="[
        'font-mono text-[12.5px] text-text-2',
        padded ? 'px-4 py-3.5' : '',
        fill ? 'min-h-0 flex-1 overflow-auto' : ''
      ]"
    >
      <slot />
    </div>
  </section>
</template>

<script setup lang="ts">
import { uid } from './utils'

withDefaults(
  defineProps<{
    title?: string
    tone?: 'panel' | 'terminal'
    padded?: boolean
    fill?: boolean
  }>(),
  { tone: 'panel', padded: true }
)

const titleId = uid('term')
</script>
