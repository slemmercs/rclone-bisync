import { defineConfig, Plugin } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'
import fs from 'node:fs'
import path from 'node:path'

function renameCockpitToDistIndex(): Plugin {
  return {
    name: 'rename-cockpit-to-dist-index',
    closeBundle() {
      const distDir = fileURLToPath(new URL('./dist', import.meta.url))
      const cockpitPath = path.join(distDir, 'cockpit.html')
      const indexPath = path.join(distDir, 'index.html')
      if (fs.existsSync(cockpitPath)) {
        fs.renameSync(cockpitPath, indexPath)
      }
    }
  }
}

function rewriteRootToCockpit(): Plugin {
  return {
    name: 'rewrite-root-to-cockpit',
    configureServer(server) {
      server.middlewares.use((req, _res, next) => {
        if (req.url === '/' || req.url === '/index.html') {
          req.url = '/cockpit.html'
        }
        next()
      })
    }
  }
}

const backendUrl = process.env.VITE_BACKEND_URL || 'http://localhost:8080'
const devPort = Number(process.env.VITE_PORT) || 5173

// https://vite.dev/config/
export default defineConfig({
  base: './',
  plugins: [vue(), renameCockpitToDistIndex(), rewriteRootToCockpit()],
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
      input: {
        main: fileURLToPath(new URL('./cockpit.html', import.meta.url))
      },
      output: {
        manualChunks: {
          'vendor-vue': ['vue', 'pinia'],
          'vendor-primevue': ['primevue'],
          'vendor-chart': ['chart.js']
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

