<!--
  DashboardHero — greeting ("<user>@lab:~$ whoami") + "continue" card for the current task + servers art.
  Variants: loading skeleton, continue (task given), congratulation (allDone), no tasks (neither).
  Props:
    username: string
    task?: TaskSummary            (task to continue; omitted when all are done or none exist)
    taskIndex?: number            (1-based position of `task` in the path)
    totalTasks: number
    checks?: { done: number; total: number }   (check progress of `task`)
    active?: boolean              (task has a running session / is in progress)
    allDone?: boolean             (every task completed -> congratulation variant)
    totalScore?: number           (shown in the congratulation variant)
    loading?: boolean
-->
<template>
  <section
    aria-labelledby="dash-hero-title"
    class="relative flex flex-wrap items-center overflow-hidden rounded-hero border border-border bg-[rgba(15,21,32,0.78)] backdrop-blur-[6px]"
  >
    <div class="flex min-w-0 flex-[999_1_520px] flex-col gap-4 px-5 py-7 sm:px-10 sm:py-9">
      <span class="font-mono text-[13px] text-muted">
        <span class="text-accent">{{ username || 'user' }}@lab</span>:~$ whoami
      </span>

      <h1 id="dash-hero-title" class="font-mono text-[26px] font-bold leading-[1.15] tracking-[-0.02em] text-text sm:text-hero">
        <template v-if="allDone">
          Tabriklaymiz, {{ name }}.
          <span class="text-accent">Barcha vazifalar bajarildi.</span>
        </template>
        <template v-else>
          Salom, {{ name }}.
          <span class="text-accent">Serveringiz kutyapti.</span>
        </template>
      </h1>

      <!-- Loading -->
      <div
        v-if="loading"
        class="flex max-w-[620px] flex-col gap-2.5 rounded-xl border border-border bg-surface-2 px-5 py-[18px]"
        aria-busy="true"
        aria-label="Yuklanmoqda"
      >
        <Skeleton width="220px" height="20px" />
        <Skeleton width="70%" height="24px" />
        <Skeleton height="6px" />
      </div>

      <!-- Congratulation -->
      <div
        v-else-if="allDone"
        class="flex max-w-[620px] flex-col gap-2.5 rounded-xl border border-accent/30 bg-surface-2 px-5 py-[18px]"
      >
        <div class="flex flex-wrap items-center gap-2.5">
          <Badge variant="done" label="Yakunlandi" />
          <span class="text-[13px] text-muted">{{ totalTasks }} / {{ totalTasks }} vazifa · {{ totalScore ?? 0 }} ball</span>
        </div>
        <div class="text-[19px] font-semibold text-text">Butun o'quv yo'li tugallandi</div>
        <div class="flex items-center gap-3">
          <ProgressSegments
            :total="totalTasks"
            :filled="totalTasks"
            :label="`Vazifalar: ${totalTasks} / ${totalTasks}`"
            class="flex-1"
          />
          <span class="shrink-0 font-mono text-[13px] text-muted">{{ totalTasks }}/{{ totalTasks }} tasks</span>
        </div>
      </div>

      <!-- Continue -->
      <div
        v-else-if="task"
        class="flex max-w-[620px] flex-col gap-2.5 rounded-xl border border-border bg-surface-2 px-5 py-[18px]"
      >
        <div class="flex flex-wrap items-center gap-2.5">
          <Badge v-if="active" variant="active" label="Davom etmoqda" :icon="false" />
          <Badge v-else-if="locked" variant="locked" />
          <Badge v-else variant="open" label="Keyingi vazifa" :icon="false" />
          <span class="text-[13px] text-muted">{{ meta }}</span>
        </div>
        <div class="text-[19px] font-semibold text-text">{{ task.title }}</div>
        <div v-if="checks && checks.total > 0" class="flex items-center gap-3">
          <ProgressSegments
            :total="checks.total"
            :filled="checks.done"
            :label="`Tekshiruvlar: ${checks.done} / ${checks.total}`"
            class="flex-1"
          />
          <span class="shrink-0 font-mono text-[13px] text-muted">{{ checks.done }}/{{ checks.total }} checks</span>
        </div>
        <p v-if="locked" class="text-[13px] text-muted">
          Bu vazifa oldingi vazifalarni yakunlaganingizdan so'ng ochiladi.
        </p>
      </div>

      <!-- No tasks -->
      <div
        v-else
        class="flex max-w-[620px] flex-col gap-1 rounded-xl border border-dashed border-border-stronger bg-surface-2 px-5 py-[18px]"
      >
        <span class="font-mono text-[13px] text-muted">ls ./tasks: 0 results</span>
        <span class="text-[15px] font-semibold text-text">Hozircha vazifalar yo'q</span>
        <span class="text-[13px] text-muted">Administrator vazifalarni qo'shgach, ular shu yerda paydo bo'ladi.</span>
      </div>

      <div v-if="!loading" class="flex flex-wrap gap-2.5">
        <template v-if="allDone">
          <Button variant="primary" size="lg" :icon="Trophy" :to="{ name: 'progress' }">Natijalarni ko'rish</Button>
          <Button variant="secondary" size="lg" :to="{ name: 'leaderboard' }">Reytingni ko'rish</Button>
        </template>
        <template v-else-if="task">
          <Button
            v-if="!locked"
            variant="primary"
            size="lg"
            :icon="TerminalIcon"
            :to="{ name: 'lab', params: { taskId: task.id } }"
          >
            Terminalni ochish
          </Button>
          <Button variant="secondary" size="lg" :to="{ name: 'tasks', query: { task: task.id } }">
            Topshiriqni o'qish
          </Button>
        </template>
        <Button v-else variant="secondary" size="lg" :to="{ name: 'tasks' }">Vazifalar sahifasi</Button>
      </div>
    </div>

    <div class="flex flex-[1_1_380px] justify-center px-6 pt-4">
      <img
        src="/assets/servers.svg"
        alt="Server stoykalari va terminal oynasi illyustratsiyasi"
        class="block h-auto w-full max-w-[480px]"
        width="480"
        height="300"
        decoding="async"
      />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Terminal as TerminalIcon, Trophy } from 'lucide-vue-next'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import ProgressSegments from '@/components/ui/ProgressSegments.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import { difficultyMeta, taskNumber, taskVisual } from '@/constants/tasks'
import type { TaskSummary } from '@/types'
import { displayName } from './dashboard'

const props = defineProps<{
  username: string
  task?: TaskSummary
  taskIndex?: number
  totalTasks: number
  checks?: { done: number; total: number }
  active?: boolean
  allDone?: boolean
  totalScore?: number
  loading?: boolean
}>()

const name = computed(() => displayName(props.username) || "do'stim")

const locked = computed(() => !!props.task && !props.task.unlocked && !props.task.completed && !props.active)

const meta = computed(() => {
  const t = props.task
  if (!t) return ''
  const n = taskNumber(props.taskIndex ?? t.order)
  const total = taskNumber(props.totalTasks)
  return `Vazifa ${n} / ${total} · ${taskVisual(t).categoryLabel} · ${difficultyMeta(t.difficulty).label} · ${t.points} ball`
})
</script>
