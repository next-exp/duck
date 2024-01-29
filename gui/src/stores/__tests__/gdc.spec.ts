import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useGDCStore } from '../gdc'
import { mockGDCs } from '@/test/fixtures/stores'
import { createMockConnectRPCResponse, createMockConnectRPCError } from '@/test/utils'
import { mockDuckApiClient } from '@/test/setup'

describe('GDC Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    // Reset mock to return empty array by default
    mockDuckApiClient.getGDCs.mockResolvedValue(createMockConnectRPCResponse({ gdcs: [] }))
  })

  describe('initial state', () => {
    it('should have empty initial state', async () => {
      const store = useGDCStore()

      // Wait for automatic getGDCs() call to complete
      await new Promise(resolve => setTimeout(resolve, 0))

      expect(store.gdcs).toEqual([])
      expect(store.errorGDCs).toBe('')
      expect(store.nActiveGDCs).toBe(0)
    })
  })

  describe('computed properties', () => {
    it('should calculate number of active GDCs correctly', () => {
      const store = useGDCStore()

      store.gdcs = mockGDCs

      expect(store.nActiveGDCs).toBe(1) // Only gdc1 is enabled
    })

    it('should return 0 when no GDCs are enabled', () => {
      const store = useGDCStore()

      store.gdcs = mockGDCs.map(gdc => ({ ...gdc, enabled: false }))

      expect(store.nActiveGDCs).toBe(0)
    })

    it('should count all enabled GDCs', () => {
      const store = useGDCStore()

      store.gdcs = mockGDCs.map(gdc => ({ ...gdc, enabled: true }))

      expect(store.nActiveGDCs).toBe(2)
    })
  })

  describe('getGDCs action', () => {
    it('should fetch GDCs successfully', async () => {
      const store = useGDCStore()

      mockDuckApiClient.getGDCs.mockResolvedValue(createMockConnectRPCResponse({ gdcs: mockGDCs }))

      store.getGDCs()

      // Wait for the promise to resolve since store.getGDCs() is not async
      await new Promise(resolve => setTimeout(resolve, 0))

      expect(mockDuckApiClient.getGDCs).toHaveBeenCalledWith({})
      expect(store.gdcs).toEqual(mockGDCs)
      expect(store.errorGDCs).toBe('')
    })

    it('should handle API errors correctly', async () => {
      const store = useGDCStore()
      const errorMessage = 'Network error'

      mockDuckApiClient.getGDCs.mockRejectedValue(createMockConnectRPCError(errorMessage))

      store.getGDCs()

      // Wait for the promise to resolve
      await new Promise(resolve => setTimeout(resolve, 0))

      expect(mockDuckApiClient.getGDCs).toHaveBeenCalledWith({})
      expect(store.gdcs).toEqual([])
      expect(store.errorGDCs).toBe(errorMessage)
    })

    it('should handle empty response', async () => {
      const store = useGDCStore()

      mockDuckApiClient.getGDCs.mockResolvedValue(createMockConnectRPCResponse({ gdcs: [] }))

      store.getGDCs()

      // Wait for the promise to resolve
      await new Promise(resolve => setTimeout(resolve, 0))

      expect(store.gdcs).toEqual([])
      expect(store.errorGDCs).toBe('')
      expect(store.nActiveGDCs).toBe(0)
    })
  })

  describe('store reset behavior', () => {
    it('should maintain state after multiple calls', async () => {
      const store = useGDCStore()

      // First successful call
      mockDuckApiClient.getGDCs.mockResolvedValueOnce(createMockConnectRPCResponse({ gdcs: mockGDCs }))
      store.getGDCs()
      await new Promise(resolve => setTimeout(resolve, 0))

      expect(store.gdcs).toEqual(mockGDCs)
      expect(store.nActiveGDCs).toBe(1)

      // Second call with error
      const errorMessage = 'Server error'
      mockDuckApiClient.getGDCs.mockRejectedValueOnce(createMockConnectRPCError(errorMessage))
      store.getGDCs()
      await new Promise(resolve => setTimeout(resolve, 0))

      // Store maintains previous data on error, only error message changes
      expect(store.gdcs).toEqual(mockGDCs) // Data remains
      expect(store.errorGDCs).toBe(errorMessage)
    })
  })

  describe('data integrity', () => {
    it('should preserve GDC data structure', async () => {
      const store = useGDCStore()

      mockDuckApiClient.getGDCs.mockResolvedValue(createMockConnectRPCResponse({ gdcs: mockGDCs }))
      store.getGDCs()
      await new Promise(resolve => setTimeout(resolve, 0))

      const gdc = store.gdcs[0]
      expect(gdc).toHaveProperty('id')
      expect(gdc).toHaveProperty('name')
      expect(gdc).toHaveProperty('hostname')
      expect(gdc).toHaveProperty('ip')
      expect(gdc).toHaveProperty('port')
      expect(gdc).toHaveProperty('datapath')
      expect(gdc).toHaveProperty('enabled')
      expect(gdc).toHaveProperty('writeOutput')
      expect(gdc).toHaveProperty('decode')
      expect(gdc).toHaveProperty('grpcPort')
      expect(gdc).toHaveProperty('prometheusPort')
    })

    it('should handle partial GDC data', async () => {
      const store = useGDCStore()
      const partialGDC = { id: 1, name: 'partial-gdc' }

      mockDuckApiClient.getGDCs.mockResolvedValue(createMockConnectRPCResponse({ gdcs: [partialGDC] }))
      store.getGDCs()
      await new Promise(resolve => setTimeout(resolve, 0))

      expect(store.gdcs[0]).toEqual(partialGDC)
    })
  })
})