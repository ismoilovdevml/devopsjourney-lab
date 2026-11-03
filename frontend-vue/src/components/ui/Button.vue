<!--
  Button — the only button style in the app.
  Props:
    variant?: 'primary' | 'secondary' | 'danger' | 'ghost'  (default 'secondary'; danger = red outline)
    size?: 'sm' | 'md' | 'lg'   (heights 36 / 42 / 46px; default 'md')
    type?: 'button' | 'submit' | 'reset'  (default 'button')
    loading?: boolean   (shows spinner, sets aria-busy, disables)
    disabled?: boolean
    block?: boolean     (full width)
    icon?: Component    (leading lucide icon)  iconRight?: Component (trailing)
    to?: RouteLocationRaw  (renders <RouterLink>)   href?: string (renders <a>)
    ariaLabel?: string  (required for icon-only buttons)
  Slots: default (label). Emits: click(MouseEvent).
-->
<template>
  <RouterLink
    v-if="to && !isDisabled"
    :to="to"
    :class="classes"
    :aria-label="ariaLabel"
    @click="onClick"
  >
    <component :is="icon" v-if="icon" class="h-4 w-4 shrink-0" aria-hidden="true" />
    <slot />
    <component :is="iconRight" v-if="iconRight" class="h-4 w-4 shrink-0" aria-hidden="true" />
  </RouterLink>
  <a
    v-else-if="href && !isDisabled"
    :href="href"
    :class="classes"
    :aria-label="ariaLabel"
    @click="onClick"
  >
    <component :is="icon" v-if="icon" class="h-4 w-4 shrink-0" aria-hidden="true" />
    <slot />
    <component :is="iconRight" v-if="iconRight" class="h-4 w-4 shrink-0" aria-hidden="true" />
  </a>
  <button
    v-else
    :type="type"
    :class="classes"
    :disabled="isDisabled"
    :aria-busy="loading || undefined"
    :aria-label="ariaLabel"
    @click="onClick"
  >
    <Loader2 v-if="loading" class="h-4 w-4 shrink-0 animate-spin" aria-hidden="true" />
    <component :is="icon" v-else-if="icon" class="h-4 w-4 shrink-0" aria-hidden="true" />
    <slot />
    <component :is="iconRight" v-if="iconRight && !loading" class="h-4 w-4 shrink-0" aria-hidden="true" />
  </button>
</template>

<script setup lang="ts">
import { computed, type Component } from 'vue'
import type { RouteLocationRaw } from 'vue-router'
import { Loader2 } from 'lucide-vue-next'

const props = withDefaults(
  defineProps<{
    variant?: 'primary' | 'secondary' | 'danger' | 'ghost'
    size?: 'sm' | 'md' | 'lg'
    type?: 'button' | 'submit' | 'reset'
    loading?: boolean
    disabled?: boolean
    block?: boolean
    icon?: Component
    iconRight?: Component
    to?: RouteLocationRaw
    href?: string
    ariaLabel?: string
  }>(),
  { variant: 'secondary', size: 'md', type: 'button' }
)

const emit = defineEmits<{ click: [event: MouseEvent] }>()

const isDisabled = computed(() => props.disabled || props.loading)

const VARIANTS = {
  primary: 'bg-accent text-accent-ink font-semibold shadow-glow hover:bg-accent-hover',
  secondary:
    'bg-surface-solid text-text border border-border-strong hover:border-border-stronger hover:bg-surface-3',
  danger: 'bg-transparent text-danger border border-danger/50 hover:bg-danger/10 hover:border-danger',
  ghost: 'bg-transparent text-muted hover:text-text hover:bg-surface-3'
} as const

const SIZES = {
  sm: 'h-9 px-3 text-[13px] gap-1.5 rounded-ctl',
  md: 'h-[42px] px-4 text-sm gap-2 rounded-btn',
  lg: 'h-[46px] px-5 text-[15px] gap-2 rounded-btn'
} as const

const classes = computed(() => [
  'inline-flex select-none items-center justify-center whitespace-nowrap font-medium transition-colors',
  'disabled:cursor-not-allowed disabled:opacity-50 disabled:shadow-none',
  VARIANTS[props.variant],
  SIZES[props.size],
  props.block ? 'w-full' : ''
])

function onClick(event: MouseEvent) {
  if (isDisabled.value) {
    event.preventDefault()
    return
  }
  emit('click', event)
}
</script>
