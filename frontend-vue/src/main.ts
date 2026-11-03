import { createApp } from 'vue'
import { createPinia } from 'pinia'
import router from './router'
import App from './App.vue'
import { setUnauthorizedHandler } from './services/api'
import { useAuthStore } from './stores/auth'
import './assets/main.css'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)

// Any 401 (except the /me bootstrap, login and password change) ends the client session.
setUnauthorizedHandler(() => {
  useAuthStore(pinia).clear()
  const current = router.currentRoute.value
  if (current.meta.requiresAuth !== false) {
    router.replace({ name: 'login', query: { redirect: current.fullPath } })
  }
})

router.isReady().then(() => app.mount('#app'))
