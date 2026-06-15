import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// Vite config.
// Proxy: all frontend requests to /api/* are forwarded to the analytics backend.
// This is needed to bypass CORS in development mode.
// If the analytics backend (@vIadimirsoIovev) uses a different port, change 8081 below.
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8081',
        changeOrigin: true,
        // frontend calls /api/v1/analytics/stats -> backend receives /v1/analytics/stats
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
    },
  },
})