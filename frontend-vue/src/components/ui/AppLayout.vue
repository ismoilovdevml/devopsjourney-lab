<!--
  AppLayout — standard page chrome: skip link, AppHeader, <main> container (max 1320px, 24px gaps),
  AppFooterStatus. Used by Dashboard, Tasks, Progress, Leaderboard, Profile.
  (Login, Lab and Admin have their own chrome per the design.)
  Props:
    mainClass?: string     (extra classes for <main>, e.g. to change the gap)
    footer?: boolean       (show the status footer; default true)
    containers?, image?    (passed through to AppFooterStatus)
  Slots: default (page content).
-->
<template>
  <div class="flex min-h-screen flex-col">
    <a
      href="#main"
      class="sr-only-focusable fixed left-4 top-2 z-50 rounded-ctl bg-accent px-3 py-2 font-medium text-accent-ink"
    >
      Asosiy kontentga o'tish
    </a>
    <AppHeader />
    <main id="main" tabindex="-1" :class="['container-page flex flex-1 flex-col gap-6 pb-14 pt-8 outline-none', mainClass]">
      <slot />
    </main>
    <AppFooterStatus v-if="footer" :containers="containers" :image="image" />
  </div>
</template>

<script setup lang="ts">
import AppHeader from './AppHeader.vue'
import AppFooterStatus from './AppFooterStatus.vue'

withDefaults(
  defineProps<{
    mainClass?: string
    footer?: boolean
    containers?: { running: number; limit: number }
    image?: string
  }>(),
  { footer: true }
)
</script>
