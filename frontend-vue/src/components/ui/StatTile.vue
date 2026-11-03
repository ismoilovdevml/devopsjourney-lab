<!--
  StatTile — colored icon tile + label + big mono value (dashboard / progress KPIs).
  Props:
    label: string
    value: string | number
    of?: string         (muted suffix after the value, e.g. "/ 6" or " xp")
    icon: Component     (lucide icon)
    color?: string      (hex accent for the icon tile, default #3DDC97)
    loading?: boolean   (shows a skeleton instead of the value)
-->
<template>
  <div class="flex items-center gap-3.5 rounded-card border border-border bg-surface px-5 py-[18px]">
    <span
      class="flex h-11 w-11 shrink-0 items-center justify-center rounded-btn border"
      :style="{ color, backgroundColor: tint(color, 0.12), borderColor: tint(color, 0.3) }"
      aria-hidden="true"
    >
      <component :is="icon" class="h-5 w-5" />
    </span>
    <span class="flex min-w-0 flex-col">
      <span class="text-[13px] text-muted">{{ label }}</span>
      <Skeleton v-if="loading" width="80px" height="28px" class="mt-1" />
      <span v-else class="font-mono text-2xl font-bold text-text">
        {{ value }}<span v-if="of" class="text-sm font-medium text-muted">{{ of }}</span>
      </span>
    </span>
  </div>
</template>

<script setup lang="ts">
import type { Component } from 'vue'
import Skeleton from './Skeleton.vue'
import { tint } from './utils'

withDefaults(
  defineProps<{
    label: string
    value: string | number
    of?: string
    icon: Component
    color?: string
    loading?: boolean
  }>(),
  { color: '#3DDC97' }
)
</script>
