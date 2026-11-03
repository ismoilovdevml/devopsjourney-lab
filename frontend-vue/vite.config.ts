import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

// Dev only: the Go backend. In production nginx proxies /api, /ws and /healthz.
const backend = 'http://localhost:8080'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: backend,
        changeOrigin: false
      },
      '/healthz': {
        target: backend,
        changeOrigin: false
      },
      '/readyz': {
        target: backend,
        changeOrigin: false
      },
      '/ws': {
        target: backend,
        ws: true,
        changeOrigin: false
      }
    }
  },
  build: {
    sourcemap: false,
    rollupOptions: {
      output: {
        manualChunks: {
          xterm: ['xterm', 'xterm-addon-fit'],
          vendor: ['vue', 'vue-router', 'pinia', 'axios']
        }
      }
    }
  }
})
