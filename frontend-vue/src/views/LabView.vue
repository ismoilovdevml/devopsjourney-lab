<!--
  LabView — STUB. Owned by the Workspace page agent; see design/Workspace.dc.html.
  Own chrome (full height, no main nav). Reference implementation: views/_legacy/LabView.vue.
  Building blocks: components/Terminal.vue, stores/session.ts (start/recreate/stop/check, remainingLabel).
-->
<template>
  <div class="flex min-h-screen flex-col">
    <header class="flex items-center gap-4 border-b border-border bg-[rgba(10,14,20,0.82)] px-6 py-3 backdrop-blur-md">
      <AppLogo />
      <span class="font-mono text-[13px] text-muted">tasks / {{ taskId ?? '—' }}</span>
    </header>
    <main id="main" class="flex flex-1 items-center justify-center p-6">
      <EmptyState command="TODO" title="Laboratoriya ishlab chiqilmoqda" description="Bu sahifa tez orada tayyor bo'ladi." />
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import AppLogo from '@/components/ui/AppLogo.vue'
import EmptyState from '@/components/ui/EmptyState.vue'

const route = useRoute()
const taskId = computed(() => (typeof route.params.taskId === 'string' && route.params.taskId) || null)
</script>
