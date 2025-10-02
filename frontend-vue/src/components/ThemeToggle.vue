<template>
  <button
    @click="toggleTheme"
    class="theme-toggle-button group relative flex items-center justify-center w-12 h-12 rounded-full bg-gradient-to-br transition-all duration-300 hover:scale-110 active:scale-95"
    :class="[
      isDark
        ? 'from-indigo-600 to-purple-600 hover:from-indigo-500 hover:to-purple-500 shadow-lg shadow-indigo-500/50'
        : 'from-amber-400 to-orange-500 hover:from-amber-300 hover:to-orange-400 shadow-lg shadow-orange-500/50'
    ]"
    :title="isDark ? 'Switch to Light Mode' : 'Switch to Dark Mode'"
  >
    <!-- Sun Icon (Light Mode) -->
    <Transition
      enter-active-class="transition-all duration-300"
      enter-from-class="opacity-0 rotate-180 scale-0"
      enter-to-class="opacity-100 rotate-0 scale-100"
      leave-active-class="transition-all duration-300"
      leave-from-class="opacity-100 rotate-0 scale-100"
      leave-to-class="opacity-0 -rotate-180 scale-0"
    >
      <Sun
        v-if="!isDark"
        class="absolute w-6 h-6 text-white animate-spin-slow"
      />
    </Transition>

    <!-- Moon Icon (Dark Mode) -->
    <Transition
      enter-active-class="transition-all duration-300"
      enter-from-class="opacity-0 -rotate-90 scale-0"
      enter-to-class="opacity-100 rotate-0 scale-100"
      leave-active-class="transition-all duration-300"
      leave-from-class="opacity-100 rotate-0 scale-100"
      leave-to-class="opacity-0 rotate-90 scale-0"
    >
      <Moon
        v-if="isDark"
        class="absolute w-6 h-6 text-white"
      />
    </Transition>

    <!-- Ripple Effect on Click -->
    <span
      v-if="isAnimating"
      class="absolute inset-0 rounded-full bg-white/30 animate-ping"
    ></span>

    <!-- Stars Background for Dark Mode -->
    <div v-if="isDark" class="absolute inset-0 rounded-full overflow-hidden pointer-events-none">
      <div
        v-for="i in 3"
        :key="i"
        class="absolute w-1 h-1 bg-white rounded-full animate-pulse"
        :style="{
          top: `${20 + i * 20}%`,
          left: `${15 + i * 25}%`,
          animationDelay: `${i * 0.3}s`
        }"
      ></div>
    </div>

    <!-- Rays for Light Mode -->
    <div v-if="!isDark" class="absolute inset-0 rounded-full overflow-hidden pointer-events-none">
      <div
        v-for="i in 8"
        :key="i"
        class="absolute w-0.5 h-2 bg-white/50 origin-center"
        :style="{
          top: '50%',
          left: '50%',
          transform: `translate(-50%, -50%) rotate(${i * 45}deg) translateY(-150%)`,
        }"
      ></div>
    </div>
  </button>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { Sun, Moon } from 'lucide-vue-next'
import { useThemeStore } from '@/stores/theme'

const themeStore = useThemeStore()
const isAnimating = ref(false)

const isDark = computed(() => themeStore.isDark())

const toggleTheme = () => {
  isAnimating.value = true
  themeStore.toggleTheme()

  setTimeout(() => {
    isAnimating.value = false
  }, 600)
}
</script>

<style scoped>
.theme-toggle-button {
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
}

.theme-toggle-button:active {
  box-shadow: 0 0 20px currentColor;
}

/* Custom animations */
@keyframes rotate-rays {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}
</style>
