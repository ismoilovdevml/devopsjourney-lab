<template>
  <div class="h-screen flex flex-col bg-gray-50 dark:bg-dark-950 transition-colors duration-300">
    <!-- Header -->
    <header class="bg-white dark:bg-dark-900 border-b-2 border-gray-100 dark:border-dark-800 px-6 py-4 shadow-sm transition-colors duration-300">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="bg-gradient-to-br from-blue-500 to-indigo-600 dark:from-indigo-600 dark:to-purple-600 p-2 rounded-xl shadow-lg">
            <Terminal class="w-7 h-7 text-white" />
          </div>
          <div>
            <h1 class="text-2xl font-bold bg-gradient-to-r from-blue-600 to-indigo-600 dark:from-blue-400 dark:to-indigo-400 bg-clip-text text-transparent">
              DevOps Journey Lab
            </h1>
            <p class="text-xs text-gray-500 dark:text-gray-400">DevOps Learning Platform</p>
          </div>
        </div>

        <div class="flex items-center gap-3">
          <div v-if="sessionStore.timeRemaining > 0" class="flex items-center gap-2 px-4 py-2 bg-blue-50 dark:bg-indigo-900/30 rounded-lg border border-blue-200 dark:border-indigo-700 transition-colors">
            <Clock class="w-4 h-4 text-blue-600 dark:text-indigo-400" />
            <span class="font-mono font-semibold text-blue-900 dark:text-indigo-300">{{ formatTime(sessionStore.timeRemaining) }}</span>
          </div>

          <ThemeToggle />

          <button @click="showProgress = true" class="px-4 py-2 bg-gradient-to-r from-purple-600 to-pink-600 dark:from-purple-500 dark:to-pink-500 text-white hover:from-purple-700 hover:to-pink-700 dark:hover:from-purple-600 dark:hover:to-pink-600 rounded-lg font-semibold transition-all duration-300 hover:scale-105 hover:shadow-lg flex items-center gap-2">
            <Trophy class="w-4 h-4" />
            Progress
          </button>

          <div class="text-right">
            <p class="text-xs text-gray-500 dark:text-gray-400">Logged in as</p>
            <p class="font-semibold text-gray-800 dark:text-gray-200">{{ authStore.user?.username }}</p>
          </div>

          <button @click="handleLogout" class="px-4 py-2 bg-gray-200 dark:bg-dark-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-dark-600 rounded-lg font-semibold transition-all duration-300 hover:scale-105 flex items-center gap-2">
            <LogOut class="w-4 h-4" />
            Logout
          </button>
        </div>
      </div>
    </header>

    <!-- Main Content -->
    <div class="flex-1 flex overflow-hidden">
      <!-- Sidebar -->
      <div class="w-[450px] bg-white dark:bg-dark-900 border-r-2 border-gray-100 dark:border-dark-800 flex flex-col shadow-sm transition-colors duration-300">
        <div class="p-6 border-b-2 border-gray-100 dark:border-dark-800 transition-colors">
          <h2 class="text-xl font-bold text-gray-800 dark:text-gray-200 mb-4 flex items-center gap-2">
            <Trophy class="w-5 h-5 text-yellow-500 dark:text-yellow-400" />
            Challenge Navigation
          </h2>

          <!-- Current Task Display -->
          <div v-if="currentTask" class="mb-4 p-4 bg-gradient-to-r from-blue-50 to-indigo-50 dark:from-indigo-900/30 dark:to-purple-900/30 rounded-xl border-2 border-blue-100 dark:border-indigo-700 transition-colors">
            <div class="flex items-center justify-between mb-2">
              <span class="text-xs font-semibold text-blue-600 dark:text-indigo-400">Task {{ currentTaskIndex + 1 }} of {{ taskStore.tasks.length }}</span>
              <span v-if="currentTask.completed" class="text-xs px-2 py-1 bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400 rounded-full">✓ Completed</span>
            </div>
            <h3 class="text-lg font-bold text-gray-800 dark:text-gray-200">{{ currentTask.title }}</h3>
          </div>

          <!-- Navigation Buttons -->
          <div class="flex gap-2 mb-4">
            <button
              @click="handlePrevious"
              :disabled="!canGoPrevious"
              class="flex-1 flex items-center justify-center gap-2 px-4 py-2 bg-gray-200 dark:bg-dark-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-dark-600 rounded-lg font-semibold transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed hover:scale-105"
            >
              <ChevronLeft class="w-4 h-4" />
              Previous
            </button>
            <button
              @click="handleSkip"
              :disabled="!canGoNext"
              class="flex-1 flex items-center justify-center gap-2 px-4 py-2 bg-gray-200 dark:bg-dark-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-dark-600 rounded-lg font-semibold transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed hover:scale-105"
            >
              Skip
              <SkipForward class="w-4 h-4" />
            </button>
            <button
              @click="handleNext"
              :disabled="!canGoNext"
              class="flex-1 flex items-center justify-center gap-2 px-4 py-2 bg-gray-200 dark:bg-dark-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-dark-600 rounded-lg font-semibold transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed hover:scale-105"
            >
              Next
              <ChevronRight class="w-4 h-4" />
            </button>
          </div>

          <div class="flex gap-3">
            <button
              v-if="!sessionStore.hasActiveSession"
              @click="handleStart"
              :disabled="!currentTask || sessionStore.isCreating"
              class="flex-1 flex items-center justify-center gap-2 px-4 py-3 bg-gradient-to-r from-blue-600 to-indigo-600 dark:from-indigo-600 dark:to-purple-600 text-white hover:from-blue-700 hover:to-indigo-700 dark:hover:from-indigo-700 dark:hover:to-purple-700 rounded-lg font-bold shadow-lg hover:shadow-xl transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed hover:scale-105"
            >
              <Loader2 v-if="sessionStore.isCreating" class="w-5 h-5 animate-spin" />
              <Play v-else class="w-5 h-5" />
              {{ sessionStore.isCreating ? 'Starting...' : 'Start Session' }}
            </button>

            <button
              v-else
              @click="handleReset"
              class="flex-1 flex items-center justify-center gap-2 px-4 py-3 bg-gradient-to-r from-red-600 to-red-700 dark:from-red-500 dark:to-red-600 text-white hover:from-red-700 hover:to-red-800 dark:hover:from-red-600 dark:hover:to-red-700 rounded-lg font-bold shadow-lg hover:shadow-xl transition-all duration-300 hover:scale-105"
            >
              <RotateCcw class="w-5 h-5" />
              Stop Session
            </button>

            <button
              @click="handleCheck"
              :disabled="!sessionStore.hasActiveSession || sessionStore.isChecking"
              class="flex-1 flex items-center justify-center gap-2 px-4 py-3 bg-gradient-to-r from-green-600 to-emerald-600 dark:from-green-500 dark:to-emerald-500 text-white hover:from-green-700 hover:to-emerald-700 dark:hover:from-green-600 dark:hover:to-emerald-600 disabled:from-gray-300 disabled:to-gray-300 dark:disabled:from-dark-700 dark:disabled:to-dark-700 rounded-lg font-bold shadow-lg hover:shadow-xl transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed hover:scale-105"
            >
              <Loader2 v-if="sessionStore.isChecking" class="w-5 h-5 animate-spin" />
              <CheckCircle v-else class="w-5 h-5" />
              {{ sessionStore.isChecking ? 'Checking...' : 'Check' }}
            </button>
          </div>

          <div v-if="sessionStore.error" class="mt-4 p-3 bg-red-50 dark:bg-red-900/20 border-2 border-red-200 dark:border-red-800 rounded-xl flex items-start gap-2 transition-colors">
            <AlertCircle class="w-5 h-5 text-red-600 dark:text-red-400 flex-shrink-0 mt-0.5" />
            <p class="text-sm text-red-800 dark:text-red-300">{{ sessionStore.error }}</p>
          </div>
        </div>

        <!-- Task Description -->
        <div class="flex-1 overflow-auto p-6 bg-gray-50/50 dark:bg-dark-900/50 transition-colors">
          <TaskDetail v-if="taskStore.selectedTask" :task="taskStore.selectedTask" />
          <CheckResults v-if="sessionStore.checkResult" :result="sessionStore.checkResult" />
        </div>
      </div>

      <!-- Terminal Area -->
      <div class="flex-1 bg-[#1e1e1e] dark:bg-black relative transition-colors duration-300">
        <TerminalComponent v-if="sessionStore.sessionId" :session-id="sessionStore.sessionId" />
        <div v-else class="h-full flex items-center justify-center text-gray-400 dark:text-gray-600">
          <div class="text-center">
            <Terminal class="w-20 h-20 mx-auto mb-6 opacity-30 dark:opacity-20" />
            <p class="text-xl mb-2 dark:text-gray-500">Terminal Ready</p>
            <p class="text-sm text-gray-500 dark:text-gray-700">Start a session to begin</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Progress Modal -->
    <div v-if="showProgress" class="fixed inset-0 bg-black/50 dark:bg-black/70 backdrop-blur-sm flex items-center justify-center p-8 z-50 transition-colors">
      <div class="max-w-2xl w-full">
        <ProgressSummary @close="showProgress = false" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useTaskStore } from '@/stores/task'
