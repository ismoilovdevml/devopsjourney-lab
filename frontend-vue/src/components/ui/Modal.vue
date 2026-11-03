<!--
  Modal — accessible dialog: teleported to <body>, focus trap, Esc closes, focus is restored on close,
  body scroll locked while open.
  Props:
    open: boolean                 (v-model:open)
    title: string
    description?: string
    size?: 'sm' | 'md' | 'lg'     (max width 400 / 520 / 720px; default 'md')
    closeOnBackdrop?: boolean     (default true)
    dismissible?: boolean         (Esc / backdrop / close button allowed; default true — set false while busy)
  Emits: update:open(false), close
  Slots: default (body), footer (right-aligned action row).
-->
<template>
  <Teleport to="body">
    <Transition name="modal">
      <div
        v-if="open"
        class="fixed inset-0 z-50 flex items-end justify-center bg-[rgba(4,7,11,0.72)] p-4 backdrop-blur-sm sm:items-center"
        @mousedown.self="onBackdrop"
      >
        <div
          ref="panel"
          role="dialog"
          aria-modal="true"
          :aria-labelledby="titleId"
          :aria-describedby="description ? descId : undefined"
          :class="['w-full overflow-hidden rounded-panel border border-border-strong bg-surface-solid shadow-2xl', SIZES[size]]"
          tabindex="-1"
          @keydown="onKeydown"
        >
          <header class="flex items-start justify-between gap-4 border-b border-divider px-6 py-5">
            <div class="min-w-0">
              <h2 :id="titleId" class="font-mono text-base font-semibold text-text">{{ title }}</h2>
              <p v-if="description" :id="descId" class="mt-1 text-[13px] text-muted">{{ description }}</p>
            </div>
            <button
              v-if="dismissible"
              type="button"
              class="-mr-2 -mt-1 flex h-9 w-9 shrink-0 items-center justify-center rounded-ctl text-muted hover:bg-surface-3 hover:text-text"
              aria-label="Yopish"
              @click="close"
            >
              <X class="h-4 w-4" aria-hidden="true" />
            </button>
          </header>
          <div v-if="$slots.default" class="px-6 py-5 text-sm text-text-2">
            <slot />
          </div>
          <footer v-if="$slots.footer" class="flex flex-wrap justify-end gap-2 border-t border-divider px-6 py-4">
            <slot name="footer" />
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { X } from 'lucide-vue-next'
import { uid } from './utils'

const props = withDefaults(
  defineProps<{
    open: boolean
    title: string
    description?: string
    size?: 'sm' | 'md' | 'lg'
    closeOnBackdrop?: boolean
    dismissible?: boolean
  }>(),
  { size: 'md', closeOnBackdrop: true, dismissible: true }
)

const emit = defineEmits<{ 'update:open': [value: boolean]; close: [] }>()

const SIZES = { sm: 'max-w-[400px]', md: 'max-w-[520px]', lg: 'max-w-[720px]' } as const
const FOCUSABLE =
  'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'

const titleId = uid('modal-title')
const descId = uid('modal-desc')
const panel = ref<HTMLElement | null>(null)
let previouslyFocused: HTMLElement | null = null

function close() {
  if (!props.dismissible) return
  emit('update:open', false)
  emit('close')
}

function onBackdrop() {
  if (props.closeOnBackdrop) close()
}

function focusables(): HTMLElement[] {
  if (!panel.value) return []
  return Array.from(panel.value.querySelectorAll<HTMLElement>(FOCUSABLE)).filter(
    (el) => el.offsetParent !== null || el === document.activeElement
  )
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.stopPropagation()
    close()
    return
  }
  if (e.key !== 'Tab') return
  const items = focusables()
  if (items.length === 0) {
    e.preventDefault()
    panel.value?.focus()
    return
  }
  const first = items[0]
  const last = items[items.length - 1]
  if (e.shiftKey && (document.activeElement === first || document.activeElement === panel.value)) {
    e.preventDefault()
    last.focus()
  } else if (!e.shiftKey && document.activeElement === last) {
    e.preventDefault()
    first.focus()
  }
}

function lockScroll(lock: boolean) {
  document.body.style.overflow = lock ? 'hidden' : ''
}

watch(
  () => props.open,
  async (isOpen) => {
    if (isOpen) {
      previouslyFocused = document.activeElement instanceof HTMLElement ? document.activeElement : null
      lockScroll(true)
      await nextTick()
      const target = panel.value?.querySelector<HTMLElement>('[autofocus]') ?? focusables()[0] ?? panel.value
      target?.focus()
    } else {
      lockScroll(false)
      previouslyFocused?.focus()
      previouslyFocused = null
    }
  },
  { immediate: true }
)

onBeforeUnmount(() => {
  if (props.open) lockScroll(false)
})
</script>

<style scoped>
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.15s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}
</style>
