<!--
  LeaderboardView — /leaderboard?period=week|month|all
  Data: GET /api/leaderboard?period=&limit=50 ({entries, me}) + GET /api/tasks (task count for the
  per-row progress cells, via the task store). Responses for a period that is no longer selected
  are dropped (request token).
-->
<template>
  <AppLayout>
    <div class="mx-auto flex w-full max-w-[952px] flex-col gap-6">
      <PageHeader command="$ sort -rn scores | head" title="Reyting">
        <template #actions>
          <SegmentedControl
            :model-value="period"
            :options="periodOptions"
            ariaLabel="Davr"
            @update:model-value="selectPeriod"
          />
        </template>
      </PageHeader>

      <EmptyState
        v-if="error"
        tone="danger"
        :icon="AlertTriangle"
        title="Reytingni yuklab bo'lmadi"
        :description="error"
      >
        <template #actions>
          <Button :icon="RotateCw" @click="load">Qayta urinish</Button>
        </template>
      </EmptyState>

      <template v-else>
        <LeaderboardPodium
          v-if="loading || entries.length > 0"
          :entries="entries"
          :me-id="meId"
          :loading="loading"
        />

        <LeaderboardMeBanner :me="me" :gap="gap" :loading="loading" />

        <EmptyState
          v-if="!loading && entries.length === 0"
          :icon="Trophy"
          command="$ sort -rn scores | head → 0 natija"
          title="Bu davrda hali hech kim ball to'plamagan"
          description="Vazifani birinchi bo'lib bajaring va reyting boshiga chiqing."
        />

        <LeaderboardTable
          v-else
          :entries="entries"
          :me-id="meId"
          :outside-me="outsideMe"
          :total-tasks="totalTasks"
          :loading="loading"
        />
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { AlertTriangle, RotateCw, Trophy } from 'lucide-vue-next'
import AppLayout from '@/components/ui/AppLayout.vue'
import PageHeader from '@/components/ui/PageHeader.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import Button from '@/components/ui/Button.vue'
import SegmentedControl from '@/components/ui/SegmentedControl.vue'
import LeaderboardPodium from '@/components/leaderboard/LeaderboardPodium.vue'
import LeaderboardMeBanner from '@/components/leaderboard/LeaderboardMeBanner.vue'
import LeaderboardTable from '@/components/leaderboard/LeaderboardTable.vue'
import { PERIODS, gapToNextRank, parsePeriod } from '@/components/leaderboard/leaderboard'
import { api, errorMessage } from '@/services/api'
import { useAuthStore } from '@/stores/auth'
import { useTaskStore } from '@/stores/task'
import type { LeaderboardPeriod, LeaderboardResponse } from '@/types'

const LIMIT = 50

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const taskStore = useTaskStore()

const periodOptions = PERIODS.map((p) => ({ value: p.value, label: p.label }))
const period = computed<LeaderboardPeriod>(() => parsePeriod(route.query.period))

const data = ref<LeaderboardResponse | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)

/** Incremented per request; a response is applied only if its token is still the latest. */
let requestToken = 0

async function load(): Promise<void> {
  const token = ++requestToken
  loading.value = true
  error.value = null
  data.value = null
  try {
    const res = await api.getLeaderboard(period.value, LIMIT)
    if (token !== requestToken) return
    data.value = res
  } catch (err) {
    if (token !== requestToken) return
    error.value = errorMessage(err, "Server javob bermadi. Birozdan so'ng qayta urinib ko'ring.")
  } finally {
    if (token === requestToken) loading.value = false
  }
}

function selectPeriod(raw: string | number): void {
  const value = parsePeriod(raw)
  if (value === period.value && route.query.period === value) return
  void router.replace({ query: { ...route.query, period: value } })
}

watch(period, () => void load())

onMounted(() => {
  void load()
  void taskStore.loadTasks()
})

onBeforeUnmount(() => {
  // Drop any in-flight response.
  requestToken++
})

const entries = computed(() => data.value?.entries ?? [])
const me = computed(() => data.value?.me ?? null)
const meId = computed(() => me.value?.user_id ?? auth.user?.id ?? null)

/** Current user's entry when they scored but are not in the returned top list. */
const outsideMe = computed(() => {
  const m = me.value
  if (!m || m.score <= 0) return null
  return entries.value.some((e) => e.user_id === m.user_id) ? null : m
})

const gap = computed(() => (me.value ? gapToNextRank(me.value, entries.value) : null))

/** Number of tasks for the per-row cells; null while unknown (cells are then hidden). */
const totalTasks = computed(() => (taskStore.loaded ? taskStore.tasks.length : null))
</script>