import { useSessionStore } from '@/stores/session'
import TaskDetail from '@/components/TaskDetail.vue'
import CheckResults from '@/components/CheckResults.vue'
import TerminalComponent from '@/components/Terminal.vue'
import ProgressSummary from '@/components/ProgressSummary.vue'
import ThemeToggle from '@/components/ThemeToggle.vue'
import {
  Terminal, Trophy, Clock, LogOut, Play, CheckCircle,
  Loader2, RotateCcw, AlertCircle, ChevronLeft, ChevronRight, SkipForward
} from 'lucide-vue-next'

const router = useRouter()
const authStore = useAuthStore()
const taskStore = useTaskStore()
const sessionStore = useSessionStore()

const currentTaskIndex = ref(0)
const currentTask = computed(() => taskStore.tasks[currentTaskIndex.value])
const showProgress = ref(false)

// Computed properties for button states - navigation allowed during session
const canGoPrevious = computed(() => currentTaskIndex.value > 0)
const canGoNext = computed(() => {
  return taskStore.tasks.length > 0 &&
         currentTaskIndex.value < taskStore.tasks.length - 1
})

onMounted(() => {
  console.log('LabView mounted, loading tasks...')

  // Clear any stale errors from previous sessions
  sessionStore.error = null
  authStore.error = null

  taskStore.loadTasks().then(() => {
    console.log('Tasks loaded, count:', taskStore.tasks.length)
    if (taskStore.tasks.length > 0) {
      console.log('Loading first task:', taskStore.tasks[0].id)
      taskStore.loadTask(taskStore.tasks[0].id)
    }
  })
})

