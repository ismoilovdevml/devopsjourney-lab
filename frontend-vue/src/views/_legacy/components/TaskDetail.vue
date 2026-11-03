<template>
  <div class="space-y-4">
    <div class="flex items-start justify-between mb-4">
      <div>
        <h3 class="text-2xl font-bold text-gray-800 dark:text-gray-200">{{ task.title }}</h3>
      </div>
      <div class="flex gap-2">
        <span class="px-3 py-1 bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-300 font-semibold rounded-lg text-sm border border-yellow-200 dark:border-yellow-700/50">
          {{ task.points }} ball
        </span>
        <span class="px-3 py-1 bg-purple-100 dark:bg-purple-900/30 text-purple-800 dark:text-purple-300 font-semibold rounded-lg text-sm border border-purple-200 dark:border-purple-700/50">
          {{ difficultyText }}
        </span>
      </div>
    </div>

    <div v-if="task.description" class="bg-gradient-to-br from-blue-50 to-indigo-50 dark:from-blue-900/20 dark:to-indigo-900/20 p-6 rounded-2xl border-2 border-blue-100 dark:border-blue-800/50 shadow-sm">
      <h4 class="font-bold text-gray-800 dark:text-gray-200 mb-2 flex items-center gap-2">
        <FileText class="w-5 h-5 text-blue-600 dark:text-blue-400" />
        Vazifa tavsifi
      </h4>
      <div class="markdown-content prose dark:prose-invert max-w-none" v-html="renderedDescription"></div>
    </div>

    <div v-if="task.hints && task.hints.length > 0" class="bg-gradient-to-br from-green-50 to-emerald-50 dark:from-green-900/20 dark:to-emerald-900/20 p-6 rounded-2xl border-2 border-green-100 dark:border-green-800/50 shadow-sm">
      <h4 class="font-bold text-gray-800 dark:text-gray-200 mb-3 flex items-center gap-2">
        <Lightbulb class="w-5 h-5 text-green-600 dark:text-green-400" />
        💡 Maslahatlar (Hints)
      </h4>
      <ol class="space-y-2">
        <li v-for="(hint, index) in task.hints" :key="index" class="flex gap-3">
          <span class="flex-shrink-0 w-6 h-6 bg-green-600 dark:bg-green-500 text-white rounded-full flex items-center justify-center text-sm font-bold">
            {{ index + 1 }}
          </span>
          <span class="text-gray-700 dark:text-gray-300 pt-0.5">{{ hint }}</span>
        </li>
      </ol>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { FileText, Lightbulb } from 'lucide-vue-next'
import { marked } from 'marked'
import type { Task } from '@/types'

const props = defineProps<{
  task: Task
}>()

const difficultyText = computed(() => {
  const levels = ['', 'Oson', 'O\'rta', 'Qiyin']
  return levels[props.task.difficulty] || 'Unknown'
})

const renderedDescription = computed(() => {
  if (!props.task.description) return ''
  return marked(props.task.description)
})
</script>

<style scoped>
.markdown-content :deep(h1) {
  @apply text-2xl font-bold text-gray-900 mb-3;
}

.markdown-content :deep(h2) {
  @apply text-xl font-bold text-gray-800 mt-6 mb-3;
}

.markdown-content :deep(h3) {
  @apply text-lg font-semibold text-gray-800 mt-4 mb-2;
}

.markdown-content :deep(p) {
  @apply text-gray-700 leading-relaxed mb-3;
}

.markdown-content :deep(ul) {
  @apply list-disc list-inside space-y-1 text-gray-700 mb-3;
}

.markdown-content :deep(ol) {
  @apply list-decimal list-inside space-y-1 text-gray-700 mb-3;
}

.markdown-content :deep(code) {
  @apply bg-blue-100 text-blue-800 px-2 py-0.5 rounded font-mono text-sm;
}

.markdown-content :deep(pre) {
  @apply bg-gray-900 text-gray-100 p-4 rounded-lg overflow-x-auto my-3;
}

.markdown-content :deep(pre code) {
  @apply bg-transparent text-gray-100 p-0;
}
</style>
