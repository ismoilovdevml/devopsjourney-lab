import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export type Theme = 'light' | 'dark'

export const useThemeStore = defineStore('theme', () => {
  // Get initial theme from localStorage or system preference
  const getInitialTheme = (): Theme => {
    const stored = localStorage.getItem('theme') as Theme | null
    if (stored) return stored

    // Check system preference
    if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
      return 'dark'
    }

    return 'light'
  }

  const theme = ref<Theme>(getInitialTheme())

  // Apply theme to document
  const applyTheme = (newTheme: Theme) => {
    if (newTheme === 'dark') {
      document.documentElement.classList.add('dark')
    } else {
      document.documentElement.classList.remove('dark')
    }
  }

  // Initialize theme on load
  applyTheme(theme.value)

  // Watch for theme changes
  watch(theme, (newTheme) => {
    localStorage.setItem('theme', newTheme)
    applyTheme(newTheme)
  })

  // Toggle theme
  const toggleTheme = () => {
    theme.value = theme.value === 'light' ? 'dark' : 'light'
  }

  // Set specific theme
  const setTheme = (newTheme: Theme) => {
    theme.value = newTheme
  }

  // Check if dark mode
  const isDark = () => theme.value === 'dark'

  return {
    theme,
    toggleTheme,
    setTheme,
    isDark
  }
})
