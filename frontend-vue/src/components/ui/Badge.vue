<!--
  Badge — pill status chip.
  Props:
    variant?: 'done' | 'active' | 'open' | 'locked'  (task status; default 'open')
    label?: string    (defaults to the Uzbek status label: Yakunlangan / Jarayonda / Yangi / Qulflangan)
    solid?: boolean   (opaque style for use on top of a cover image)
    color?: string    (hex; overrides variant with a generic tinted badge, e.g. '#F5B342' for "sudo")
    icon?: boolean    (show the status glyph: check / dot / lock; default true)
  Slots: default (overrides label).
-->
<template>
  <span
    :class="[
      'inline-flex shrink-0 items-center gap-1.5 whitespace-nowrap rounded-full px-2.5 py-0.5 text-xs font-semibold leading-5',
      color ? 'border' : solid ? SOLID[variant] : TINTED[variant]
    ]"
    :style="color ? { color, backgroundColor: tint(color, 0.1), borderColor: tint(color, 0.32) } : undefined"
  >
    <template v-if="icon && !color">
      <Check v-if="variant === 'done'" class="h-3 w-3" aria-hidden="true" />
      <span v-else-if="variant === 'active'" class="h-1.5 w-1.5 rounded-full bg-current" aria-hidden="true" />
      <Lock v-else-if="variant === 'locked'" class="h-3 w-3" aria-hidden="true" />
    </template>
    <slot>{{ label ?? TASK_STATUS_LABEL[variant] }}</slot>
  </span>
</template>

<script setup lang="ts">
import { Check, Lock } from 'lucide-vue-next'
import { TASK_STATUS_LABEL, type TaskStatus } from '@/constants/tasks'
import { tint } from './utils'

withDefaults(
  defineProps<{
    variant?: TaskStatus
    label?: string
    solid?: boolean
    color?: string
    icon?: boolean
  }>(),
  { variant: 'open', icon: true }
)

const TINTED: Record<TaskStatus, string> = {
  done: 'text-accent bg-accent/10 border border-accent/30',
  active: 'text-info bg-info/10 border border-info/30',
  open: 'text-text-2 bg-surface-3 border border-border-strong',
  locked: 'text-muted bg-transparent border border-dashed border-border-stronger'
}

const SOLID: Record<TaskStatus, string> = {
  done: 'text-accent-ink bg-accent',
  active: 'text-[#04121C] bg-info',
  open: 'text-text bg-[rgba(10,14,20,.75)]',
  locked: 'text-warn bg-[rgba(10,14,20,.8)]'
}
</script>
