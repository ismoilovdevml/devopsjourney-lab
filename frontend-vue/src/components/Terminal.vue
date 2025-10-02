<template>
  <div class="h-full w-full bg-[#1e1e1e] relative">
    <div class="h-full w-full" ref="terminalRef"></div>

    <!-- Connection Status Indicator -->
    <div class="absolute top-4 right-4 z-10">
      <div v-if="!isConnected" class="bg-yellow-900/90 text-yellow-200 px-3 py-1.5 rounded-lg text-sm flex items-center gap-2 shadow-lg backdrop-blur-sm">
        <div class="w-2 h-2 bg-yellow-400 rounded-full animate-pulse"></div>
        <span v-if="reconnectAttempts > 0">
          Reconnecting... ({{ reconnectAttempts }}/{{ maxReconnectAttempts }})
        </span>
        <span v-else>Connecting...</span>
      </div>
      <div v-else class="bg-green-900/90 text-green-200 px-3 py-1.5 rounded-lg text-sm flex items-center gap-2 shadow-lg backdrop-blur-sm">
        <div class="w-2 h-2 bg-green-400 rounded-full animate-pulse"></div>
        Connected
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import 'xterm/css/xterm.css'

const props = defineProps<{
  sessionId: string
}>()

const terminalRef = ref<HTMLDivElement>()
const isConnected = ref(false)
const reconnectAttempts = ref(0)
const maxReconnectAttempts = 5

let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null
let ws: WebSocket | null = null
let reconnectTimeout: number | null = null
let pasteListener: ((e: ClipboardEvent) => void) | null = null

function initTerminal() {
  if (!terminalRef.value) return

  terminal = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: {
      background: '#1e1e1e',
      foreground: '#d4d4d4',
      cursor: '#ffffff',
      black: '#000000',
      red: '#cd3131',
      green: '#0dbc79',
      yellow: '#e5e510',
      blue: '#2472c8',
      magenta: '#bc3fbc',
      cyan: '#11a8cd',
      white: '#e5e5e5',
      brightBlack: '#666666',
      brightRed: '#f14c4c',
      brightGreen: '#23d18b',
      brightYellow: '#f5f543',
      brightBlue: '#3b8eea',
      brightMagenta: '#d670d6',
      brightCyan: '#29b8db',
      brightWhite: '#ffffff',
    },
    convertEol: true,
    scrollback: 1000,
    allowProposedApi: true,
    rightClickSelectsWord: true,
    windowsMode: false,
  })

  fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  terminal.open(terminalRef.value)

  setTimeout(() => {
    if (fitAddon) fitAddon.fit()
    if (terminal) terminal.focus()
  }, 100)

  connectWebSocket()

  // Handle window resize
  window.addEventListener('resize', handleResize)

  // Handle paste from clipboard
  pasteListener = handlePaste
  if (terminalRef.value) {
    terminalRef.value.addEventListener('paste', pasteListener)
  }
}

function handlePaste(e: ClipboardEvent) {
  e.preventDefault()
  const text = e.clipboardData?.getData('text')
  if (text && ws && ws.readyState === WebSocket.OPEN) {
    ws.send(text)
  }
}

function connectWebSocket() {
  const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
  const port = import.meta.env.VITE_WS_PORT || '8080'
  const token = localStorage.getItem('auth_token')

  if (!token) {
    console.error('No auth token found')
    if (terminal) {
      terminal.write('\r\n\x1b[31m✗ Authentication required. Please login.\x1b[0m\r\n')
    }
    return
  }

  const wsUrl = `${proto}://${window.location.hostname}:${port}/ws?sessionId=${props.sessionId}&token=${token}`

  console.log('Connecting to WebSocket:', wsUrl.replace(token, 'TOKEN'))

  ws = new WebSocket(wsUrl)
  ws.binaryType = 'arraybuffer'

  ws.onopen = () => {
    console.log('WebSocket connected')
    isConnected.value = true
    reconnectAttempts.value = 0

    if (terminal) {
      terminal.write('\r\n\x1b[32m✓ Connected to terminal\x1b[0m\r\n\r\n')
      terminal.focus()
    }
  }

  ws.onmessage = (event) => {
    if (!terminal) return

    if (typeof event.data === 'string') {
      terminal.write(event.data)
    } else if (event.data instanceof ArrayBuffer) {
      terminal.write(new Uint8Array(event.data))
    }
  }

  ws.onerror = (error) => {
    console.error('WebSocket connection error:', error)
    isConnected.value = false
  }

  ws.onclose = (event) => {
    console.log('WebSocket closed:', event.code, event.reason)
    isConnected.value = false

    if (terminal) {
      terminal.write('\r\n\x1b[33m⚠ Connection closed\x1b[0m\r\n')
    }

    // Attempt reconnection
    if (reconnectAttempts.value < maxReconnectAttempts) {
      reconnectAttempts.value++
      const delay = Math.min(1000 * Math.pow(2, reconnectAttempts.value), 10000)

      if (terminal) {
        terminal.write(`\x1b[33mReconnecting in ${delay/1000}s... (${reconnectAttempts.value}/${maxReconnectAttempts})\x1b[0m\r\n`)
      }

      reconnectTimeout = window.setTimeout(() => {
        console.log(`Reconnect attempt ${reconnectAttempts.value}/${maxReconnectAttempts}`)
        connectWebSocket()
      }, delay)
    } else {
      if (terminal) {
        terminal.write('\r\n\x1b[31m✗ Connection failed after maximum retry attempts.\x1b[0m\r\n')
        terminal.write('\x1b[31mPlease refresh the page or restart your session.\x1b[0m\r\n')
      }
    }
  }

  if (terminal) {
    terminal.onData((data) => {
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(data)
      }
    })
  }

  if (terminalRef.value) {
    terminalRef.value.addEventListener('click', () => {
      if (terminal) terminal.focus()
    })
  }
}

function handleResize() {
  if (fitAddon) fitAddon.fit()
}

function cleanup() {
  console.log('Cleaning up terminal')

  // Clear reconnect timeout
  if (reconnectTimeout !== null) {
    clearTimeout(reconnectTimeout)
    reconnectTimeout = null
  }

  // Remove event listeners
  window.removeEventListener('resize', handleResize)

  if (terminalRef.value && pasteListener) {
    terminalRef.value.removeEventListener('paste', pasteListener)
  }

  // Close WebSocket
  if (ws) {
    ws.close()
    ws = null
  }

  // Dispose terminal
  if (terminal) {
    terminal.dispose()
    terminal = null
  }

  // Reset state
  isConnected.value = false
  reconnectAttempts.value = 0
}

watch(() => props.sessionId, (newId, oldId) => {
  if (newId !== oldId) {
    cleanup()
    setTimeout(() => {
      initTerminal()
    }, 100)
  }
})

onMounted(() => {
  initTerminal()
})

onUnmounted(() => {
  cleanup()
})
</script>

<style scoped>
/* Enable text selection in terminal */
:deep(.xterm) {
  user-select: text !important;
  -webkit-user-select: text !important;
  -moz-user-select: text !important;
}

:deep(.xterm-selection) {
  background-color: rgba(255, 255, 255, 0.3) !important;
}
</style>
