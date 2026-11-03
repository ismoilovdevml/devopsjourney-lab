<!--
  SectionError — Uzbek error state for one admin section with a retry button.
  Props:
    message: string        (errorMessage(err) text)
    command?: string       (mono line, e.g. "GET /api/admin/users")
    retrying?: boolean     (spinner on the retry button)
  Emits: retry
-->
<template>
  <EmptyState
    tone="danger"
    :icon="AlertTriangle"
    :command="command"
    title="Ma'lumotlarni yuklab bo'lmadi"
    :description="message"
  >
    <template #actions>
      <Button size="sm" :icon="RefreshCw" :loading="retrying" @click="emit('retry')">Qayta urinish</Button>
    </template>
  </EmptyState>
</template>

<script setup lang="ts">
import { AlertTriangle, RefreshCw } from 'lucide-vue-next'
import EmptyState from '@/components/ui/EmptyState.vue'
import Button from '@/components/ui/Button.vue'

defineProps<{ message: string; command?: string; retrying?: boolean }>()
const emit = defineEmits<{ retry: [] }>()
</script>
