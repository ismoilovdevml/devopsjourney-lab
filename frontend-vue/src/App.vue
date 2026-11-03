<template>
  <RouterView />
  <ToastHost />
</template>

<script setup lang="ts">
import { watch } from 'vue'
import ToastHost from '@/components/ui/ToastHost.vue'
import { useAuthStore } from '@/stores/auth'
import { useSessionStore } from '@/stores/session'

const auth = useAuthStore()
const session = useSessionStore()

// Restore the running lab session whenever a user becomes known (page load or login).
watch(
  () => auth.user?.id,
  (id) => {
    if (id) session.restore().catch(() => undefined)
    else session.clear()
  },
  { immediate: true }
)
</script>
