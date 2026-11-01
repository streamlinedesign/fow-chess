import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

const api = `http://127.0.0.1:${process.env.PORT ?? 4100}`

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: Number(process.env.WEB_PORT ?? 4101),
    strictPort: true,
    // Transform the app up front so the first page load after `make dev` isn't slow.
    warmup: { clientFiles: ['./src/main.tsx'] },
    proxy: {
      '/api': api,
      '/ws': { target: api.replace('http', 'ws'), ws: true },
    },
  },
  build: { target: 'es2022' },
})
