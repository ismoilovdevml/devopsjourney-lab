<!--
  LoginView — STUB with a minimal working sign-in form so the app is usable during development.
  Owned by the Login page agent; see design/Login.dc.html (two columns, servers.svg, login/register
  tabs). Reference implementation: views/_legacy/LoginView.vue.
-->
<template>
  <main id="main" class="flex min-h-screen items-center justify-center p-4">
    <Card as="div" class="w-full max-w-[420px]" padding="lg">
      <div class="mb-6 flex flex-col gap-4">
        <AppLogo />
        <span class="cmd-line">$ ssh student@lab</span>
        <h1 class="font-mono text-2xl font-bold">Kirish</h1>
      </div>
      <form class="flex flex-col gap-4" novalidate @submit.prevent="submit">
        <div>
          <label for="login-username" class="label">Foydalanuvchi nomi</label>
          <input id="login-username" v-model.trim="username" class="input font-mono" autocomplete="username" required />
        </div>
        <div>
          <label for="login-password" class="label">Parol</label>
          <input
            id="login-password"
            v-model="password"
            type="password"
            class="input"
            autocomplete="current-password"
            required
          />
        </div>
        <p v-if="error" class="text-[13px] text-danger" role="alert">{{ error }}</p>
        <Button type="submit" variant="primary" size="lg" block :loading="auth.isLoading">Kirish</Button>
      </form>
    </Card>
  </main>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppLogo from '@/components/ui/AppLogo.vue'
import Button from '@/components/ui/Button.vue'
import Card from '@/components/ui/Card.vue'
import { errorMessage } from '@/services/api'
import { useAuthStore } from '@/stores/auth'
import { safeRedirect } from '@/router'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

const username = ref('')
const password = ref('')
const error = ref<string | null>(null)

async function submit() {
  error.value = null
  if (!username.value || !password.value) {
    error.value = 'Foydalanuvchi nomi va parolni kiriting.'
    return
  }
  try {
    await auth.login({ username: username.value, password: password.value })
    password.value = ''
    await router.replace(safeRedirect(route.query.redirect) ?? { name: 'dashboard' })
  } catch (err) {
    error.value = errorMessage(err, "Foydalanuvchi nomi yoki parol noto'g'ri.")
  }
}
</script>
