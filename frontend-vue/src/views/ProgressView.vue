<!--
  ProgressView (/progress) — the user's results: completion ring + KPIs, per-task table,
  skills by category, achievements, certificate (not available yet) and progress reset.
  Data: GET /api/progress + GET /api/tasks (task store).
-->
<template>
  <AppLayout>
    <PageHeader command="$ cat progress.json" title="Natijalar">
      <template #actions>
        <Button
          :icon="Download"
          disabled
          title="Sertifikat yaratish hali mavjud emas"
          :aria-label="certificateLabel"
        >
          Sertifikat (PDF)
          <span class="rounded-chip border border-border-stronger px-1.5 font-mono text-[11px] text-muted">tez orada</span>
        </Button>
      </template>
    </PageHeader>

    <EmptyState
      v-if="state === 'error'"
      tone="danger"
      :icon="AlertTriangle"
      command="cat: progress.json: xatolik"
      title="Natijalarni yuklab bo'lmadi"
      :description="errorText"
    >
      <template #actions>
        <Button variant="primary" :icon="RefreshCw" @click="load(true)">Qayta urinish</Button>
      </template>
    </EmptyState>

    <template v-else>
      <ProgressOverview :progress="progress ?? undefined" :total-attempts="totalAttempts" :loading="state === 'loading'" />

      <EmptyState
        v-if="state === 'ready' && progress && progress.progress.length === 0"
        :icon="Rocket"
        command="progress.json: []"
        title="Hali natija yo'q"
        description="Birinchi vazifani boshlang — tekshiruv natijalari, vaqt va yutuqlar shu yerda paydo bo'ladi."
      >
        <template #actions>
          <Button variant="primary" :to="{ name: 'tasks' }" :icon-right="ArrowRight">Vazifalarni ko'rish</Button>
        </template>
      </EmptyState>

      <ProgressTaskTable
        :tasks="taskStore.tasks"
        :entries="progress?.progress ?? []"
        :active-task-id="session.taskId"
        :loading="state === 'loading'"
      />

      <div class="flex flex-wrap items-start gap-6">
        <SkillBars :skills="progress?.skills ?? []" :loading="state === 'loading'" />
        <AchievementGrid :achievements="progress?.achievements ?? []" :loading="state === 'loading'" />
      </div>

      <section
        v-if="state === 'ready'"
        aria-labelledby="danger-zone-title"
        class="flex flex-wrap items-center justify-between gap-4 rounded-panel border border-danger/30 bg-surface px-5 py-5 backdrop-blur-[6px] sm:px-6"
      >
        <div class="min-w-0 max-w-2xl">
          <h2 id="danger-zone-title" class="font-mono text-[15px] font-semibold text-danger">Progressni tiklash</h2>
          <p class="mt-1 text-[13px] text-muted">
            Barcha natijalar, urinishlar va yutuqlarni o'chirib, vazifalarni boshidan boshlang.
            <template v-if="!hasProgress">Hozircha tiklanadigan natija yo'q.</template>
          </p>
        </div>
        <Button variant="danger" :icon="RotateCcw" :disabled="!hasProgress || !auth.user" @click="resetOpen = true">
          Progressni tiklash
        </Button>
      </section>
    </template>

    <ResetProgressDialog v-model:open="resetOpen" :username="auth.user?.username ?? ''" @reset="onReset" />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { AlertTriangle, ArrowRight, Download, RefreshCw, Rocket, RotateCcw } from 'lucide-vue-next'
import AppLayout from '@/components/ui/AppLayout.vue'
import Button from '@/components/ui/Button.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import PageHeader from '@/components/ui/PageHeader.vue'
import AchievementGrid from '@/components/progress/AchievementGrid.vue'
import ProgressOverview from '@/components/progress/ProgressOverview.vue'
import ProgressTaskTable from '@/components/progress/ProgressTaskTable.vue'
import ResetProgressDialog from '@/components/progress/ResetProgressDialog.vue'
import SkillBars from '@/components/progress/SkillBars.vue'
import { useToast } from '@/composables/useToast'
import { ApiError, api } from '@/services/api'
import { useAuthStore } from '@/stores/auth'
import { useSessionStore } from '@/stores/session'
import { useTaskStore } from '@/stores/task'
import type { ProgressResponse } from '@/types'

const auth = useAuthStore()
const session = useSessionStore()
const taskStore = useTaskStore()
const toast = useToast()

const state = ref<'loading' | 'error' | 'ready'>('loading')
const errorText = ref('')
const progress = ref<ProgressResponse | null>(null)
const resetOpen = ref(false)

const totalAttempts = computed(() => (progress.value?.progress ?? []).reduce((sum, p) => sum + (p.attempts || 0), 0))
const hasProgress = computed(() => (progress.value?.progress.length ?? 0) > 0)

const certificateLabel = computed(() =>
  progress.value && progress.value.percentage >= 100
    ? 'Sertifikat (PDF) — tez orada'
    : "Sertifikat (PDF) — tez orada; barcha vazifalarni yakunlaganingizdan so'ng mavjud bo'ladi"
)

let requestId = 0

async function load(force = false) {
  const id = ++requestId
  state.value = 'loading'
  try {
    const [p] = await Promise.all([api.getProgress(), taskStore.loadTasks(force)])
    if (id !== requestId) return
    if (taskStore.error) throw new Error(taskStore.error)
    progress.value = p
    state.value = 'ready'
  } catch (err) {
    if (id !== requestId) return
    // Network errors already carry an Uzbek message; server bodies are English, so use our own copy.
    errorText.value =
      err instanceof ApiError && err.status === 0
        ? err.message
        : "Server javob bermadi. Birozdan so'ng qayta urinib ko'ring."
    state.value = 'error'
  }
}

async function onReset() {
  toast.success('Progress tiklandi.')
  taskStore.invalidate()
  await load(true)
}

onMounted(() => {
  void load()
})
</script>
