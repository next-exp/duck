import { vi } from 'vitest'
import { config } from '@vue/test-utils'

// Global test setup for Vue Test Utils
config.global.stubs = {
  'font-awesome-icon': true,
  'AlertError': true,
  'AlertSuccess': true,
  'AlertWarning': true
}

// Mock ConnectRPC client for all tests
// Returns default empty values - individual tests can override using mockDuckApiClient
// For tests needing specific data, use fixtures from '@/test/fixtures/stores'
const mockDuckApiClient = {
  getDecoderConfiguration: vi.fn(() => Promise.resolve({ configuration: {} })),
  getLDCs: vi.fn(() => Promise.resolve({ ldcs: [] })),
  getGDCs: vi.fn(() => Promise.resolve({ gdcs: [] })),
  getEquipments: vi.fn(() => Promise.resolve({ equipments: [] })),
  getProcessStates: vi.fn(() => Promise.resolve({ states: {} })),
  getRunNumber: vi.fn(() => Promise.resolve({ runNumber: 0 })),
  updateDecoderConfiguration: vi.fn(() => Promise.resolve({})),
  createGDC: vi.fn(() => Promise.resolve({})),
  updateGDC: vi.fn(() => Promise.resolve({})),
  deleteGDC: vi.fn(() => Promise.resolve({})),
  createLDC: vi.fn(() => Promise.resolve({})),
  updateLDC: vi.fn(() => Promise.resolve({})),
  deleteLDC: vi.fn(() => Promise.resolve({})),
  createEquipment: vi.fn(() => Promise.resolve({})),
  updateEquipment: vi.fn(() => Promise.resolve({})),
  deleteEquipment: vi.fn(() => Promise.resolve({})),
  setDecoderConfiguration: vi.fn(() => Promise.resolve({})),
  setLDC: vi.fn(() => Promise.resolve({})),
  setGDC: vi.fn(() => Promise.resolve({})),
  setEquipment: vi.fn(() => Promise.resolve({})),
  startRun: vi.fn(() => Promise.resolve({})),
  stopRun: vi.fn(() => Promise.resolve({})),
  forceStopRun: vi.fn(() => Promise.resolve({})),
  restartServices: vi.fn(() => Promise.resolve({})),
  checkDisabled: vi.fn(() => Promise.resolve({ gdcs: [], ldcs: [], equipments: [], writing: [] })),
  // Add other methods as needed based on the actual API
}

vi.mock('@/api/client', () => ({
  default: mockDuckApiClient,
  duckApiClient: mockDuckApiClient
}))

export { mockDuckApiClient }

/**
 * Reset all mock API calls to their default state
 * Use this in afterEach hooks to prevent test interference
 */
export function resetMockAPI() {
  Object.values(mockDuckApiClient).forEach((mockFn: any) => {
    if (typeof mockFn === 'function' && typeof mockFn.mockReset === 'function') {
      // Reset call counts and clear mock implementations
      mockFn.mockReset()
    }
  })
}

/**
 * Clear all mocks and restore original implementations
 * Use this in afterEach hooks for complete cleanup
 */
export function clearAllMocks() {
  vi.clearAllMocks()
  vi.restoreAllMocks()
}

// Mock Centrifuge WebSocket for all tests
vi.mock('centrifuge', () => ({
  Centrifuge: vi.fn().mockImplementation(() => ({
    on: vi.fn(),
    newSubscription: vi.fn().mockReturnValue({
      subscribe: vi.fn(),
      on: vi.fn(),
      unsubscribe: vi.fn()
    }),
    connect: vi.fn(),
    disconnect: vi.fn()
  }))
}))

// Mock environment variables
vi.mock('@/env', () => ({
  VITE_API_SERVER: 'http://localhost:1323',
  VITE_WS_SERVER: 'ws://localhost:1323/daq'
}))

// Mock fetch for WebSocket token
global.fetch = vi.fn(() =>
  Promise.resolve({
    ok: true,
    status: 200,
    json: () => Promise.resolve({ token: 'mock-token' })
  })
)

// Mock console.log/debug to reduce noise in tests, but keep error/warn for debugging
global.console.log = vi.fn()
global.console.debug = vi.fn()
// Keep error and warn for debugging - don't silence them!
// This helps catch issues during test development

// Mock HTMLDialogElement.showModal() and close() for modal tests
// jsdom doesn't fully support dialog element methods
if (typeof HTMLDialogElement !== 'undefined') {
  HTMLDialogElement.prototype.showModal = vi.fn(function(this: HTMLDialogElement) {
    this.open = true
  })
  HTMLDialogElement.prototype.close = vi.fn(function(this: HTMLDialogElement) {
    this.open = false
  })
}

// ============================================================
// GLOBAL TEST CLEANUP FOR BETTER ISOLATION
// ============================================================

// Import afterEach from vitest for global cleanup
import { afterEach } from 'vitest'

// Global afterEach hook runs after every test
// This ensures proper cleanup and prevents test interference
afterEach(() => {
  // Clear all mock call counts and restore original implementations
  vi.clearAllMocks()

  // Clear any pending timers
  vi.clearAllTimers()

  // Reset the mock API to prevent call count bleeding between tests
  resetMockAPI()

  // Clean up DOM to prevent element leakage
  document.body.innerHTML = ''
  document.head.innerHTML = ''
})