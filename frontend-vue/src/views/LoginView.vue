<template>
  <div class="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 via-indigo-50 to-purple-50 dark:from-dark-900 dark:via-dark-800 dark:to-indigo-950 transition-colors duration-500">
    <!-- Theme Toggle - Floating Top Right -->
    <div class="fixed top-6 right-6 z-50">
      <ThemeToggle />
    </div>

    <div class="bg-white dark:bg-dark-800 p-10 rounded-3xl shadow-2xl dark:shadow-indigo-900/50 w-full max-w-md border border-gray-100 dark:border-dark-700 transition-all duration-300">
      <div class="flex items-center justify-center mb-6">
        <div class="bg-gradient-to-br from-blue-500 to-indigo-600 dark:from-indigo-600 dark:to-purple-600 p-4 rounded-2xl shadow-lg">
          <Terminal class="w-12 h-12 text-white" />
        </div>
      </div>

      <h1 class="text-4xl font-bold text-center mb-2 bg-gradient-to-r from-blue-600 to-indigo-600 dark:from-blue-400 dark:to-indigo-400 bg-clip-text text-transparent">
        DevOps Journey Lab
      </h1>
      <p class="text-center text-gray-600 dark:text-gray-400 mb-8 text-lg">
        DevOps Learning Platform
      </p>

      <div class="mb-6 flex gap-2">
        <button
          @click="isLogin = true"
          :class="isLogin ? 'bg-gradient-to-r from-blue-600 to-indigo-600 text-white shadow-lg shadow-blue-500/50' : 'bg-gray-100 dark:bg-dark-700 text-gray-700 dark:text-gray-300'"
          class="flex-1 py-2 rounded-lg font-semibold transition-all duration-300 hover:scale-105"
        >
          Login
        </button>
        <button
          @click="isLogin = false"
          :class="!isLogin ? 'bg-gradient-to-r from-blue-600 to-indigo-600 text-white shadow-lg shadow-blue-500/50' : 'bg-gray-100 dark:bg-dark-700 text-gray-700 dark:text-gray-300'"
          class="flex-1 py-2 rounded-lg font-semibold transition-all duration-300 hover:scale-105"
        >
          Register
        </button>
      </div>

      <form @submit.prevent="handleSubmit" class="space-y-6">
        <div>
          <label class="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">Username</label>
          <input
            v-model="formData.username"
            type="text"
            class="w-full px-4 py-3 rounded-xl border-2 border-gray-200 dark:border-dark-600 bg-white dark:bg-dark-700 text-gray-900 dark:text-gray-100 focus:border-blue-500 dark:focus:border-indigo-500 focus:ring-4 focus:ring-blue-500/10 dark:focus:ring-indigo-500/20 outline-none transition-all duration-300"
            placeholder="Enter username"
            required
          />
        </div>

        <div v-if="!isLogin">
          <label class="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">Email</label>
          <input
            v-model="formData.email"
            type="email"
            class="w-full px-4 py-3 rounded-xl border-2 border-gray-200 dark:border-dark-600 bg-white dark:bg-dark-700 text-gray-900 dark:text-gray-100 focus:border-blue-500 dark:focus:border-indigo-500 focus:ring-4 focus:ring-blue-500/10 dark:focus:ring-indigo-500/20 outline-none transition-all duration-300"
            placeholder="Enter email"
            required
          />
        </div>

        <div>
          <label class="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">Password</label>
          <input
            v-model="formData.password"
            type="password"
            class="w-full px-4 py-3 rounded-xl border-2 border-gray-200 dark:border-dark-600 bg-white dark:bg-dark-700 text-gray-900 dark:text-gray-100 focus:border-blue-500 dark:focus:border-indigo-500 focus:ring-4 focus:ring-blue-500/10 dark:focus:ring-indigo-500/20 outline-none transition-all duration-300"
            placeholder="Enter password"
            required
          />
        </div>

        <div v-if="authStore.error" class="p-4 bg-red-50 dark:bg-red-900/20 border-2 border-red-200 dark:border-red-800 rounded-xl">
          <p class="text-sm text-red-800 dark:text-red-300 mb-3">{{ authStore.error }}</p>
          <button
            v-if="isLogin && authStore.error.includes('not found')"
            @click="isLogin = false"
            type="button"
            class="w-full px-4 py-2 bg-blue-600 hover:bg-blue-700 dark:bg-indigo-600 dark:hover:bg-indigo-700 text-white font-semibold rounded-lg transition-all duration-300 flex items-center justify-center gap-2 shadow-lg hover:shadow-xl hover:scale-105"
          >
            <UserPlus class="w-4 h-4" />
            Ro'yxatdan o'tish / Register
          </button>
        </div>

        <button
          type="submit"
          :disabled="authStore.isLoading"
          class="w-full px-6 py-3 bg-gradient-to-r from-blue-600 to-indigo-600 dark:from-indigo-600 dark:to-purple-600 text-white font-bold rounded-xl shadow-lg hover:shadow-2xl hover:scale-105 disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:scale-100 transition-all duration-300 flex items-center justify-center gap-2"
        >
          <Loader2 v-if="authStore.isLoading" class="w-5 h-5 animate-spin" />
          <LogIn v-else class="w-5 h-5" />
          {{ isLogin ? 'Login' : 'Register' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { Terminal, LogIn, Loader2, UserPlus } from 'lucide-vue-next'
import ThemeToggle from '@/components/ThemeToggle.vue'

const router = useRouter()
const authStore = useAuthStore()

const isLogin = ref(true)
const formData = reactive({
  username: '',
  email: '',
  password: ''
})

// Clear error when switching between login/register or when typing
watch([isLogin, () => formData.username, () => formData.password, () => formData.email], () => {
  authStore.error = null
})

async function handleSubmit() {
  const success = isLogin.value
    ? await authStore.login({ username: formData.username, password: formData.password })
    : await authStore.register(formData)

  if (success) {
    router.push('/lab')
  }
}
</script>
