import { beforeAll, afterAll, afterEach, beforeEach } from 'vitest'
import { server } from '../mocks/server'
import { createPinia, setActivePinia } from 'pinia'

// Start MSW server before all tests
beforeAll(() => {
  // Enable request interception
  server.listen({
    onUnhandledRequest: 'warn'
  })
})

// Reset any request handlers that are added as part of tests
afterEach(() => {
  server.resetHandlers()
})

// Close server after all tests
afterAll(() => {
  server.close()
})

// Setup fresh Pinia instance for each test
beforeEach(() => {
  const pinia = createPinia()
  setActivePinia(pinia)
})

// Export server for use in tests
export { server }