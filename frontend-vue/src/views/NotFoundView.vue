<!--
  NotFoundView — 404 for unknown routes. Public (works logged in or out), own minimal chrome.
-->
<template>
  <main id="main" class="flex min-h-screen flex-col items-center justify-center gap-8 p-4">
    <AppLogo :to="homeRoute" />
    <TerminalWindow title="bash — 404" tone="terminal" class="w-full max-w-[560px]">
      <div class="flex flex-col gap-1.5 text-[13px] leading-relaxed">
        <p>
          <span class="text-accent">{{ promptUser }}@lab</span><span class="text-muted">:~$</span>
          cd <span class="break-all">{{ path }}</span>
        </p>
        <p class="text-danger">bash: cd: {{ path }}: No such file or directory</p>
        <p>
          <span class="text-accent">{{ promptUser }}@lab</span><span class="text-muted">:~$</span>
          <span class="ml-1 inline-block h-4 w-2 translate-y-0.5 animate-pulse bg-accent" aria-hidden="true" />
        </p>
      </div>
    </TerminalWindow>
    <div class="flex flex-col items-center gap-2 text-center">
      <h1 class="font-mono text-title">404 · Sahifa topilmadi</h1>
      <p class="max-w-md text-sm text-muted">Siz qidirgan manzil mavjud emas yoki ko'chirilgan.</p>
    </div>
    <div class="flex flex-wrap justify-center gap-2">
      <Button variant="primary" :icon="Home" :to="homeRoute">{{ auth.isAuthenticated ? 'Bosh sahifaga' : 'Kirish sahifasiga' }}</Button>
      <Button v-if="auth.isAuthenticated" :icon="ListChecks" :to="{ name: 'tasks' }">Vazifalar</Button>
    </div>
  </main>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { Home, ListChecks } from 'lucide-vue-next'
import AppLogo from '@/components/ui/AppLogo.vue'
import Button from '@/components/ui/Button.vue'
import TerminalWindow from '@/components/ui/TerminalWindow.vue'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const route = useRoute()

const path = computed(() => route.path)
const promptUser = computed(() => auth.user?.username ?? 'guest')
const homeRoute = computed(() => (auth.isAuthenticated ? { name: 'dashboard' } : { name: 'login' }))
</script>
