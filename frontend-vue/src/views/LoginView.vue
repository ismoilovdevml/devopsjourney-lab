<!--
  LoginView — public /login page (design/Login.dc.html).
  Left: LoginShowcase (pitch, servers.svg, topic chips). Right: AuthCard (Kirish / Ro'yxatdan o'tish).
  On success navigates to safeRedirect(route.query.redirect) or '/'.
-->
<template>
  <main id="main" class="flex min-h-screen flex-col text-text lg:flex-row lg:flex-wrap">
    <LoginShowcase />
    <section
      aria-label="Hisobga kirish"
      class="flex items-center justify-center px-4 py-8 sm:px-6 lg:flex-[1_1_460px] lg:py-12"
    >
      <AuthCard
        v-model:mode="mode"
        :loading="auth.isLoading"
        :server-error="serverError"
        @submit="submit"
        @clear-server-error="serverError = null"
      />
    </section>
  </main>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AuthCard from '@/components/login/AuthCard.vue'
import LoginShowcase from '@/components/login/LoginShowcase.vue'
import { mapAuthError, type AuthErrorView, type AuthMode, type AuthSubmitPayload } from '@/components/login/validation'
import { safeRedirect } from '@/router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

const mode = ref<AuthMode>(route.query.mode === 'register' ? 'register' : 'login')
const serverError = ref<AuthErrorView | null>(null)

async function submit(payload: AuthSubmitPayload) {
  if (auth.isLoading) return
  serverError.value = null
  try {
    if (payload.mode === 'login') {
      await auth.login({ username: payload.username, password: payload.password })
    } else {
      await auth.register({ username: payload.username, email: payload.email, password: payload.password })
    }
  } catch (err) {
    serverError.value = mapAuthError(err, payload.mode)
    return
  }
  await router.replace(safeRedirect(route.query.redirect) ?? '/')
}
</script>
