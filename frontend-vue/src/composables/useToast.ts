/**
 * Global toast queue rendered by components/ui/ToastHost.vue.
 * Usage: const toast = useToast(); toast.success('Saqlandi'); toast.error(errorMessage(err))
 * Durations in ms; 0 = sticky. Errors stay longer by default.
 */
import { ref } from 'vue'

export interface Toast {
  id: number
  message: string
  type: 'success' | 'error' | 'info' | 'warning'
  duration: number
}

const toasts = ref<Toast[]>([])
let toastId = 0

function removeToast(id: number) {
  toasts.value = toasts.value.filter((t) => t.id !== id)
}

function addToast(message: string, type: Toast['type'] = 'info', duration = 3500) {
  const id = ++toastId
  toasts.value = [...toasts.value.slice(-4), { id, message, type, duration }]
  if (duration > 0) setTimeout(() => removeToast(id), duration)
  return id
}

export function useToast() {
  return {
    toasts,
    addToast,
    removeToast,
    success: (message: string, duration?: number) => addToast(message, 'success', duration),
    error: (message: string, duration = 6000) => addToast(message, 'error', duration),
    info: (message: string, duration?: number) => addToast(message, 'info', duration),
    warning: (message: string, duration = 5000) => addToast(message, 'warning', duration)
  }
}
