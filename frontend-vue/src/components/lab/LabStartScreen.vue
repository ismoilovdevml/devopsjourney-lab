<!--
  LabStartScreen — shown in the terminal area when there is no running session for this task.
  Starts a container via the session store (parent handles the call).
  Props:
    task: TaskDetail
    starting: boolean
    error?: string | null
    otherTaskId?: string | null     (a session is running for another task; starting stops it)
  Emits: start
-->
<template>
  <section aria-label="Muhit" class="flex min-h-[60vh] min-w-0 flex-1 flex-col bg-terminal lg:min-h-0">
    <div class="flex h-10 items-center border-b border-divider bg-surface-2 px-4 font-mono text-meta text-muted">
      <span class="text-faint" aria-hidden="true">❯</span>
      <span class="ml-2">bash — konteyner ishga tushirilmagan</span>
    </div>

    <div class="flex flex-1 items-center justify-center p-6">
      <div class="w-full max-w-[460px] rounded-panel border border-border bg-surface-2 p-6 sm:p-7">
        <p class="font-mono text-[13px] text-muted">
          <span class="text-accent">root@lab</span>:<span class="text-info">~</span>$ lab start {{ task.id }}
        </p>
        <h2 class="mt-3 font-mono text-lg font-semibold text-text">Muhit hali ishga tushirilmagan</h2>
        <p class="mt-2 text-[13.5px] text-text-2">
          Siz uchun alohida Linux konteyner yaratiladi<template v-if="task.image">
            (<code class="font-mono text-meta text-violet">{{ task.image }}</code>)</template>.
          Terminal brauzerda ochiladi, ishni tugatgach «Yechimni tekshirish» tugmasini bosing.
        </p>

        <p
          v-if="otherTaskId"
          class="mt-4 flex gap-2 rounded-btn border border-warn/30 bg-warn/[0.08] px-3.5 py-3 text-[13px] text-[#F5D9A3]"
        >
          <AlertTriangle class="mt-0.5 h-4 w-4 shrink-0 text-warn" aria-hidden="true" />
          <span>
            Sizda <RouterLink :to="`/lab/${otherTaskId}`" class="font-mono text-warn underline underline-offset-2">{{ otherTaskId }}</RouterLink>
            uchun faol sessiya bor. Yangi muhit uni to'xtatadi.
          </span>
        </p>

        <p v-if="error" role="alert" class="mt-4 rounded-btn border border-danger/40 bg-danger/10 px-3.5 py-3 text-[13px] text-danger">
          {{ error }}
        </p>

        <Button class="mt-5" variant="primary" size="lg" block :icon="Power" :loading="starting" @click="emit('start')">
          {{ starting ? 'Ishga tushirilmoqda…' : 'Muhitni ishga tushirish' }}
        </Button>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { AlertTriangle, Power } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import type { TaskDetail } from '@/types'

defineProps<{
  task: TaskDetail
  starting: boolean
  error?: string | null
  otherTaskId?: string | null
}>()

const emit = defineEmits<{ start: [] }>()
</script>
