<!--
  LabTerminalPane — right workspace pane: terminal tab bar, xterm body (Terminal.vue, re-mounted per
  sessionId so "recreate" reconnects cleanly) and the bottom bar with "Yechimni tekshirish".
  Props:
    sessionId: string
    connection: ConnectionState
    checking: boolean
    disabled?: boolean      (session is being recreated / stopped)
  Emits: check, status(ConnectionState)
  Exposes: focus()
-->
<template>
  <section aria-label="Terminal" class="flex min-h-[60vh] min-w-0 flex-1 flex-col bg-terminal lg:min-h-0">
    <div class="flex items-stretch border-b border-divider bg-surface-2">
      <span
        class="flex h-10 min-w-0 items-center gap-2 border-r border-divider bg-terminal px-4 font-mono text-meta text-text shadow-[inset_0_2px_0_#3DDC97]"
      >
        <span class="text-accent" aria-hidden="true">❯</span>
        <span class="truncate">bash — root@lab-{{ shortId }}</span>
      </span>
      <button
        type="button"
        disabled
        title="Bir nechta terminal oynasi — tez orada"
        aria-label="Yangi terminal oynasi (tez orada)"
        class="flex w-10 cursor-not-allowed items-center justify-center text-faint"
      >
        <Plus class="h-4 w-4" aria-hidden="true" />
      </button>
      <span class="ml-auto hidden items-center px-3.5 text-xs text-muted sm:flex">
        Ctrl+Shift+C / V — nusxa / qo'yish
      </span>
    </div>

    <div class="relative min-h-0 flex-1">
      <div class="absolute inset-0">
        <Terminal :key="sessionId" ref="terminal" :session-id="sessionId" @status="emit('status', $event)" />
      </div>
    </div>

    <div class="flex flex-wrap items-center gap-3 border-t border-divider bg-surface-2 px-4 py-3">
      <span class="flex items-center gap-2 text-[13px] text-muted" aria-live="polite">
        <span :class="['text-[10px]', CONNECTION_COLOR[connection]]" aria-hidden="true">●</span>
        <span class="hidden sm:inline">Tayyor bo'lsangiz — tekshiring. Urinishlar cheklanmagan.</span>
        <span class="sm:hidden">{{ CONNECTION_LABEL[connection] }}</span>
      </span>
      <span class="ml-auto hidden font-mono text-xs text-faint md:inline">{{ shortcut }}</span>
      <Button
        variant="primary"
        size="lg"
        :icon="Play"
        :loading="checking"
        :disabled="disabled"
        :aria-keyshortcuts="isMac ? 'Meta+Enter' : 'Control+Enter'"
        class="max-md:ml-auto"
        @click="emit('check')"
      >
        {{ checking ? 'Tekshirilmoqda…' : 'Yechimni tekshirish' }}
      </Button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Play, Plus } from 'lucide-vue-next'
import Terminal from '@/components/Terminal.vue'
import Button from '@/components/ui/Button.vue'
import { CONNECTION_COLOR, CONNECTION_LABEL, type ConnectionState } from './types'

const props = defineProps<{
  sessionId: string
  connection: ConnectionState
  checking: boolean
  disabled?: boolean
}>()

const emit = defineEmits<{ check: []; status: [status: ConnectionState] }>()

const terminal = ref<InstanceType<typeof Terminal> | null>(null)

const isMac = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.platform || navigator.userAgent)
const shortcut = isMac ? '⌘ + Enter' : 'Ctrl + Enter'

/** Container hostname-like suffix for the tab title (first 4 hex chars of the session id). */
const shortId = computed(() => props.sessionId.replace(/[^a-z0-9]/gi, '').slice(0, 4).toLowerCase() || 'lab')

function focus() {
  terminal.value?.focus()
}

defineExpose({ focus })
</script>
