<template>
  <div v-if="progressData" class="bg-gradient-to-br from-blue-500 to-indigo-600 dark:from-indigo-600 dark:to-purple-700 text-white rounded-2xl p-6 shadow-2xl dark:shadow-indigo-900/50 transition-all duration-300">
    <div class="flex items-center justify-between mb-4">
      <h3 class="text-2xl font-bold flex items-center gap-2">
        <Trophy class="w-7 h-7" />
        Your Progress
      </h3>
      <div class="text-right">
        <div class="text-4xl font-bold">{{ progressData.percentage }}%</div>
        <div class="text-sm opacity-90">Completed</div>
      </div>
    </div>

    <div class="bg-white/20 dark:bg-white/10 rounded-lg p-4 backdrop-blur-sm">
      <div class="flex items-center justify-between mb-2">
        <span class="text-lg font-semibold">Total Score</span>
        <span class="text-3xl font-bold">{{ progressData.totalScore }} / {{ progressData.totalPossible }}</span>
      </div>

      <div class="w-full bg-white/30 dark:bg-white/20 rounded-full h-3 overflow-hidden">
        <div
          class="bg-white h-full rounded-full transition-all duration-500"
          :style="{ width: `${progressData.percentage}%` }"
        ></div>
      </div>
    </div>

    <div v-if="progressData.progress.length > 0" class="mt-4 space-y-2">
      <div class="text-sm font-semibold mb-2 opacity-90">Completed Tasks:</div>
      <div
        v-for="item in progressData.progress"
        :key="item.task_id"
        class="bg-white/10 rounded-lg p-3 flex items-center justify-between backdrop-blur-sm"
      >
        <div class="flex items-center gap-2">
          <CheckCircle v-if="item.passed" class="w-5 h-5 text-green-300" />
          <XCircle v-else class="w-5 h-5 text-orange-300" />
          <span class="font-medium">{{ getTaskTitle(item.task_id) }}</span>
        </div>
        <div class="text-lg font-bold">{{ item.score }} / {{ item.total }}</div>
      </div>
    </div>

    <button
      @click="$emit('close')"
      class="mt-4 w-full bg-white text-blue-600 hover:bg-blue-50 py-2 px-4 rounded-lg font-semibold transition-colors"
    >
      Continue Learning
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Trophy, CheckCircle, XCircle } from 'lucide-vue-next'
import { api } from '@/services/api'
import type { ProgressResponse } from '@/types'

const progressData = ref<ProgressResponse | null>(null)

const taskTitles: Record<string, string> = {
  'nginx-health': 'Nginx Web Server',
  'docker-basics': 'Docker Management',
  'systemd-service': 'Systemd Service',
  'user-management': 'User Management',
  'firewall-config': 'Firewall Config',
  'cron-backup': 'Cron & Backup'
}

function getTaskTitle(taskId: string): string {
  return taskTitles[taskId] || taskId
}

onMounted(async () => {
  try {
    console.log('Loading progress...')
    const data = await api.getProgress()
    console.log('Progress data received:', data)

    // Backend returns { progress, totalScore, totalPossible, percentage }
    // No need to recalculate
    progressData.value = {
      progress: data.progress || [],
      totalScore: data.totalScore || 0,
      totalPossible: data.totalPossible || 0,
      percentage: data.percentage || 0
    }
    console.log('Progress loaded successfully:', progressData.value)
  } catch (err: any) {
    console.error('Failed to load progress:', err)
    console.error('Error details:', err.response?.data || err.message)

    // Show empty state instead of crashing
    progressData.value = {
      progress: [],
      totalScore: 0,
      totalPossible: 210, // Total from all tasks
      percentage: 0
    }
  }
})
</script>
