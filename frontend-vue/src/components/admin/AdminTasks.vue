<!--
  AdminTasks — task catalogue (covers, id, title, difficulty, checks, points) with
  "Qayta yuklash" (POST /api/admin/tasks/reload) which re-reads backend-go/tasks/*/task.yaml.
  Data: task store (GET /api/tasks).
  Emits: changed (after a successful reload)
  Exposes: refresh()
-->
<template>
  <section class="overflow-hidden rounded-panel border border-border bg-[rgba(15,21,32,0.88)]" aria-label="Vazifalar katalogi">
    <div class="flex flex-wrap items-center justify-between gap-3 border-b border-divider px-4 py-3">
      <span class="text-muted">
        Manba: <code class="font-mono text-meta text-violet">backend-go/tasks/*/task.yaml</code>
      </span>
      <Button :icon="reloaded === null ? RotateCw : Check" :loading="reloading" @click="reload">
        {{ reloaded === null ? 'Qayta yuklash' : `${reloaded} ta yuklandi` }}
      </Button>
    </div>

    <div v-if="store.error && !store.loaded && !store.isLoading" class="p-4">
      <SectionError :message="store.error" command="GET /api/tasks" :retrying="store.isLoading" @retry="refresh" />
    </div>

    <div v-else-if="store.loaded && store.tasks.length === 0" class="p-4">
      <EmptyState :icon="FolderOpen" command="ls backend-go/tasks → 0" title="Vazifalar topilmadi" description="task.yaml fayllarini tekshirib, qayta yuklang." />
    </div>

    <ul v-else class="grid grid-cols-[repeat(auto-fill,minmax(260px,1fr))] gap-3 p-4">
      <template v-if="!store.loaded">
        <li v-for="i in 6" :key="i" class="overflow-hidden rounded-xl border border-border bg-surface-2">
          <Skeleton height="0" rounded="rounded-none" class="!aspect-[320/110] !h-auto" />
          <div class="flex flex-col gap-2 px-3.5 py-3">
            <Skeleton width="40%" height="11px" />
            <Skeleton width="80%" />
            <Skeleton width="55%" height="11px" />
          </div>
        </li>
      </template>
      <li v-for="t in store.tasks" v-else :key="t.id" class="overflow-hidden rounded-xl border border-border bg-surface-2">
        <TaskCover :task-id="t.id" :category="t.category" aspect="320 / 110" />
        <div class="flex flex-col gap-1 px-3.5 py-3">
          <span class="font-mono text-xs text-muted">{{ taskNumber(t.order) }} · {{ t.id }}</span>
          <span class="font-semibold text-text">{{ t.title }}</span>
          <span class="text-meta text-muted">
            <span :style="{ color: difficultyMeta(t.difficulty).color }">{{ difficultyMeta(t.difficulty).label }}</span>
            · {{ t.checks_count }} tekshiruv · <span class="font-mono text-accent">+{{ t.points }} xp</span>
          </span>
        </div>
      </li>
    </ul>
  </section>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { Check, FolderOpen, RotateCw } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import TaskCover from '@/components/ui/TaskCover.vue'
import SectionError from './SectionError.vue'
import { difficultyMeta, taskNumber } from '@/constants/tasks'
import { api, errorMessage } from '@/services/api'
import { useTaskStore } from '@/stores/task'
import { useToast } from '@/composables/useToast'

const emit = defineEmits<{ changed: [] }>()

const store = useTaskStore()
const toast = useToast()

const reloading = ref(false)
const reloaded = ref<number | null>(null)
let resetTimer: ReturnType<typeof setTimeout> | null = null

async function refresh(): Promise<void> {
  await store.loadTasks(true)
}

async function reload() {
  reloading.value = true
  try {
    const res = await api.adminReloadTasks()
    reloaded.value = res.count
    toast.success(`${res.count} ta vazifa qayta yuklandi`)
    emit('changed')
    await store.loadTasks(true)
    if (resetTimer) clearTimeout(resetTimer)
    resetTimer = setTimeout(() => (reloaded.value = null), 4000)
  } catch (err) {
    toast.error(errorMessage(err, "Vazifalarni qayta yuklab bo'lmadi."))
  } finally {
    reloading.value = false
  }
}

onMounted(() => void store.loadTasks())
onBeforeUnmount(() => {
  if (resetTimer) clearTimeout(resetTimer)
})

defineExpose({ refresh, loading: reloading })
</script>
