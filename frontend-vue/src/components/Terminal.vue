<!--
  Terminal — xterm.js bound to the lab container over WebSocket.
  Connects to same-origin `/ws?sessionId=<id>`; the httpOnly auth cookie authenticates (no token in URL).
  Frames (docs/API.md): client->server raw text = stdin, {"type":"resize","cols","rows"} = resize;
  server->client text/binary = terminal output.
  Theme and font size come from the preferences store and update live.
  Props:
    sessionId: string        (reconnects when it changes, e.g. after "recreate")
    showStatus?: boolean     (connection pill in the top-right corner; default true)
  Emits: status('connecting' | 'open' | 'reconnecting' | 'closed' | 'failed')
  Keys: Ctrl+Shift+C copies the selection, Ctrl+Shift+V pastes (Ctrl+C stays SIGINT).
  Exposes: focus(), fit(), clear()
-->
<template>
  <div class="relative h-full w-full" :style="{ background: prefs.xtermTheme.background }">
    <div ref="host" class="xterm-host" :style="{ background: prefs.xtermTheme.background }" @click="focus" />

    <div v-if="showStatus" class="pointer-events-none absolute right-3 top-3 z-10" aria-live="polite">
      <span
        v-if="status !== 'open'"
        :class="[
          'inline-flex items-center gap-2 rounded-full border px-2.5 py-1 font-mono text-xs backdrop-blur-sm',
          status === 'failed' || status === 'closed'
            ? 'border-danger/40 bg-danger/10 text-danger'
            : 'border-warn/35 bg-warn/10 text-warn'
        ]"
      >
        <span class="h-1.5 w-1.5 rounded-full bg-current" aria-hidden="true" />
        {{ statusLabel }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import 'xterm/css/xterm.css'
import { usePreferencesStore } from '@/stores/preferences'
import { useAuthStore } from '@/stores/auth'
import router from '@/router'
import type { TerminalResizeFrame } from '@/types'

type TerminalStatus = 'connecting' | 'open' | 'reconnecting' | 'closed' | 'failed'

const props = withDefaults(defineProps<{ sessionId: string; showStatus?: boolean }>(), { showStatus: true })
const emit = defineEmits<{ status: [status: TerminalStatus] }>()

const MAX_RECONNECTS = 5
/** Close codes after which reconnecting is pointless (policy / auth / session gone). */
const FATAL_CLOSE_CODES = new Set([1008, 4001, 4003, 4004, 4401, 4403, 4404])

const prefs = usePreferencesStore()
const host = ref<HTMLDivElement | null>(null)
const status = ref<TerminalStatus>('connecting')
const attempts = ref(0)

let term: Terminal | null = null
let fitAddon: FitAddon | null = null
let ws: WebSocket | null = null
let resizeObserver: ResizeObserver | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let lastSize = { cols: 0, rows: 0 }
let disposed = false

const statusLabel = computed(() => {
  switch (status.value) {
    case 'connecting':
      return 'Ulanmoqda…'
    case 'reconnecting':
      return `Qayta ulanish ${attempts.value}/${MAX_RECONNECTS}…`
    case 'closed':
      return 'Ulanish yopildi'
    case 'failed':
      return "Ulanib bo'lmadi"
    default:
      return 'Ulangan'
  }
})

function setStatus(next: TerminalStatus) {
  status.value = next
  emit('status', next)
}

function wsUrl(sessionId: string): string {
  const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
  return `${proto}://${window.location.host}/ws?sessionId=${encodeURIComponent(sessionId)}`
}

function sendResize() {
  if (!term || !ws || ws.readyState !== WebSocket.OPEN) return
  const { cols, rows } = term
  if (cols === lastSize.cols && rows === lastSize.rows) return
  lastSize = { cols, rows }
  const frame: TerminalResizeFrame = { type: 'resize', cols, rows }
  ws.send(JSON.stringify(frame))
}

function fit() {
  if (!fitAddon || !host.value || host.value.clientWidth === 0) return
  try {
    fitAddon.fit()
  } catch {
    // Element not measurable yet (hidden tab); next resize will fit.
  }
  sendResize()
}

function sendInput(data: string) {
  if (ws && ws.readyState === WebSocket.OPEN) ws.send(data)
}

function onTerminalKey(e: KeyboardEvent): boolean {
  if (e.type !== 'keydown' || !e.ctrlKey || !e.shiftKey || e.altKey || e.metaKey) return true
  const key = e.key.toLowerCase()
  if (key === 'c') {
    const selection = term?.getSelection()
    if (selection) navigator.clipboard?.writeText(selection).catch(() => undefined)
    e.preventDefault()
    return false
  }
  if (key === 'v') {
    e.preventDefault()
    navigator.clipboard
      ?.readText()
      .then((text) => {
        if (text) term?.paste(text)
      })
      .catch(() => undefined)
    return false
  }
  return true
}

function focus() {
  term?.focus()
}

function clear() {
  term?.clear()
}

function closeSocket() {
  if (reconnectTimer !== null) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
  if (ws) {
    ws.onopen = ws.onmessage = ws.onerror = ws.onclose = null
    ws.close()
    ws = null
  }
}

function connect() {
  closeSocket()
  if (disposed || !props.sessionId) return
  setStatus(attempts.value > 0 ? 'reconnecting' : 'connecting')
  lastSize = { cols: 0, rows: 0 }

  const socket = new WebSocket(wsUrl(props.sessionId))
  socket.binaryType = 'arraybuffer'
  ws = socket

  socket.onopen = () => {
    attempts.value = 0
    setStatus('open')
    fit()
    // Do not steal focus from another control (e.g. a button the user is on).
    const active = document.activeElement
    if (!active || active === document.body || host.value?.contains(active)) focus()
  }

  socket.onmessage = (event: MessageEvent<string | ArrayBuffer>) => {
    if (!term) return
    if (typeof event.data === 'string') term.write(event.data)
    else term.write(new Uint8Array(event.data))
  }

  socket.onclose = (event: CloseEvent) => {
    if (disposed || ws !== socket) return
    ws = null
    if (event.code === 4401) {
      // Account disabled or password changed elsewhere: the session is no longer valid.
      setStatus('closed')
      useAuthStore().clear()
      void router.replace({ name: 'login', query: { redirect: router.currentRoute.value.fullPath } })
      return
    }
    if (FATAL_CLOSE_CODES.has(event.code) || attempts.value >= MAX_RECONNECTS) {
      setStatus(attempts.value >= MAX_RECONNECTS ? 'failed' : 'closed')
      term?.write('\r\n\x1b[31mUlanish yopildi. Sahifani yangilang yoki sessiyani qayta yarating.\x1b[0m\r\n')
      return
    }
    attempts.value += 1
    setStatus('reconnecting')
    const delay = Math.min(1000 * 2 ** (attempts.value - 1), 10000)
    reconnectTimer = setTimeout(connect, delay)
  }
}

function createTerminal() {
  if (!host.value) return
  term = new Terminal({
    cursorBlink: true,
    fontSize: prefs.fontSize,
    fontFamily: '"JetBrains Mono", ui-monospace, SFMono-Regular, Menlo, monospace',
    lineHeight: 1.2,
    theme: prefs.xtermTheme,
    scrollback: 2000,
    convertEol: true,
    allowProposedApi: false,
    rightClickSelectsWord: true
  })
  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.open(host.value)

  // Registered once per terminal (not per socket) so reconnects never duplicate input.
  term.onData(sendInput)

  // Ctrl+Shift+C / Ctrl+Shift+V copy and paste (plain Ctrl+C must stay SIGINT).
  term.attachCustomKeyEventHandler(onTerminalKey)

  resizeObserver = new ResizeObserver(() => fit())
  resizeObserver.observe(host.value)

  // Refit once the web font is ready so cell metrics are correct.
  document.fonts?.ready.then(() => fit()).catch(() => undefined)
  fit()
}

function destroy() {
  disposed = true
  closeSocket()
  resizeObserver?.disconnect()
  resizeObserver = null
  term?.dispose()
  term = null
  fitAddon = null
}

watch(
  () => props.sessionId,
  (next, prev) => {
    if (next === prev || disposed) return
    attempts.value = 0
    term?.reset()
    connect()
  }
)

watch(
  () => [prefs.xtermTheme, prefs.fontSize] as const,
  ([theme, size]) => {
    if (!term) return
    term.options.theme = theme
    term.options.fontSize = size
    fit()
  }
)

onMounted(() => {
  disposed = false
  createTerminal()
  connect()
})

onBeforeUnmount(destroy)

defineExpose({ focus, fit, clear })
</script>
