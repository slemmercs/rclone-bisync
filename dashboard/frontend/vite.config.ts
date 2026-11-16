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

// https://vite.dev/config/
export default defineConfig({
  base: './',
  plugins: [vue(), renameCockpitToDistIndex()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    rollupOptions: {
      input: {
        main: fileURLToPath(new URL('./cockpit.html', import.meta.url))
      }
    }
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true
      }
    }
  }
})