function handlePrevious() {
  if (currentTaskIndex.value > 0) {
    currentTaskIndex.value--
    taskStore.loadTask(taskStore.tasks[currentTaskIndex.value].id)
  }
}

function handleNext() {
  if (currentTaskIndex.value < taskStore.tasks.length - 1) {
    currentTaskIndex.value++
    taskStore.loadTask(taskStore.tasks[currentTaskIndex.value].id)
  }
}

function handleSkip() {
  handleNext()
}

async function handleStart() {
  console.log('Start button clicked')
  console.log('Current task:', currentTask.value)

  // Clear any previous errors before starting
  sessionStore.error = null

  if (!currentTask.value) {
    console.error('No current task!')
    return
  }
  try {
    await sessionStore.createSession(currentTask.value.id)
    console.log('Session created successfully')
  } catch (error) {
    console.error('Failed to start session:', error)
  }
}

async function handleCheck() {
  try {
    await sessionStore.checkSolution(currentTask.value?.id)
  } catch (error) {
    console.error('Failed to check solution:', error)
  }
}

async function handleReset() {
  await sessionStore.deleteSession()
  // Don't clear task selection, just reload current task to keep description visible
  if (currentTask.value) {
    taskStore.loadTask(currentTask.value.id)
  }
}

function handleLogout() {
  if (sessionStore.hasActiveSession) {
    sessionStore.deleteSession()
  }
  authStore.logout()
  router.push('/login')
}

function formatTime(seconds: number): string {
  const hrs = Math.floor(seconds / 3600)
  const mins = Math.floor((seconds % 3600) / 60)
  const secs = seconds % 60
  return `${hrs.toString().padStart(2, '0')}:${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
}
</script>
