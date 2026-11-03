<!--
  LabView — lab workspace (/lab/:taskId?). Full-height layout with its own top bar (no main nav).
  - No taskId: go to the active session's task, else /tasks.
  - Locked task: locked state with a link back. No session for this task: start screen.
  - Leaving the page does NOT stop the session (the header pill elsewhere keeps showing it).
  Design: design/Workspace.dc.html. Data: stores/task (detail, path), stores/session (container, check).
-->
<template>
  <div class="flex min-h-screen flex-col lg:h-screen lg:min-h-[640px]">
    <a href="#lab-main" class="sr-only-focusable absolute left-2 top-2 z-50 rounded-btn bg-accent px-3 py-2 text-accent-ink">
      Asosiy qismga o'tish
    </a>

    <LabTopBar
      :task-id="taskId"
      :tasks="taskStore.tasks"
      :session-active="hasSession"
      :remaining-seconds="session.remainingSeconds"
      :remaining-label="session.remainingLabel"
      :recreating="session.isRecreating"
      :stopping="session.isStopping"
      :checking="session.isChecking"
      @recreate="recreateOpen = true"
      @finish="finishOpen = true"
    />

    <main id="lab-main" class="flex min-h-0 flex-1 flex-col lg:flex-row">
      <div v-if="phase !== 'loading' && phase !== 'ready'" class="flex flex-1 items-center justify-center p-4 sm:p-6">
        <div class="w-full max-w-[520px]">
          <EmptyState
            v-if="phase === 'locked'"
            :icon="Lock"
            :command="`$ lab start ${taskId} → permission denied`"
            :title="task?.available === false ? 'Bu vazifa hozircha mavjud emas' : 'Bu vazifa hali qulflangan'"
            :description="task?.available === false
              ? 'Server bu vazifani xavfsiz sandbox (gVisor) bo\'lmasa ishga tushirmaydi.'
              : 'Avval oldingi vazifalarni yakunlang — keyin bu laboratoriya ochiladi.'"
          >
            <template #actions>
              <Button variant="primary" :icon="ArrowLeft" to="/tasks">Vazifalarga qaytish</Button>
            </template>
          </EmptyState>
          <EmptyState
            v-else-if="phase === 'notfound'"
            :icon="SearchX"
            :command="`$ cat tasks/${taskId} → No such file or directory`"
            title="Vazifa topilmadi"
            description="Bunday vazifa mavjud emas yoki o'chirilgan."
          >
            <template #actions>
              <Button variant="primary" :icon="ArrowLeft" to="/tasks">Vazifalarga qaytish</Button>
            </template>
          </EmptyState>
          <EmptyState
            v-else
            tone="danger"
            :icon="AlertTriangle"
            title="Vazifani yuklab bo'lmadi"
            :description="loadError ?? undefined"
          >
            <template #actions>
              <Button :icon="RefreshCw" @click="init">Qayta urinish</Button>
              <Button variant="ghost" to="/tasks">Vazifalarga qaytish</Button>
            </template>
          </EmptyState>
        </div>
      </div>

      <template v-else>
        <LabTaskPanel
          v-model:tab="tab"
          :task="task"
          :check="hasSession ? session.lastCheck : null"
          :checking="session.isChecking"
          :attempts="attempts"
          :checked-at="checkedAt"
          :can-check="hasSession"
          :next-task="nextTask"
          :next-loading="nextBusy"
          @run-check="runCheck"
          @next="nextOpen = true"
        />

        <LabTerminalPane
          v-if="hasSession && session.sessionId"
          :session-id="session.sessionId"
          :connection="connection"
          :checking="session.isChecking"
          :disabled="sessionBusy"
          @check="runCheck"
          @status="connection = $event"
        />
        <LabStartScreen
          v-else-if="task"
          :task="task"
          :starting="session.isStarting"
          :error="startError"
          :other-task-id="session.isActive ? session.taskId : null"
          @start="startSession"
        />
        <section v-else aria-label="Terminal" class="flex min-h-[60vh] flex-1 flex-col bg-terminal lg:min-h-0" aria-busy="true">
          <div class="h-10 border-b border-divider bg-surface-2" />
          <div class="flex flex-1 flex-col gap-2.5 p-5">
            <Skeleton width="55%" />
            <Skeleton width="35%" />
            <Skeleton width="45%" />
          </div>
        </section>
      </template>
    </main>

    <LabStatusBar
      :connection="hasSession ? connection : 'idle'"
      :task-id="taskId"
      :image="task?.image"
      :remaining-label="hasSession ? session.remainingLabel : null"
    />

    <ConfirmDialog
      v-model:open="recreateOpen"
      title="Muhitni qayta yaratish"
      message="Joriy konteyner o'chiriladi va toza muhit yaratiladi. Terminaldagi barcha o'zgarishlar yo'qoladi."
      confirm-label="Qayta yaratish"
      :loading="session.isRecreating"
      @confirm="recreate"
    />
    <ConfirmDialog
      v-model:open="finishOpen"
      title="Sessiyani tugatish"
      message="Konteyner to'xtatiladi va o'chiriladi. Tekshirilgan natijalar saqlanib qoladi."
      confirm-label="Tugatish"
      :loading="session.isStopping"
      @confirm="finish"
    />
    <ConfirmDialog
      v-model:open="nextOpen"
      title="Keyingi vazifaga o'tish"
      variant="primary"
      :message="nextMessage"
      confirm-label="Davom etish"
      :loading="nextBusy"
      @confirm="goNext"
    />

    <Modal v-model:open="expiredOpen" title="Sessiya muddati tugadi" :dismissible="!session.isStarting">
      <p>Konteyner vaqti tugadi va u o'chirildi. Davom etish uchun yangi muhit ishga tushiring.</p>
      <template #footer>
        <Button variant="secondary" :disabled="session.isStarting" to="/tasks">Vazifalarga qaytish</Button>
        <Button variant="primary" :icon="Power" :loading="session.isStarting" :disabled="!task" @click="restartAfterExpiry">
          Qayta ishga tushirish
        </Button>
      </template>
    </Modal>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { AlertTriangle, ArrowLeft, Lock, Power, RefreshCw, SearchX } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import Modal from '@/components/ui/Modal.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import LabStartScreen from '@/components/lab/LabStartScreen.vue'
