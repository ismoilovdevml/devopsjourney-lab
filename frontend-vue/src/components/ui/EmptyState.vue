<!--
  EmptyState — dashed placeholder for empty lists / errors.
  Props:
    title: string
    description?: string
    icon?: Component     (lucide icon, shown in a tinted tile)
    command?: string     (mono line above the title, e.g. "0 results")
    tone?: 'neutral' | 'danger'  (danger = error state; default 'neutral')
  Slots: actions (buttons under the text).
-->
<template>
  <div
    :class="[
      'flex flex-col items-center justify-center gap-3 rounded-card border border-dashed bg-surface px-6 py-10 text-center',
      tone === 'danger' ? 'border-danger/40' : 'border-border-stronger'
    ]"
    :role="tone === 'danger' ? 'alert' : undefined"
  >
    <span
      v-if="icon"
      :class="[
        'flex h-11 w-11 items-center justify-center rounded-btn border',
        tone === 'danger' ? 'border-danger/30 bg-danger/10 text-danger' : 'border-border-strong bg-surface-3 text-muted'
      ]"
      aria-hidden="true"
    >
      <component :is="icon" class="h-5 w-5" />
    </span>
    <p v-if="command" class="font-mono text-[13px] text-muted">{{ command }}</p>
    <h3 class="text-[15px] font-semibold text-text">{{ title }}</h3>
    <p v-if="description" class="max-w-md text-[13px] text-muted">{{ description }}</p>
    <div v-if="$slots.actions" class="mt-2 flex flex-wrap items-center justify-center gap-2">
      <slot name="actions" />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Component } from 'vue'

withDefaults(
  defineProps<{
    title: string
    description?: string
    icon?: Component
    command?: string
    tone?: 'neutral' | 'danger'
  }>(),
  { tone: 'neutral' }
)
</script>
