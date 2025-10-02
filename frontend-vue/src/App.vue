<template>
  <router-view />
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useTaskStore } from '@/stores/task'

const taskStore = useTaskStore()

onMounted(() => {
  taskStore.loadTasks()
})
</script>
