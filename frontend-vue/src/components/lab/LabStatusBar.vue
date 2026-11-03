<!--
  LabStatusBar — bottom IDE-style status bar (green tint): connection state, task id, image,
  session time left.
  Props:
    connection: ConnectionState
    taskId: string | null
    image?: string
    remainingLabel?: string | null   (shown when a session is running)
-->
<template>
  <footer
    class="flex flex-wrap gap-x-[18px] gap-y-0.5 border-t border-[#1B4A37] bg-[#0F2A20] px-4 py-[5px] font-mono text-xs text-[#A7F3D0]"
  >
    <span>
      <span :class="connection === 'open' ? 'text-[#A7F3D0]' : CONNECTION_COLOR[connection]" aria-hidden="true">●</span>
      {{ CONNECTION_LABEL[connection] }}
    </span>
    <span v-if="taskId">tasks/{{ taskId }}</span>
    <span v-if="image">{{ image }}</span>
    <span v-if="remainingLabel" class="tabular-nums">ttl {{ remainingLabel }}</span>
    <span class="ml-auto hidden sm:inline">ws · utf-8</span>
  </footer>
</template>

<script setup lang="ts">
import { CONNECTION_COLOR, CONNECTION_LABEL, type ConnectionState } from './types'

defineProps<{
  connection: ConnectionState
  taskId: string | null
  image?: string
  remainingLabel?: string | null
}>()
</script>
