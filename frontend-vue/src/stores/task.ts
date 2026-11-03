/**
 * Tasks list + per-task detail cache.
 *
 * State:   tasks (TaskSummary[], sorted by order), details (id -> TaskDetail), loaded, isLoading, error
 * Getters: byId(id), nextTask(id), completedCount
 * Actions: loadTasks(force?) (cached after first success), loadTask(id, force?) -> TaskDetail,
 *          invalidate() (call after a passing check / session start so flags refresh), clear()
 * loadTasks stores a message in `error` instead of throwing; loadTask throws `ApiError`.
 */
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api, errorMessage } from '@/services/api'
import type { TaskDetail, TaskSummary } from '@/types'

export const useTaskStore = defineStore('task', () => {
  const tasks = ref<TaskSummary[]>([])
  const details = ref<Record<string, TaskDetail>>({})
  const loaded = ref(false)
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  let inflight: Promise<TaskSummary[]> | null = null

  const completedCount = computed(() => tasks.value.filter((t) => t.completed).length)

  function byId(id: string | null | undefined): TaskSummary | undefined {
    if (!id) return undefined
    return tasks.value.find((t) => t.id === id)
  }

  /** The task after `id` in `order`, or undefined if it is the last one. */
  function nextTask(id: string): TaskSummary | undefined {
    const idx = tasks.value.findIndex((t) => t.id === id)
    return idx >= 0 ? tasks.value[idx + 1] : undefined
  }

  async function loadTasks(force = false): Promise<TaskSummary[]> {
    if (loaded.value && !force) return tasks.value
    if (inflight) return inflight
    isLoading.value = true
    error.value = null
    inflight = (async () => {
      try {
        const list = await api.getTasks()
        tasks.value = [...list].sort((a, b) => a.order - b.order)
        loaded.value = true
      } catch (err) {
        error.value = errorMessage(err, "Vazifalarni yuklab bo'lmadi.")
      } finally {
        isLoading.value = false
        inflight = null
      }
      return tasks.value
    })()
    return inflight
  }

  async function loadTask(id: string, force = false): Promise<TaskDetail> {
    const cached = details.value[id]
    if (cached && !force) return cached
    const detail = await api.getTask(id)
    details.value = { ...details.value, [id]: detail }
    return detail
  }

  /** Mark cached data stale; the next load refetches. */
  function invalidate(): void {
    loaded.value = false
    details.value = {}
  }

  function clear(): void {
    tasks.value = []
    details.value = {}
    loaded.value = false
    error.value = null
  }

  return {
    tasks,
    details,
    loaded,
    isLoading,
    error,
    completedCount,
    byId,
    nextTask,
    loadTasks,
    loadTask,
    invalidate,
    clear
  }
})
