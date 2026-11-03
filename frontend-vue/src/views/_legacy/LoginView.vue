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
          <label class="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
            Password
            <span v-if="!isLogin" class="text-gray-500 dark:text-gray-400 text-xs font-normal">(minimum 8 characters)</span>
            <span v-else class="text-gray-500 dark:text-gray-400 text-xs font-normal">(minimum 6 characters)</span>
          </label>
          <input
            v-model="formData.password"
            type="password"
            :minlength="isLogin ? 6 : 8"
            class="w-full px-4 py-3 rounded-xl border-2 border-gray-200 dark:border-dark-600 bg-white dark:bg-dark-700 text-gray-900 dark:text-gray-100 focus:border-blue-500 dark:focus:border-indigo-500 focus:ring-4 focus:ring-blue-500/10 dark:focus:ring-indigo-500/20 outline-none transition-all duration-300"
            :placeholder="isLogin ? 'Enter password (min 6 chars)' : 'Enter password (min 8 chars)'"
            required
          />
          <p v-if="formData.password && formData.password.length < (isLogin ? 6 : 8)" class="mt-1 text-xs text-red-600 dark:text-red-400">
            Password must be at least {{ isLogin ? 6 : 8 }} characters
          </p>
        </div>

        <!-- Error Message - Beautiful Design -->
        <Transition
          enter-active-class="transition-all duration-300 ease-out"
          enter-from-class="opacity-0 scale-95 -translate-y-2"
          enter-to-class="opacity-100 scale-100 translate-y-0"
          leave-active-class="transition-all duration-200 ease-in"
          leave-from-class="opacity-100 scale-100"
          leave-to-class="opacity-0 scale-95"
        >
          <div v-if="authStore.error" class="relative overflow-hidden">
            <!-- Animated Background Gradient -->
            <div class="absolute inset-0 bg-gradient-to-r from-red-50 via-orange-50 to-red-50 dark:from-red-900/20 dark:via-orange-900/20 dark:to-red-900/20 animate-gradient"></div>

            <!-- Content -->
            <div class="relative p-5 border-2 border-red-200 dark:border-red-700 rounded-2xl shadow-lg backdrop-blur-sm">
              <div class="flex items-start gap-3 mb-3">
                <!-- Icon with pulse animation -->
                <div class="flex-shrink-0 mt-0.5">
                  <div class="relative">
                    <div class="absolute inset-0 bg-red-500 rounded-full opacity-20 animate-ping"></div>
                    <AlertCircle class="relative w-5 h-5 text-red-600 dark:text-red-400" />
                  </div>
                </div>

                <!-- Error Text -->
                <div class="flex-1">
                  <h4 class="text-sm font-bold text-red-900 dark:text-red-300 mb-1">
                    {{ isLogin ? 'Login Failed' : 'Registration Failed' }}
                  </h4>
                  <p class="text-sm text-red-800 dark:text-red-300 leading-relaxed">
                    {{ authStore.error }}
                  </p>
                </div>

                <!-- Close Button -->
                <button
                  @click="authStore.error = null"
                  class="flex-shrink-0 p-1 hover:bg-red-200 dark:hover:bg-red-800/30 rounded-lg transition-colors"
                  type="button"
                >
                  <XCircle class="w-4 h-4 text-red-600 dark:text-red-400" />
                </button>
              </div>

              <!-- Action Button for "User not found" -->
              <button
                v-if="isLogin && authStore.error.includes('not found')"
                @click="isLogin = false; authStore.error = null"
                type="button"
                class="w-full px-4 py-3 bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-700 hover:to-indigo-700 dark:from-indigo-600 dark:to-purple-600 dark:hover:from-indigo-700 dark:hover:to-purple-700 text-white font-bold rounded-xl transition-all duration-300 flex items-center justify-center gap-2 shadow-lg hover:shadow-xl hover:scale-105"
              >
                <UserPlus class="w-5 h-5" />
                <span>Create New Account</span>
              </button>
            </div>
          </div>
        </Transition>

        <button
          type="submit"
          :disabled="authStore.isLoading || !isFormValid"
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
import { ref, reactive, watch, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { Terminal, LogIn, Loader2, UserPlus, AlertCircle, XCircle } from 'lucide-vue-next'
import ThemeToggle from '@/components/ThemeToggle.vue'

const router = useRouter()
const authStore = useAuthStore()

const isLogin = ref(true)
const formData = reactive({
  username: '',
  email: '',
  password: ''
})

// Form validation
const isFormValid = computed(() => {
  const minPasswordLength = isLogin.value ? 6 : 8

  if (!formData.username || !formData.password) {
    return false
  }

  if (formData.password.length < minPasswordLength) {
    return false
  }

  if (!isLogin.value && !formData.email) {
    return false
  }

  return true
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
