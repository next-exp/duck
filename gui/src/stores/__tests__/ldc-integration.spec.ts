import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useLDCStore } from '@/stores/ldc'
import { useGDCStore } from '@/stores/gdc'
import { mockLDCs, mockGDCs } from '@/test/fixtures/stores'
import { createMockConnectRPCResponse, createMockConnectRPCError } from '@/test/utils'

describe('LDC Store Integration Tests with ConnectRPC', () => {
  let store: any
  let gdcStore: any
  let mockDuckApiClient: any

  beforeEach(async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    store = useLDCStore()
    gdcStore = useGDCStore()
    
    // Get the mocked ConnectRPC client
    mockDuckApiClient = vi.mocked((await import('@/api/client')).duckApiClient)
    
    // Setup default successful responses
    mockDuckApiClient.getLDCs.mockResolvedValue(createMockConnectRPCResponse({ ldcs: mockLDCs }))
    mockDuckApiClient.getGDCs.mockResolvedValue(createMockConnectRPCResponse({ gdcs: mockGDCs }))
    
    // Clear all mocks
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  describe('API Integration with ConnectRPC', () => {
    it('should fetch LDCs successfully from mock API', async () => {
      mockDuckApiClient.getLDCs.mockResolvedValue(createMockConnectRPCResponse({ ldcs: mockLDCs }))

      await store.getLDCs()

      expect(store.ldcs).toEqual(mockLDCs)
      expect(store.errorLDCs).toBe('')
    })

    it('should handle API errors gracefully', async () => {
      // First populate with data
      mockDuckApiClient.getLDCs.mockResolvedValue(createMockConnectRPCResponse({ ldcs: mockLDCs }))
      await store.getLDCs()
      
      // Then simulate error
      mockDuckApiClient.getLDCs.mockRejectedValue(new Error('Internal Server Error'))

      await store.getLDCs()

      expect(store.ldcs).toEqual(mockLDCs) // Maintains previous data
      expect(store.errorLDCs).toContain('Internal Server Error')
    })

    it('should handle network errors', async () => {
      // First populate with data
      mockDuckApiClient.getLDCs.mockResolvedValue(createMockConnectRPCResponse({ ldcs: mockLDCs }))
      await store.getLDCs()
      
      // Then simulate network error
      mockDuckApiClient.getLDCs.mockRejectedValue(new Error('Network connection failed'))

      await store.getLDCs()

      expect(store.errorLDCs).toBeTruthy() // Error from ConnectRPC
      expect(store.ldcs).toEqual(mockLDCs) // Maintains previous data
    })

    it('should handle 404 errors', async () => {
      mockDuckApiClient.getLDCs.mockRejectedValue(new Error('Request failed with status code 404'))

      await store.getLDCs()

      expect(store.errorLDCs).toContain('Request failed with status code 404')
    })
  })

  describe('Cross-store interactions', () => {
    it('should work alongside GDC store', async () => {
      // Mock GDC data
      mockDuckApiClient.getGDCs.mockResolvedValue(createMockConnectRPCResponse({ gdcs: mockGDCs }))
      await gdcStore.getGDCs()

      // Mock LDC data
      mockDuckApiClient.getLDCs.mockResolvedValue(createMockConnectRPCResponse({ ldcs: mockLDCs }))
      await store.getLDCs()

      expect(gdcStore.gdcs).toEqual(mockGDCs)
      expect(store.ldcs).toEqual(mockLDCs)

      // Verify stores are independent
      gdcStore.gdcs = []
      expect(store.ldcs).toEqual(mockLDCs) // LDC store unaffected
    })

    it('should handle concurrent operations across stores', async () => {
      // Mock both GDC and LDC data
      mockDuckApiClient.getGDCs.mockResolvedValue(createMockConnectRPCResponse({ gdcs: mockGDCs }))
      mockDuckApiClient.getLDCs.mockResolvedValue(createMockConnectRPCResponse({ ldcs: mockLDCs }))
      
      const gdcPromise = gdcStore.getGDCs()
      const ldcPromise = store.getLDCs()

      await Promise.all([gdcPromise, ldcPromise])

      expect(gdcStore.gdcs).toEqual(mockGDCs)
      expect(store.ldcs).toEqual(mockLDCs)
    })
  })

  describe('Complex scenarios', () => {
    it('should handle slow network responses', async () => {
      // Mock slow response
      mockDuckApiClient.getLDCs.mockImplementation(async () => {
        await new Promise(resolve => setTimeout(resolve, 100))
        return createMockConnectRPCResponse({ ldcs: mockLDCs })
      })

      const startTime = Date.now()
      await store.getLDCs()
      const endTime = Date.now()

      expect(endTime - startTime).toBeGreaterThanOrEqual(90)
      expect(store.ldcs).toEqual(mockLDCs)
    })

    it('should recover from intermittent failures', async () => {
      // First call fails
      mockDuckApiClient.getLDCs.mockRejectedValue(new Error('Network connection failed'))
      await store.getLDCs()
      expect(store.errorLDCs).toBeTruthy() // Error from ConnectRPC

      // Second call succeeds
      mockDuckApiClient.getLDCs.mockResolvedValue(createMockConnectRPCResponse({ ldcs: mockLDCs }))
      await store.getLDCs()
      expect(store.ldcs).toEqual(mockLDCs)
      expect(store.errorLDCs).toBe('')
    })

    it('should handle concurrent API calls', async () => {
      mockDuckApiClient.getLDCs.mockResolvedValue(createMockConnectRPCResponse({ ldcs: mockLDCs }))
      
      const promises = [
        store.getLDCs(),
        store.getLDCs(),
        store.getLDCs()
      ]

      await Promise.all(promises)

      expect(store.ldcs).toEqual(mockLDCs)
    })
  })

  describe('State management with API responses', () => {
    it('should clear errors on successful API call', async () => {
      // First, set an error
      mockDuckApiClient.getLDCs.mockRejectedValue(new Error('Request failed with status code 500'))
      await store.getLDCs()
      expect(store.errorLDCs).toContain('Request failed with status code 500')

      // Then, successful call should clear error
      mockDuckApiClient.getLDCs.mockResolvedValue(createMockConnectRPCResponse({ ldcs: mockLDCs }))
      await store.getLDCs()
      expect(store.errorLDCs).toBe('')
    })

    it('should handle partial API responses', async () => {
      const partialLDC = {
        id: 1,
        name: 'partial-ldc',
        hostname: 'partial-host'
        // Missing other fields
      }

      mockDuckApiClient.getLDCs.mockResolvedValue(createMockConnectRPCResponse({ ldcs: [partialLDC] }))

      await store.getLDCs()

      expect(store.ldcs).toEqual([partialLDC])
    })

    it('should compute nActiveLDCs correctly', async () => {
      mockDuckApiClient.getLDCs.mockResolvedValue(createMockConnectRPCResponse({ ldcs: mockLDCs }))
      await store.getLDCs()

      const enabledLDCs = mockLDCs.filter(ldc => ldc.enabled)
      expect(store.nActiveLDCs).toBe(enabledLDCs.length)
    })
  })

  describe('Real-world simulation scenarios', () => {
    it('should simulate typical user workflow', async () => {
      // 1. Load initial LDCs
      mockDuckApiClient.getLDCs.mockResolvedValue(createMockConnectRPCResponse({ ldcs: mockLDCs }))
      await store.getLDCs()
      expect(store.ldcs).toEqual(mockLDCs)

      // 2. Refresh data
      await store.getLDCs()
      expect(store.ldcs).toEqual(mockLDCs)
    })

    it('should handle server maintenance scenario', async () => {
      // First populate with data
      mockDuckApiClient.getLDCs.mockResolvedValue(createMockConnectRPCResponse({ ldcs: mockLDCs }))
      await store.getLDCs()
      
      // Simulate server maintenance (503 error)
      mockDuckApiClient.getLDCs.mockRejectedValue(new Error('Request failed with status code 503'))

      await store.getLDCs()
      expect(store.errorLDCs).toContain('Request failed with status code 503')
      expect(store.ldcs).toEqual(mockLDCs) // Maintains previous data
    })

    it('should handle authentication errors', async () => {
      mockDuckApiClient.getLDCs.mockRejectedValue(new Error('Request failed with status code 401'))

      await store.getLDCs()
      expect(store.errorLDCs).toContain('Request failed with status code 401')
    })

    it('should handle rate limiting scenario', async () => {
      // Simulate rate limiting (403 error)
      mockDuckApiClient.getLDCs.mockRejectedValue(new Error('Request failed with status code 403'))

      await store.getLDCs()
      expect(store.errorLDCs).toContain('Request failed with status code 403')
    })
  })

  describe('Performance optimization', () => {
    it('should handle large LDC datasets efficiently', async () => {
      const largeDataset = Array.from({ length: 50 }, (_, i) => ({
        id: i + 1,
        name: `ldc-${i + 1}`,
        hostname: `ldc-${i + 1}`,
        ip: `192.168.1.${(i % 254) + 1}`,
        grpcPort: 50053 + i,
        prometheusPort: 9092 + i,
        enabled: i % 2 === 0,
        equipments: []
      }))

      mockDuckApiClient.getLDCs.mockResolvedValue(createMockConnectRPCResponse({ ldcs: largeDataset }))

      const startTime = Date.now()
      await store.getLDCs()
      const endTime = Date.now()

      expect(endTime - startTime).toBeLessThan(1000)
      expect(store.ldcs).toHaveLength(50)
      expect(store.ldcs[0]).toEqual(largeDataset[0])
    })

    it('should optimize repeated requests', async () => {
      mockDuckApiClient.getLDCs.mockResolvedValue(createMockConnectRPCResponse({ ldcs: mockLDCs }))
      
      const promises = Array.from({ length: 10 }, () => store.getLDCs())

      await Promise.all(promises)

      expect(store.ldcs).toEqual(mockLDCs)
    })
  })
})