import LabStatusBar from '@/components/lab/LabStatusBar.vue'
import LabTaskPanel from '@/components/lab/LabTaskPanel.vue'
import LabTerminalPane from '@/components/lab/LabTerminalPane.vue'
import LabTopBar from '@/components/lab/LabTopBar.vue'
import type { ConnectionState, LabTab } from '@/components/lab/types'
import { useToast } from '@/composables/useToast'
import { errorMessage, isApiStatus } from '@/services/api'
import { sessionStartMessage } from '@/services/sessionErrors'
import { useSessionStore } from '@/stores/session'
import { useTaskStore } from '@/stores/task'
import type { TaskDetail } from '@/types'

type Phase = 'loading' | 'ready' | 'notfound' | 'error' | 'locked'

const route = useRoute()
const router = useRouter()
const session = useSessionStore()
const taskStore = useTaskStore()
const toast = useToast()

const taskId = computed(() => (typeof route.params.taskId === 'string' && route.params.taskId) || null)

const task = ref<TaskDetail | null>(null)
const loadState = ref<'loading' | 'ready' | 'notfound' | 'error'>('loading')
const loadError = ref<string | null>(null)
const startError = ref<string | null>(null)
const tab = ref<LabTab>('task')
const connection = ref<ConnectionState>('connecting')
const attempts = ref(0)
const checkedAt = ref<number | null>(null)

const recreateOpen = ref(false)
const finishOpen = ref(false)
const nextOpen = ref(false)
const expiredOpen = ref(false)
const nextBusy = ref(false)

/** The session is being replaced or removed: checks must not run against it. */
const sessionBusy = computed(() => session.isStarting || session.isRecreating || session.isStopping)

const hasSession = computed(() => session.isActive && !!taskId.value && session.taskId === taskId.value)

const phase = computed<Phase>(() => {
  if (loadState.value !== 'ready') return loadState.value
  const t = task.value
  if (t && !t.unlocked && !t.completed && !hasSession.value) return 'locked'
  return 'ready'
})

const nextTask = computed(() => (taskId.value ? taskStore.nextTask(taskId.value) ?? null : null))

const nextMessage = computed(() => {
  const n = nextTask.value
  const name = n ? `«${n.title}»` : 'keyingi vazifa'
  return `Joriy muhit to'xtatiladi va ${name} uchun yangi konteyner yaratiladi.`
})

// ---------- Loading ----------

let loadToken = 0

