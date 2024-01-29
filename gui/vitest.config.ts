import { fileURLToPath } from 'node:url'
import { defineConfig, configDefaults } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath as fileURLToPath2, URL } from 'node:url'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath2(new URL('./src', import.meta.url))
    }
  },
  test: {
    environment: 'jsdom',
    exclude: [...configDefaults.exclude, 'e2e/*', 'e2e-integration/*', 'cypress/*'],
    root: fileURLToPath(new URL('./', import.meta.url)),
    setupFiles: ['./src/test/setup.ts'],
    globals: true,
    environmentOptions: {
      jsdom: {}
    },
    hookTimeout: 10000,
    teardownTimeout: 10000,
    coverage: {
      reporter: ['text', 'json', 'html'],
      exclude: [
        'node_modules/',
        'src/test/',
        '**/*.d.ts',
        '**/*.config.*',
        'src/main.ts',
        'src/App.vue',
        'cypress/',
        'e2e/'
      ],
      thresholds: {
        global: {
          branches: 70,
          functions: 75,
          lines: 75,
          statements: 75
        }
      }
    }
  }
})