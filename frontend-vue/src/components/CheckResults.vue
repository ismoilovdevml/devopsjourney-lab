<template>
  <div class="mt-8 space-y-4">
    <div class="flex items-center justify-between">
      <h3 class="text-xl font-bold text-gray-800 dark:text-gray-200">Results</h3>
      <div class="flex items-center gap-2 px-4 py-2 bg-blue-50 dark:bg-blue-900/30 rounded-lg border border-blue-200 dark:border-blue-700">
        <Trophy class="w-5 h-5 text-blue-600 dark:text-blue-400" />
        <span class="font-bold text-blue-900 dark:text-blue-300">{{ result.score }}/{{ result.total }}</span>
      </div>
    </div>

    <div class="space-y-3">
      <div
        v-for="(check, idx) in result.results"
        :key="idx"
        :class="[
          'p-4 rounded-xl border-2 transition',
          check.ok
            ? 'bg-green-50 dark:bg-green-900/20 border-green-300 dark:border-green-700'
            : 'bg-red-50 dark:bg-red-900/20 border-red-300 dark:border-red-700'
        ]"
      >
        <div class="flex items-center gap-3">
          <CheckCircle v-if="check.ok" class="w-6 h-6 text-green-600 dark:text-green-400 flex-shrink-0" />
          <XCircle v-else class="w-6 h-6 text-red-600 dark:text-red-400 flex-shrink-0" />
          <span :class="['font-semibold', check.ok ? 'text-green-900 dark:text-green-300' : 'text-red-900 dark:text-red-300']">
            {{ check.name }}
          </span>
        </div>
        <pre
          v-if="check.output"
          class="mt-3 text-xs bg-white dark:bg-dark-800 p-3 rounded-lg overflow-x-auto border border-gray-200 dark:border-dark-600 text-gray-700 dark:text-gray-300 font-mono"
        >{{ check.output }}</pre>
      </div>
    </div>

    <div
      :class="[
        'p-6 rounded-xl text-center font-bold text-lg',
        result.passed
          ? 'bg-gradient-to-r from-green-500 to-emerald-500 dark:from-green-600 dark:to-emerald-600 text-white shadow-lg'
          : 'bg-gradient-to-r from-red-500 to-rose-500 dark:from-red-600 dark:to-rose-600 text-white shadow-lg'
      ]"
    >
      <div class="flex items-center justify-center gap-3">
        <CheckCircle v-if="result.passed" class="w-8 h-8" />
        <XCircle v-else class="w-8 h-8" />
        <span>{{ result.passed ? 'ALL TESTS PASSED!' : 'SOME TESTS FAILED' }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { CheckCircle, XCircle, Trophy } from 'lucide-vue-next'
import type { CheckResponse } from '@/types'

defineProps<{
  result: CheckResponse
}>()
</script>