async function init(): Promise<void> {
  const token = ++loadToken
  const id = taskId.value
  loadState.value = 'loading'
  loadError.value = null
  startError.value = null
  task.value = null
  tab.value = 'task'

  await Promise.all([taskStore.loadTasks(), session.restore().catch(() => null)])
  if (token !== loadToken) return

  if (!id) {
    if (session.taskId) await router.replace({ name: 'lab', params: { taskId: session.taskId } })
    else await router.replace({ name: 'tasks' })
    return
  }

  try {
    const detail = await taskStore.loadTask(id)
    if (token !== loadToken) return
    task.value = detail
    loadState.value = 'ready'
  } catch (err) {
    if (token !== loadToken) return
    if (isApiStatus(err, 404)) {
      loadState.value = 'notfound'
    } else {
      loadError.value = errorMessage(err, "Vazifa ma'lumotlarini olishda xatolik yuz berdi.")
      loadState.value = 'error'
    }
  }
}

watch(taskId, () => init())

watch(task, (t) => {
  if (t) document.title = `${t.title} · Laboratoriya · DevOps Journey Lab`
})

// A new container means a new terminal connection and a fresh attempt counter.
watch(
  () => session.sessionId,
  () => {
    connection.value = 'connecting'
    attempts.value = 0
    checkedAt.value = null
  }
)

watch(
  () => session.expired,
  (expired) => {
    if (expired && loadState.value === 'ready') expiredOpen.value = true
  }
)

watch(hasSession, (active) => {
  if (active) expiredOpen.value = false
})

/** Refresh completion/unlock flags (path indicator, next task) after progress changes. */
async function refreshTasks(): Promise<void> {
  taskStore.invalidate()
  await taskStore.loadTasks(true)
}

// ---------- Actions ----------

async function startSession(): Promise<void> {
  const id = taskId.value
  if (!id || session.isStarting) return
  startError.value = null
  try {
    await session.start(id)
  } catch (err) {
    startError.value = sessionStartMessage(err, "Muhitni ishga tushirib bo'lmadi. Birozdan so'ng qayta urinib ko'ring.")
  }
}

async function restartAfterExpiry(): Promise<void> {
  await startSession()
  if (!startError.value) expiredOpen.value = false
}

async function runCheck(): Promise<void> {
  if (!hasSession.value || session.isChecking || sessionBusy.value) return
  tab.value = 'results'
  try {
    const res = await session.check()
    if (!res) return
    attempts.value += 1
    checkedAt.value = Date.now()
    if (res.passed) {
      await refreshTasks()
    }
  } catch (err) {
    if (isApiStatus(err, 404)) {
      toast.error("Sessiya topilmadi — muddati tugagan bo'lishi mumkin.")
      await session.restore().catch(() => null)
    } else {
      toast.error(errorMessage(err, "Tekshiruvni bajarib bo'lmadi."))
    }
  }
}

async function recreate(): Promise<void> {
  if (session.isChecking) return
  try {
    await session.recreate()
    recreateOpen.value = false
    toast.success('Yangi muhit yaratildi.')
  } catch (err) {
    recreateOpen.value = false
    toast.error(errorMessage(err, "Muhitni qayta yaratib bo'lmadi."))
  }
}

async function finish(): Promise<void> {
  if (session.isChecking) return
  try {
    await session.stop()
    finishOpen.value = false
    toast.info('Sessiya yakunlandi.')
    await router.push({ name: 'tasks' })
  } catch (err) {
    finishOpen.value = false
    toast.error(errorMessage(err, "Sessiyani to'xtatib bo'lmadi."))
  }
}

async function goNext(): Promise<void> {
  const next = nextTask.value
  // A check still in flight would land its result on the new session (see report).
  if (!next || nextBusy.value || session.isChecking) return
  nextBusy.value = true
  try {
    await session.start(next.id)
    nextOpen.value = false
    await router.push({ name: 'lab', params: { taskId: next.id } })
  } catch (err) {
    nextOpen.value = false
    toast.error(sessionStartMessage(err, "Keyingi vazifa uchun muhit yaratib bo'lmadi."))
  } finally {
    nextBusy.value = false
  }
}

// ---------- Keyboard: Ctrl/Cmd+Enter runs the check (captured before xterm swallows Enter) ----------

function onKeydown(e: KeyboardEvent): void {
  if (e.key !== 'Enter' || !(e.ctrlKey || e.metaKey) || e.shiftKey || e.altKey) return
  if (recreateOpen.value || finishOpen.value || nextOpen.value || expiredOpen.value) return
  if (!hasSession.value) return
  e.preventDefault()
  e.stopPropagation()
  void runCheck()
}

onMounted(() => {
  window.addEventListener('keydown', onKeydown, { capture: true })
  void init()
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown, { capture: true })
})
</script>
