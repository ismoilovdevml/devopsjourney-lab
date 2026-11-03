<!--
  DashboardView — route "/". Hero with the task to continue, 4 KPI tiles, learning path,
  activity log and quick links. Data: GET /api/tasks (task store), GET /api/progress,
  GET /api/leaderboard (me), session store (running task + last check).
-->
<template>
  <AppLayout>
    <template v-if="error">
      <div class="flex flex-col gap-1.5">
        <span class="cmd-line"><span class="text-accent">{{ username || 'user' }}@lab</span>:~$ whoami</span>
        <h1 class="page-title">Bosh sahifa</h1>
      </div>
      <EmptyState
        tone="danger"
        :icon="ServerCrash"
        command="exit code 1"
        title="Ma'lumotlarni yuklab bo'lmadi"
        :description="error"
      >
        <template #actions>
          <Button variant="primary" :icon="RotateCw" :loading="loading" @click="load(true)">Qayta urinish</Button>
        </template>
      </EmptyState>
    </template>

    <template v-else>
      <DashboardHero
        :username="username"
        :task="continueTask"
        :task-index="continueIndex"
        :total-tasks="tasks.length"
        :checks="continueChecks"
        :active="continueActive"
        :all-done="allDone"
        :total-score="progress?.totalScore"
        :loading="loading"
      />

      <section aria-label="Statistika" class="grid gap-4 [grid-template-columns:repeat(auto-fit,minmax(230px,1fr))]">
        <StatTile
          label="Umumiy ball"
          :value="progress?.totalScore ?? '—'"
          :of="progress ? ` / ${progress.totalPossible}` : undefined"
          :icon="Star"
          color="#3DDC97"
          :loading="loading"
        />
        <StatTile
          label="Yakunlangan"
          :value="progress?.completed ?? '—'"
          :of="progress ? ` / ${progress.totalTasks}` : undefined"
          :icon="Check"
          color="#38BDF8"
          :loading="loading"
        />
        <StatTile
          label="Laboratoriyada"
          :value="progress ? formatLabTime(progress.timeInLabSeconds) : '—'"
          :icon="Clock3"
          color="#A78BFA"
          :loading="loading"
        />
        <StatTile
          label="Reyting"
          :value="rank ? `#${rank}` : '—'"
          :icon="Trophy"
          color="#F5B342"
          :loading="loading || rankLoading"
        />
      </section>

      <div class="flex flex-wrap items-start gap-6">
        <LearningPath
          :tasks="tasks"
          :active-task-id="session.taskId"
          :total-score="progress?.totalScore"
          :total-possible="progress?.totalPossible"
          :loading="loading"
        />

        <aside class="flex min-w-0 flex-[1_1_340px] flex-col gap-6" aria-label="Faoliyat va havolalar">
          <ActivityLog :events="activity" :loading="loading" />
          <QuickLinks
            :rank="rank"
            :achievements="achievementCount"
            :session-label="session.isActive ? session.remainingLabel : null"
            :session-task-id="session.taskId"
            :loading="loading || rankLoading"
          />
        </aside>
      </div>
    </template>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Check, Clock3, RotateCw, ServerCrash, Star, Trophy } from 'lucide-vue-next'
import AppLayout from '@/components/ui/AppLayout.vue'
import Button from '@/components/ui/Button.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import StatTile from '@/components/ui/StatTile.vue'
import ActivityLog from '@/components/dashboard/ActivityLog.vue'
import DashboardHero from '@/components/dashboard/DashboardHero.vue'
import LearningPath from '@/components/dashboard/LearningPath.vue'
import QuickLinks from '@/components/dashboard/QuickLinks.vue'
import { buildActivity, checkProgress, formatLabTime, pickContinueTask } from '@/components/dashboard/dashboard'
import { api, errorMessage } from '@/services/api'
import { useAuthStore } from '@/stores/auth'
import { useSessionStore } from '@/stores/session'
import { useTaskStore } from '@/stores/task'
import type { ProgressResponse } from '@/types'

const auth = useAuthStore()
const session = useSessionStore()
const taskStore = useTaskStore()

const progress = ref<ProgressResponse | null>(null)
const rank = ref<number | null>(null)
const loading = ref(true)
const rankLoading = ref(true)
const error = ref<string | null>(null)

const username = computed(() => auth.user?.username ?? '')
const tasks = computed(() => taskStore.tasks)

const allDone = computed(() => tasks.value.length > 0 && tasks.value.every((t) => t.completed))

const continueTask = computed(() => (allDone.value ? undefined : pickContinueTask(tasks.value, session.taskId)))

const continueIndex = computed(() => {
  const t = continueTask.value
  return t ? tasks.value.findIndex((x) => x.id === t.id) + 1 : undefined
})

const continueActive = computed(() => {
  const t = continueTask.value
  return !!t && (t.id === session.taskId || (t.in_progress && !t.completed))
})

const continueChecks = computed(() => {
  const t = continueTask.value
  if (!t) return undefined
  const entry = progress.value?.progress.find((p) => p.task_id === t.id)
  return checkProgress(t, entry, session.lastCheck, session.taskId)
})

const activity = computed(() =>
  progress.value
    ? buildActivity({
        progress: progress.value.progress,
        tasks: tasks.value,
        achievements: progress.value.achievements ?? [],
        session: session.session
      })
    : []
)

const achievementCount = computed(() => {
  const list = progress.value?.achievements
  if (!list || list.length === 0) return null
  return { earned: list.filter((a) => a.earned).length, total: list.length }
})

async function loadRank(): Promise<void> {
  rankLoading.value = true
  try {
    const board = await api.getLeaderboard('all', 50)
    rank.value = board.me?.rank ?? null
  } catch {
    // Rank is secondary information: the tile shows "—" instead of failing the page.
    rank.value = null
  } finally {
    rankLoading.value = false
  }
}

async function load(force = false): Promise<void> {
  loading.value = true
  error.value = null
  void loadRank()
  try {
    const [, prog] = await Promise.all([taskStore.loadTasks(force), api.getProgress()])
    if (taskStore.error) throw new Error(taskStore.error)
    progress.value = prog
  } catch (err) {
    error.value = errorMessage(err, "Bosh sahifa ma'lumotlarini olishda xatolik yuz berdi. Qayta urinib ko'ring.")
  } finally {
    loading.value = false
  }
}

// Task flags (completed / in_progress / unlocked) change after checks, so always refetch here.
onMounted(() => load(true))
</script>
