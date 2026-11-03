<!--
  ProfileHeaderCard — avatar ring with initials, username (page <h1>), role · member since · level,
  and stat chips (xp, completed tasks, earned achievements). servers.svg is faded decorative art.
  Props:
    user: User
    progress: ProgressResponse | null
    loading?: boolean   (progress still loading -> chip skeletons)
    error?: boolean     (progress failed -> short inline note instead of chips)
  Emits: retry (reload progress)
-->
<template>
  <section
    aria-labelledby="profile-username"
    class="relative overflow-hidden rounded-hero border border-border bg-[rgba(15,21,32,0.88)]"
  >
    <img
      src="/assets/servers.svg"
      alt=""
      aria-hidden="true"
      class="pointer-events-none absolute -right-5 -top-8 hidden w-[380px] select-none opacity-35 sm:block"
    />
    <div class="relative flex flex-wrap items-center gap-[22px] px-5 py-6 sm:px-8 sm:py-7">
      <span
        class="flex h-[84px] w-[84px] shrink-0 items-center justify-center rounded-full border-2 border-accent bg-surface-3 font-mono text-[26px] font-bold text-accent shadow-glow"
        aria-hidden="true"
      >
        {{ initials }}
      </span>
      <div class="flex min-w-0 flex-col gap-1">
        <span class="cmd-line">$ id</span>
        <h1 id="profile-username" class="break-all font-mono text-[22px] font-bold leading-tight text-text">
          {{ user.username }}
        </h1>
        <p class="text-muted">
          {{ ROLE_LABEL[user.role] ?? user.role }}<template v-if="since"> · {{ since }}</template>
          <template v-if="level">
            · <span :style="{ color: level.color }">{{ level.title }}</span>
          </template>
        </p>
        <div v-if="loading" class="mt-1.5 flex gap-2" aria-hidden="true">
          <Skeleton width="64px" height="24px" rounded="rounded-full" />
          <Skeleton width="88px" height="24px" rounded="rounded-full" />
          <Skeleton width="72px" height="24px" rounded="rounded-full" />
        </div>
        <p v-else-if="error" class="mt-1.5 text-[13px] text-muted">
          Natijalarni yuklab bo'lmadi.
          <button type="button" class="link font-medium" @click="emit('retry')">Qayta urinish</button>
        </p>
        <ul v-else-if="progress" class="mt-1.5 flex flex-wrap gap-2 font-mono text-meta" aria-label="Statistika">
          <li class="rounded-full border border-border-strong bg-surface-3 px-2.5 py-[3px] text-accent">
            {{ progress.totalScore }} xp
          </li>
          <li class="rounded-full border border-border-strong bg-surface-3 px-2.5 py-[3px] text-info">
            {{ progress.completed }}/{{ progress.totalTasks }} vazifa
          </li>
          <li class="rounded-full border border-border-strong bg-surface-3 px-2.5 py-[3px] text-violet">
            {{ earned }} yutuq
          </li>
        </ul>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import type { ProgressResponse, User } from '@/types'
import { ROLE_LABEL, levelFor, memberSince } from './profile'

const props = defineProps<{
  user: User
  progress: ProgressResponse | null
  loading?: boolean
  error?: boolean
}>()

const emit = defineEmits<{ retry: [] }>()

const initials = computed(() => props.user.username.slice(0, 2).toUpperCase())
const since = computed(() => memberSince(props.user.created_at))
const level = computed(() => (props.progress ? levelFor(props.progress.completed) : null))
const earned = computed(() => (props.progress?.achievements ?? []).filter((a) => a.earned).length)
</script>
