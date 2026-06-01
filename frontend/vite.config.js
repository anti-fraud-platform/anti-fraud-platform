import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Конфиг Vite.
// Прокси: все запросы фронта на /api/* будут перенаправлены на бэкенд аналитики.
// Это нужно, чтобы обойти CORS в режиме разработки.
// Если у бэкенда аналитики (@vIadimirsoIovev) другой порт — поменяй 8081 ниже.
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8081',
        changeOrigin: true,
        // фронт зовёт /api/v1/analytics/stats -> бэк получает /v1/analytics/stats
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
    },
  },
})