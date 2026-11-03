<!--
  AppLogo — green ">_" square + "devops_journey/lab" wordmark, links to the dashboard.
  Props:
    to?: RouteLocationRaw   (default '/')
    suffix?: string         (muted text after "devops_journey", default '/lab'; Admin uses '/admin')
-->
<template>
  <RouterLink
    :to="to"
    class="flex shrink-0 items-center gap-2.5 font-mono text-[15px] font-semibold text-text hover:text-text"
    aria-label="DevOps Journey Lab — bosh sahifa"
  >
    <span class="flex h-[30px] w-[30px] items-center justify-center rounded-lg bg-accent shadow-[0_0_18px_rgba(61,220,151,0.45)]">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#04130C" stroke-width="2.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <path d="M5 7l5 5-5 5" />
        <path d="M13 17h6" />
      </svg>
    </span>
    <span>devops_journey<span class="text-muted">{{ suffix }}</span></span>
  </RouterLink>
</template>

<script setup lang="ts">
import type { RouteLocationRaw } from 'vue-router'

withDefaults(defineProps<{ to?: RouteLocationRaw; suffix?: string }>(), { to: '/', suffix: '/lab' })
</script>
