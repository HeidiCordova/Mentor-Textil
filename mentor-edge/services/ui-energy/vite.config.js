import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    host: '0.0.0.0',
    port: 8087,
    proxy: {
      '/api': 'http://localhost:8086',
      '/health': 'http://localhost:8086',
      '/write-api': 'http://localhost:8088'
    }
  }
})
