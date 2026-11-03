<!--
  PageHeader — mono "command" line + page title (30px mono) + optional subtitle and actions.
  Props:
    command: string     (e.g. "$ ls ./tasks"; a leading "$ " is rendered as-is)
    title: string       (rendered as the page <h1>)
    subtitle?: string
  Slots: actions (right side, wraps under the title on narrow screens).
-->
<template>
  <div class="flex flex-wrap items-end justify-between gap-4">
    <div class="flex min-w-0 flex-col gap-1.5">
      <span class="cmd-line">{{ command }}</span>
      <h1 class="page-title">{{ title }}</h1>
      <p v-if="subtitle" class="text-sm text-muted">{{ subtitle }}</p>
    </div>
    <div v-if="$slots.actions" class="flex flex-wrap items-center gap-2">
      <slot name="actions" />
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  command: string
  title: string
  subtitle?: string
}>()
</script>
