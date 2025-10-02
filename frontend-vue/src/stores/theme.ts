import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export type Theme = 'light' | 'dark' | 'system'

export const useThemeStore = defineStore('theme', () => {
  // Get system preference
  const getSystemTheme = (): 'light' | 'dark' => {
    if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
      return 'dark'
    }
    return 'light'
  }

  // Get initial theme from localStorage or default to system
  const getInitialTheme = (): Theme => {
    const stored = localStorage.getItem('theme') as Theme | null
    return stored || 'system' // Default to system theme
  }

  const theme = ref<Theme>(getInitialTheme())
  const systemTheme = ref<'light' | 'dark'>(getSystemTheme())

  // Get actual theme to apply (resolve 'system' to 'light' or 'dark')
  const getActualTheme = (): 'light' | 'dark' => {
    if (theme.value === 'system') {
      return systemTheme.value
    }
    return theme.value
  }

  // Apply theme to document
  const applyTheme = () => {
    const actualTheme = getActualTheme()
    if (actualTheme === 'dark') {
      document.documentElement.classList.add('dark')
    } else {
      document.documentElement.classList.remove('dark')
    }
  }

  // Initialize theme on load
  applyTheme()

  // Listen to system theme changes
  if (window.matchMedia) {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
    mediaQuery.addEventListener('change', (e) => {
      systemTheme.value = e.matches ? 'dark' : 'light'
      applyTheme()
    })
  }

  // Watch for theme changes
  watch(theme, (newTheme) => {
    localStorage.setItem('theme', newTheme)
    applyTheme()
  })

  // Toggle theme (cycles through: system -> light -> dark -> system)
  const toggleTheme = () => {
    if (theme.value === 'system') {
      theme.value = 'light'
    } else if (theme.value === 'light') {
      theme.value = 'dark'
    } else {
      theme.value = 'system'
    }
  }

  // Set specific theme
  const setTheme = (newTheme: Theme) => {
    theme.value = newTheme
  }

  // Check if current applied theme is dark
  const isDark = () => getActualTheme() === 'dark'

  return {
    theme,
    systemTheme,
    toggleTheme,
    setTheme,
    isDark,
    getActualTheme
  }
})
