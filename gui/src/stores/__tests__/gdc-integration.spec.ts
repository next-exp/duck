import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useGDCStore } from '@/stores/gdc'
import { mockGDCs } from '@/test/fixtures/stores'
import { createMockConnectRPCResponse, createMockConnectRPCError } from '@/test/utils'

describe('GDC Store Integration Tests with ConnectRPC', () => {
  let store: any
  let mockDuckApiClient: any

  beforeEach(async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    store = useGDCStore()
    
    // Get the mocked ConnectRPC client
    mockDuckApiClient = vi.mocked((await import('@/api/client')).duckApiClient)
    
    // Setup default successful responses
    mockDuckApiClient.getGDCs.mockResolvedValue(createMockConnectRPCResponse({ gdcs: mockGDCs }))
    
    // Clear all mocks
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  describe('API Integration with ConnectRPC', () => {
    it('should fetch GDCs successfully from mock API', async () => {
      mockDuckApiClient.getGDCs.mockResolvedValue(createMockConnectRPCResponse({ gdcs: mockGDCs }))

      await store.getGDCs()

      expect(store.gdcs).toEqual(mockGDCs)
      expect(store.errorGDCs).toBe('')
    })

    it('should handle API errors gracefully', async () => {
      // First populate with data
      mockDuckApiClient.getGDCs.mockResolvedValue(createMockConnectRPCResponse({ gdcs: mockGDCs }))
      await store.getGDCs()
      
      // Then simulate error
      mockDuckApiClient.getGDCs.mockRejectedValue(new Error('Internal Server Error'))

      await store.getGDCs()

      expect(store.gdcs).toEqual(mockGDCs) // Maintains previous data
      expect(store.errorGDCs).toContain('Internal Server Error')
    })

    it('should handle network errors', async () => {
      // First populate with data
      mockDuckApiClient.getGDCs.mockResolvedValue(createMockConnectRPCResponse({ gdcs: mockGDCs }))
      await store.getGDCs()
      
      // Then simulate network error
      mockDuckApiClient.getGDCs.mockRejectedValue(new Error('Network connection failed'))

      await store.getGDCs()

      expect(store.errorGDCs).toBeTruthy() // Error from ConnectRPC
      expect(store.gdcs).toEqual(mockGDCs) // Maintains previous data
    })

    it('should handle 404 errors', async () => {
      mockDuckApiClient.getGDCs.mockRejectedValue(new Error('Request failed with status code 404'))

      await store.getGDCs()

      expect(store.errorGDCs).toContain('Request failed with status code 404')
    })
  })

  describe('Complex scenarios', () => {
    it('should handle slow network responses', async () => {
      // Mock slow response
      mockDuckApiClient.getGDCs.mockImplementation(async () => {
        await new Promise(resolve => setTimeout(resolve, 1000))
        return createMockConnectRPCResponse({ gdcs: mockGDCs })
      })

      const startTime = Date.now()
      await store.getGDCs()
      const endTime = Date.now()

      expect(endTime - startTime).toBeGreaterThanOrEqual(1000)
      expect(store.gdcs).toEqual(mockGDCs)
    })

    it('should handle concurrent API calls', async () => {
      mockDuckApiClient.getGDCs.mockResolvedValue(createMockConnectRPCResponse({ gdcs: mockGDCs }))
      
      const promises = [
        store.getGDCs(),
        store.getGDCs(),
        store.getGDCs()
      ]

      await Promise.all(promises)

      expect(store.gdcs).toEqual(mockGDCs)
    })

    it('should recover from intermittent failures', async () => {
      // First call fails
      mockDuckApiClient.getGDCs.mockRejectedValue(new Error('Network connection failed'))
      await store.getGDCs()
      expect(store.errorGDCs).toBeTruthy() // Error from ConnectRPC

      // Second call succeeds
      mockDuckApiClient.getGDCs.mockResolvedValue(createMockConnectRPCResponse({ gdcs: mockGDCs }))
      await store.getGDCs()
      expect(store.gdcs).toEqual(mockGDCs)
      expect(store.errorGDCs).toBe('')
    })
  })

  describe('State management with API responses', () => {
    it('should clear errors on successful API call', async () => {
      // First, set an error
      mockDuckApiClient.getGDCs.mockRejectedValue(new Error('Request failed with status code 500'))
      await store.getGDCs()
      expect(store.errorGDCs).toContain('Request failed with status code 500')

      // Then, successful call should clear error
      mockDuckApiClient.getGDCs.mockResolvedValue(createMockConnectRPCResponse({ gdcs: mockGDCs }))
      await store.getGDCs()
      expect(store.errorGDCs).toBe('')
    })

    it('should handle partial API responses', async () => {
      const partialGDC = {
        id: 1,
        name: 'partial-gdc',
        hostname: 'partial-host'
        // Missing other fields
      }

      mockDuckApiClient.getGDCs.mockResolvedValue(createMockConnectRPCResponse({ gdcs: [partialGDC] }))

      await store.getGDCs()

      expect(store.gdcs).toEqual([partialGDC])
    })

    it('should compute nActiveGDCs correctly', async () => {
      mockDuckApiClient.getGDCs.mockResolvedValue(createMockConnectRPCResponse({ gdcs: mockGDCs }))
      await store.getGDCs()

      const enabledGDCs = mockGDCs.filter(gdc => gdc.enabled)
      expect(store.nActiveGDCs).toBe(enabledGDCs.length)
    })
  })

  describe('Real-world simulation scenarios', () => {
    it('should simulate typical user workflow', async () => {
      // 1. Load initial GDCs
      mockDuckApiClient.getGDCs.mockResolvedValue(createMockConnectRPCResponse({ gdcs: mockGDCs }))
      await store.getGDCs()
      expect(store.gdcs).toEqual(mockGDCs)

      // 2. Refresh data
      await store.getGDCs()
      expect(store.gdcs).toEqual(mockGDCs)
    })

    it('should handle server maintenance scenario', async () => {
      // First populate with data
      mockDuckApiClient.getGDCs.mockResolvedValue(createMockConnectRPCResponse({ gdcs: mockGDCs }))
      await store.getGDCs()
      
      // Simulate server maintenance (503 error)
      mockDuckApiClient.getGDCs.mockRejectedValue(new Error('Request failed with status code 503'))

      await store.getGDCs()
      expect(store.errorGDCs).toContain('Request failed with status code 503')
      expect(store.gdcs).toEqual(mockGDCs) // Maintains previous data
    })

    it('should handle authentication errors', async () => {
      mockDuckApiClient.getGDCs.mockRejectedValue(new Error('Request failed with status code 401'))

      await store.getGDCs()
      expect(store.errorGDCs).toContain('Request failed with status code 401')
    })

    it('should handle rate limiting scenario', async () => {
      // Simulate rate limiting (403 error)
      mockDuckApiClient.getGDCs.mockRejectedValue(new Error('Request failed with status code 403'))

      await store.getGDCs()
      expect(store.errorGDCs).toContain('Request failed with status code 403')
    })
  })

  describe('Performance optimization', () => {
    it('should handle large GDC datasets efficiently', async () => {
      const largeDataset = Array.from({ length: 100 }, (_, i) => ({
        id: i + 1,
        name: `gdc-${i + 1}`,
        hostname: `gdc-${i + 1}`,
        ip: `192.168.1.${(i % 254) + 1}`,
        port: 6005 + i,
        datapath: `/data/gdc-${i + 1}`,
        grpcPort: 50051 + i,
        prometheusPort: 9090 + i,
        enabled: i % 2 === 0,
        writeOutput: i % 3 === 0,
        decode: i % 4 === 0
      }))

      mockDuckApiClient.getGDCs.mockResolvedValue(createMockConnectRPCResponse({ gdcs: largeDataset }))

      const startTime = Date.now()
      await store.getGDCs()
      const endTime = Date.now()

      expect(endTime - startTime).toBeLessThan(1000)
      expect(store.gdcs).toHaveLength(100)
      expect(store.gdcs[0]).toEqual(largeDataset[0])
    })

    it('should optimize repeated requests', async () => {
      mockDuckApiClient.getGDCs.mockResolvedValue(createMockConnectRPCResponse({ gdcs: mockGDCs }))
      
      const promises = Array.from({ length: 10 }, () => store.getGDCs())

      await Promise.all(promises)

      expect(store.gdcs).toEqual(mockGDCs)
    })
  })
})