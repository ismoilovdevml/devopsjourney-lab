<!--
  ToastHost — renders toasts from composables/useToast (mounted once in App.vue).
  Props: none. Use `const toast = useToast(); toast.success('...')` anywhere.
-->
<template>
  <Teleport to="body">
    <div class="pointer-events-none fixed bottom-4 right-4 z-[60] flex w-[min(380px,calc(100vw-32px))] flex-col gap-2" aria-live="polite" role="status">
      <TransitionGroup name="toast">
        <div
          v-for="t in toasts"
          :key="t.id"
          :class="['pointer-events-auto flex items-start gap-3 rounded-btn border bg-surface-solid px-4 py-3 shadow-2xl', BORDER[t.type]]"
        >
          <component :is="ICON[t.type]" :class="['mt-0.5 h-4 w-4 shrink-0', COLOR[t.type]]" aria-hidden="true" />
          <p class="flex-1 text-[13px] text-text">{{ t.message }}</p>
          <button
            type="button"
            class="-mr-1 flex h-6 w-6 shrink-0 items-center justify-center rounded text-muted hover:text-text"
            aria-label="Yopish"
            @click="removeToast(t.id)"
          >
            <X class="h-3.5 w-3.5" aria-hidden="true" />
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { AlertTriangle, CheckCircle2, Info, X, XCircle } from 'lucide-vue-next'
import { useToast, type Toast } from '@/composables/useToast'

const { toasts, removeToast } = useToast()

const ICON = { success: CheckCircle2, error: XCircle, warning: AlertTriangle, info: Info } as const
const COLOR: Record<Toast['type'], string> = {
  success: 'text-accent',
  error: 'text-danger',
  warning: 'text-warn',
  info: 'text-info'
}
const BORDER: Record<Toast['type'], string> = {
  success: 'border-accent/35',
  error: 'border-danger/40',
  warning: 'border-warn/35',
  info: 'border-info/35'
}
</script>

<style scoped>
.toast-enter-active,
.toast-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateY(8px);
}
</style>
