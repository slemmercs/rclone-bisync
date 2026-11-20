import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

const backendUrl = process.env.VITE_BACKEND_URL || 'http://localhost:8080'
const devPort = Number(process.env.VITE_PORT) || 5173

// https://vite.dev/config/
export default defineConfig({
  base: './',
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    chunkSizeWarningLimit: 700,
    rollupOptions: {
      output: {
        manualChunks(id: string) {
          if (id.includes('node_modules/vue') || id.includes('node_modules/pinia')) {
            return 'vendor-vue'
          }
          if (id.includes('node_modules/primevue') || id.includes('node_modules/@primevue')) {
            return 'vendor-primevue'
          }
          if (id.includes('node_modules/chart.js')) {
            return 'vendor-chart'
          }
        }
      }
    }
  },
  server: {
    port: devPort,
    host: true,
    proxy: {
      '/api': {
        target: backendUrl,
        changeOrigin: true,
        ws: true
      }
    }
  }
})


