import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/services/api'
import type { Task } from '@/types'

export const useTaskStore = defineStore('task', () => {
  const tasks = ref<Task[]>([])
  const selectedTask = ref<Task | null>(null)
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  async function loadTasks() {
    isLoading.value = true
    error.value = null
    try {
      tasks.value = await api.getTasks()
      console.log('Tasks loaded:', tasks.value.length)
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Failed to load tasks'
      console.error('Failed to load tasks:', err)
    } finally {
      isLoading.value = false
    }
  }

  async function loadTask(id: string) {
    isLoading.value = true
    error.value = null
    try {
      selectedTask.value = await api.getTask(id)
      console.log('Task loaded:', selectedTask.value?.title)
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Failed to load task'
      console.error('Failed to load task:', err)
    } finally {
      isLoading.value = false
    }
  }

  function selectTask(task: Task) {
    selectedTask.value = task
  }

  function clearSelection() {
    selectedTask.value = null
  }

  return {
    tasks,
    selectedTask,
    isLoading,
    error,
    loadTasks,
    loadTask,
    selectTask,
    clearSelection
  }
})
