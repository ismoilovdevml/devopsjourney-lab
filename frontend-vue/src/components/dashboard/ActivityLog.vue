<!--
  ActivityLog — "tail -f activity.log" terminal card with the user's own recent events
  (built by buildActivity() from progress data; nothing is invented).
  Props:
    events: ActivityEvent[]
    loading?: boolean
-->
<template>
  <TerminalWindow title="tail -f activity.log">
    <ul v-if="loading" class="m-0 flex list-none flex-col gap-2.5 p-0" aria-busy="true" aria-label="Yuklanmoqda">
      <li v-for="i in 4" :key="i" class="flex gap-2.5">
        <Skeleton width="40px" height="14px" />
        <Skeleton :width="`${50 + i * 8}%`" height="14px" />
      </li>
    </ul>

    <ul v-else class="m-0 flex list-none flex-col gap-2.5 p-0">
      <li v-for="e in events" :key="e.key" class="flex gap-2.5">
        <span
          class="w-[42px] shrink-0 text-faint"
          :title="e.at === null ? 'Vaqti saqlanmagan' : undefined"
        >{{ formatWhen(e.at, now) }}</span>
        <span :class="['min-w-0 break-words', TONE[e.tone]]">{{ e.text }}</span>
      </li>
      <li v-if="!events.length" class="flex gap-2.5 text-muted">
        <span class="w-[42px] shrink-0 text-faint">--:--</span>
        <span># hali faoliyat yo'q — birinchi vazifani boshlang</span>
      </li>
      <li class="flex gap-2.5" aria-hidden="true">
        <span class="w-[42px] shrink-0" />
        <span class="cursor-blink inline-block h-3.5 w-2 bg-accent/80" />
      </li>
    </ul>
  </TerminalWindow>
</template>

<script setup lang="ts">
import Skeleton from '@/components/ui/Skeleton.vue'
import TerminalWindow from '@/components/ui/TerminalWindow.vue'
import { formatWhen, type ActivityEvent, type ActivityTone } from './dashboard'

defineProps<{
  events: ActivityEvent[]
  loading?: boolean
}>()

const now = new Date()

const TONE: Record<ActivityTone, string> = {
  done: 'text-accent',
  check: 'text-info',
  achievement: 'text-violet',
  session: 'text-text-2'
}
</script>

<style scoped>
.cursor-blink {
  animation: blink 1.1s steps(1) infinite;
}

@keyframes blink {
  50% {
    opacity: 0;
  }
}
</style>
