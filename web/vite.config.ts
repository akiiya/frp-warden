import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// 构建输出到 web/dist，后续 Phase 7 通过 Go embed 内嵌到二进制。
// base: './' 保持相对路径，使 SPA 在 Go 静态服务下正常工作。
export default defineConfig({
  plugins: [vue()],
  base: './',
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    // 开发模式下将 /api 代理到 Go 后端 admin server。
    proxy: {
      '/api': 'http://127.0.0.1:8080',
    },
  },
})
