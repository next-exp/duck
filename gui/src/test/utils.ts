import { mount, VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import type { ComponentMountingOptions } from '@vue/test-utils'
import { createRouter, createWebHistory } from 'vue-router'

export interface TestWrapper extends VueWrapper {
  router: any
}

export function createTestWrapper(
  component: any,
  options: ComponentMountingOptions<any> = {}
): TestWrapper {
  const pinia = createPinia()
  setActivePinia(pinia)

  const router = createRouter({
    history: createWebHistory(),
    routes: [
      { path: '/', component: { template: '<div>Home</div>' } },
      { path: '/detector', component: { template: '<div>Detector</div>' } }
    ]
  })

  const wrapper = mount(component, {
    global: {
      plugins: [pinia, router],
      stubs: {
        'font-awesome-icon': true,
        'router-link': true,
        'router-view': true,
        'AlertError': true,
        'AlertWarning': true,
        'AlertSuccess': true
      }
    },
    ...options
  }) as TestWrapper

  wrapper.router = router
  return wrapper
}

export function createMockResponse<T>(data: T, status = 200) {
  return {
    data,
    status,
    statusText: 'OK',
    headers: {},
    config: {}
  }
}

export function createMockError(message: string, status = 500) {
  const error = new Error(message) as any
  error.response = {
    status,
    statusText: 'Internal Server Error',
    data: { message }
  }
  error.message = message
  return error
}

export function createMockConnectRPCResponse<T>(data: T) {
  return data
}

export function createMockConnectRPCError(message: string) {
  const error = new Error(message) as any
  error.message = message
  return error
}

export function flushPromises() {
  return new Promise(resolve => setTimeout(resolve, 0))
}

export type MockedConnectRPCClient = {
  getDecoderConfiguration: ReturnType<typeof vi.fn>
  getLDCs: ReturnType<typeof vi.fn>
  getGDCs: ReturnType<typeof vi.fn>
  getEquipments: ReturnType<typeof vi.fn>
  getProcessStates: ReturnType<typeof vi.fn>
  getRunNumber: ReturnType<typeof vi.fn>
  setDecoderConfiguration: ReturnType<typeof vi.fn>
  setLDC: ReturnType<typeof vi.fn>
  setGDC: ReturnType<typeof vi.fn>
  setEquipment: ReturnType<typeof vi.fn>
  startRun: ReturnType<typeof vi.fn>
  stopRun: ReturnType<typeof vi.fn>
}

export async function getMockedConnectRPCClient(): Promise<MockedConnectRPCClient> {
  const client = await import('@/api/client')
  return client.default as MockedConnectRPCClient
}

// Legacy exports for backward compatibility during migration
export type MockedAxios = MockedConnectRPCClient
export async function getMockedAxios(): Promise<MockedAxios> {
  return getMockedConnectRPCClient()
}

/**
 * Reset a Pinia store to its initial state
 * This helps with test isolation by clearing store state between tests
 */
export function resetStore(store: any) {
  if (!store) return

  // Use Pinia's $reset if available (requires defineStore with setup: true)
  if (typeof store.$reset === 'function') {
    store.$reset()
    return
  }

  // Manual reset for stores without $reset
  // Reset reactive properties to empty/initial values
  const keys = Object.keys(store)

  for (const key of keys) {
    // Skip internal Pinia properties
    if (key.startsWith('$') || key.startsWith('_')) {
      continue
    }

    const value = store[key]

    if (Array.isArray(value)) {
      store[key] = []
    } else if (typeof value === 'object' && value !== null && !(value instanceof Date)) {
      // Reset objects to empty state
      if (value.constructor === Object) {
        store[key] = {}
      }
    }
    // Primitives and other types are left as-is since they can't be easily reset
  }
}

/**
 * Reset all store-related error states
 * Use this in afterEach to clear error messages between tests
 */
export function resetStoreErrors(stores: any[]) {
  for (const store of stores) {
    if (!store) continue

    // Clear common error property names
    const errorKeys = Object.keys(store).filter(key =>
      key.toLowerCase().includes('error') ||
      key.toLowerCase().includes('warning')
    )

    for (const key of errorKeys) {
      store[key] = ''
    }
  }
}