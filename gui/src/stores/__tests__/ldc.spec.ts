import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useLDCStore } from '../ldc'
import { mockLDCs } from '@/test/fixtures/stores'
import { createMockConnectRPCResponse, createMockConnectRPCError } from '@/test/utils'
import { mockDuckApiClient } from '@/test/setup'

describe('LDC Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    // Reset mock to return empty array by default
    mockDuckApiClient.getLDCs.mockResolvedValue(createMockConnectRPCResponse({ ldcs: [] }))
  })

  describe('initial state', () => {
    it('should have empty initial state', async () => {
      const store = useLDCStore()

      // Wait for automatic getLDCs() call to complete
      await new Promise(resolve => setTimeout(resolve, 0))

      expect(store.ldcs).toEqual([])
      expect(store.errorLDCs).toBe('')
      expect(store.nActiveLDCs).toBe(0)
    })
  })

  describe('computed properties', () => {
    it('should calculate number of active LDCs correctly', () => {
      const store = useLDCStore()

      store.ldcs = mockLDCs

      expect(store.nActiveLDCs).toBe(1) // Only ldc1 is enabled
    })

    it('should return 0 when no LDCs are enabled', () => {
      const store = useLDCStore()

      store.ldcs = mockLDCs.map(ldc => ({ ...ldc, enabled: false }))

      expect(store.nActiveLDCs).toBe(0)
    })

    it('should count all enabled LDCs', () => {
      const store = useLDCStore()

      store.ldcs = mockLDCs.map(ldc => ({ ...ldc, enabled: true }))

      expect(store.nActiveLDCs).toBe(2)
    })

    it('should handle LDCs with no equipment', () => {
      const store = useLDCStore()

      store.ldcs = mockLDCs

      expect(store.nActiveLDCs).toBe(1) // Only ldc1 is enabled, even though ldc2 has no equipment
    })
  })

  describe('getLDCs action', () => {
    it('should fetch LDCs successfully', async () => {
      const store = useLDCStore()

      mockDuckApiClient.getLDCs.mockResolvedValue(createMockConnectRPCResponse({ ldcs: mockLDCs }))

      store.getLDCs()

      // Wait for the promise to resolve since store.getLDCs() is not async
      await new Promise(resolve => setTimeout(resolve, 0))

      expect(mockDuckApiClient.getLDCs).toHaveBeenCalledWith({})
      expect(store.ldcs).toEqual(mockLDCs)
      expect(store.errorLDCs).toBe('')
    })

    it('should handle API errors correctly', async () => {
      const store = useLDCStore()
      const errorMessage = 'Network error'

      mockDuckApiClient.getLDCs.mockRejectedValue(createMockConnectRPCError(errorMessage))

      store.getLDCs()

      // Wait for the promise to resolve
      await new Promise(resolve => setTimeout(resolve, 0))

      expect(mockDuckApiClient.getLDCs).toHaveBeenCalledWith({})
      expect(store.ldcs).toEqual([])
      expect(store.errorLDCs).toBe(errorMessage)
    })

    it('should handle empty response', async () => {
      const store = useLDCStore()

      mockDuckApiClient.getLDCs.mockResolvedValue(createMockConnectRPCResponse({ ldcs: [] }))

      store.getLDCs()

      // Wait for the promise to resolve
      await new Promise(resolve => setTimeout(resolve, 0))

      expect(store.ldcs).toEqual([])
      expect(store.errorLDCs).toBe('')
      expect(store.nActiveLDCs).toBe(0)
    })
  })

  describe('equipment handling', () => {
    it('should preserve equipment data structure', async () => {
      const store = useLDCStore()

      mockDuckApiClient.getLDCs.mockResolvedValue(createMockConnectRPCResponse({ ldcs: mockLDCs }))
      store.getLDCs()
      await new Promise(resolve => setTimeout(resolve, 0))

      const ldc = store.ldcs[0]
      expect(ldc.equipments).toBeDefined()
      expect(Array.isArray(ldc.equipments)).toBe(true)

      if (ldc.equipments.length > 0) {
        const equipment = ldc.equipments[0]
        expect(equipment).toHaveProperty('id')
        expect(equipment).toHaveProperty('type')
        expect(equipment).toHaveProperty('deviceIp')
        expect(equipment).toHaveProperty('hostIp')
        expect(equipment).toHaveProperty('hostPort')
        expect(equipment).toHaveProperty('ldcId')
        expect(equipment).toHaveProperty('enabled')
      }
    })

    it('should handle LDCs with empty equipment arrays', async () => {
      const store = useLDCStore()

      mockDuckApiClient.getLDCs.mockResolvedValue(createMockConnectRPCResponse({ ldcs: mockLDCs }))
      store.getLDCs()
      await new Promise(resolve => setTimeout(resolve, 0))

      const ldc2 = store.ldcs.find(ldc => ldc.name === 'ldc2')
      expect(ldc2?.equipments).toEqual([])
    })
  })

  describe('store reset behavior', () => {
    it('should maintain state after multiple calls', async () => {
      const store = useLDCStore()

      // First successful call
      mockDuckApiClient.getLDCs.mockResolvedValueOnce(createMockConnectRPCResponse({ ldcs: mockLDCs }))
      store.getLDCs()
      await new Promise(resolve => setTimeout(resolve, 0))

      expect(store.ldcs).toEqual(mockLDCs)
      expect(store.nActiveLDCs).toBe(1)

      // Second call with error
      const errorMessage = 'Server error'
      mockDuckApiClient.getLDCs.mockRejectedValueOnce(createMockConnectRPCError(errorMessage))
      store.getLDCs()
      await new Promise(resolve => setTimeout(resolve, 0))

      // Store maintains previous data on error, only error message changes
      expect(store.ldcs).toEqual(mockLDCs) // Data remains
      expect(store.errorLDCs).toBe(errorMessage)
    })
  })

  describe('data integrity', () => {
    it('should preserve LDC data structure', async () => {
      const store = useLDCStore()

      mockDuckApiClient.getLDCs.mockResolvedValue(createMockConnectRPCResponse({ ldcs: mockLDCs }))
      store.getLDCs()
      await new Promise(resolve => setTimeout(resolve, 0))

      const ldc = store.ldcs[0]
      expect(ldc).toHaveProperty('id')
      expect(ldc).toHaveProperty('name')
      expect(ldc).toHaveProperty('hostname')
      expect(ldc).toHaveProperty('ip')
      expect(ldc).toHaveProperty('enabled')
      expect(ldc).toHaveProperty('equipments')
      expect(ldc).toHaveProperty('grpcPort')
      expect(ldc).toHaveProperty('prometheusPort')
    })

    it('should handle partial LDC data', async () => {
      const store = useLDCStore()
      const partialLDC = { id: 1, name: 'partial-ldc' }

      mockDuckApiClient.getLDCs.mockResolvedValue(createMockConnectRPCResponse({ ldcs: [partialLDC] }))
      store.getLDCs()
      await new Promise(resolve => setTimeout(resolve, 0))

      expect(store.ldcs[0]).toEqual(partialLDC)
    })
  })

  describe('console logging', () => {
    it('should log getLDCs call', async () => {
      const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {})

      const store = useLDCStore()
      mockDuckApiClient.getLDCs.mockResolvedValue(createMockConnectRPCResponse({ ldcs: mockLDCs }))

      store.getLDCs()
      await new Promise(resolve => setTimeout(resolve, 0))

      expect(consoleSpy).toHaveBeenCalledWith('get ldcs')
      consoleSpy.mockRestore()
    })

    it('should log successful response', async () => {
      const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {})

      const store = useLDCStore()
      const response = createMockConnectRPCResponse({ ldcs: mockLDCs })
      mockDuckApiClient.getLDCs.mockResolvedValue(response)

      store.getLDCs()
      await new Promise(resolve => setTimeout(resolve, 0))

      // Check that console was called with the response object (not just the data)
      expect(consoleSpy).toHaveBeenCalledWith(response)
      consoleSpy.mockRestore()
    })

    it('should log error response', async () => {
      const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {})
      const errorSpy = vi.spyOn(console, 'log').mockImplementation(() => {})

      const store = useLDCStore()
      const error = createMockConnectRPCError('Test error')
      mockDuckApiClient.getLDCs.mockRejectedValue(error)

      store.getLDCs()
      await new Promise(resolve => setTimeout(resolve, 0))

      expect(errorSpy).toHaveBeenCalledWith(error)
      consoleSpy.mockRestore()
      errorSpy.mockRestore()
    })
  })
})