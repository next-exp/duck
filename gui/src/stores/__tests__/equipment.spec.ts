import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useEquipmentStore, type EquipmentType } from '../equipment'
import { mockEquipments } from '@/test/fixtures/stores'
import { createMockConnectRPCResponse, createMockConnectRPCError } from '@/test/utils'
import { mockDuckApiClient } from '@/test/setup'

describe('Equipment Store Tests', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  describe('Initial State Management', () => {
    it('should have empty initial state', () => {
      const store = useEquipmentStore()

      expect(store.equipments).toEqual([])
      expect(store.errorEquipments).toBe("")
    })

    it('should have correct reactive properties', () => {
      const store = useEquipmentStore()

      expect(store.equipments).toBeDefined()
      expect(store.errorEquipments).toBeDefined()
      expect(typeof store.getEquipments).toBe('function')
    })
  })

  describe('API Integration - getEquipments', () => {
    it('should fetch equipments successfully', async () => {
      mockDuckApiClient.getEquipments.mockResolvedValue(createMockConnectRPCResponse({ equipments: mockEquipments }))

      const store = useEquipmentStore()
      await store.getEquipments()

      expect(mockDuckApiClient.getEquipments).toHaveBeenCalledWith({})
      expect(store.equipments).toEqual(mockEquipments)
      expect(store.errorEquipments).toBe("")
    })

    it('should handle API errors correctly', async () => {
      const errorMessage = 'Network error'
      mockDuckApiClient.getEquipments.mockRejectedValue(createMockConnectRPCError(errorMessage))

      const store = useEquipmentStore()
      await store.getEquipments()

      expect(mockDuckApiClient.getEquipments).toHaveBeenCalledWith({})
      expect(store.equipments).toEqual([])
      expect(store.errorEquipments).toBe(errorMessage)
    })

    it('should handle empty response', async () => {
      mockDuckApiClient.getEquipments.mockResolvedValue(createMockConnectRPCResponse({ equipments: [] }))

      const store = useEquipmentStore()
      await store.getEquipments()

      expect(store.equipments).toEqual([])
      expect(store.errorEquipments).toBe("")
    })
  })

  describe('Data Validation and Type Safety', () => {
    it('should accept valid equipment data', async () => {
      const validEquipment: EquipmentType = {
        id: 1,
        type: 22,
        deviceIp: '192.168.1.100',
        hostIp: '192.168.1.20',
        hostPort: 6006,
        ldcId: 1,
        enabled: true
      }

      mockDuckApiClient.getEquipments.mockResolvedValue(createMockConnectRPCResponse({ equipments: [validEquipment] }))

      const store = useEquipmentStore()
      await store.getEquipments()

      expect(store.equipments[0]).toEqual(validEquipment)
    })

    it('should handle equipment with different types', async () => {
      const equipmentWithDifferentTypes = [
        { ...mockEquipments[0], type: 22 },
        { ...mockEquipments[1], type: 23 },
        { id: 3, type: 24, deviceIp: '192.168.1.102', hostIp: '192.168.1.21', hostPort: 6008, ldcId: 2, enabled: true }
      ]

      mockDuckApiClient.getEquipments.mockResolvedValue(createMockConnectRPCResponse({ equipments: equipmentWithDifferentTypes }))

      const store = useEquipmentStore()
      await store.getEquipments()

      expect(store.equipments).toHaveLength(3)
      expect(store.equipments.map(eq => eq.type)).toEqual([22, 23, 24])
    })

    it('should handle mixed enabled/disabled equipment states', async () => {
      const mixedEquipment = [
        { ...mockEquipments[0], enabled: true },
        { ...mockEquipments[1], enabled: false },
        { id: 3, type: 24, deviceIp: '192.168.1.102', hostIp: '192.168.1.22', hostPort: 6009, ldcId: 2, enabled: true },
        { id: 4, type: 25, deviceIp: '192.168.1.103', hostIp: '192.168.1.23', hostPort: 6010, ldcId: 3, enabled: false }
      ]

      mockDuckApiClient.getEquipments.mockResolvedValue(createMockConnectRPCResponse({ equipments:mixedEquipment }))

      const store = useEquipmentStore()
      await store.getEquipments()

      const enabledCount = store.equipments.filter(eq => eq.enabled).length
      const disabledCount = store.equipments.filter(eq => !eq.enabled).length

      expect(enabledCount).toBe(2)
      expect(disabledCount).toBe(2)
    })
  })

  describe('State Management and Reactivity', () => {
    it('should update equipments reactively', async () => {
      const initialData = [mockEquipments[0]]
      const updatedData = mockEquipments

      // Initial load
      mockDuckApiClient.getEquipments.mockResolvedValue(createMockConnectRPCResponse({ equipments:initialData }))
      const store = useEquipmentStore()
      await store.getEquipments()

      expect(store.equipments).toEqual(initialData)

      // Update with new data
      mockDuckApiClient.getEquipments.mockResolvedValue(createMockConnectRPCResponse({ equipments:updatedData }))
      await store.getEquipments()

      expect(store.equipments).toEqual(updatedData)
    })

    it('should clear error state on successful fetch', async () => {
      // Set initial error state
      mockDuckApiClient.getEquipments.mockRejectedValue(createMockConnectRPCError('Initial error'))
      const store = useEquipmentStore()
      await store.getEquipments()

      expect(store.errorEquipments).toBe('Initial error')

      // Successful fetch should clear error
      mockDuckApiClient.getEquipments.mockResolvedValue(createMockConnectRPCResponse({ equipments: mockEquipments }))
      await store.getEquipments()

      expect(store.errorEquipments).toBe("")
    })

    it('should handle concurrent getEquipments calls', async () => {
      mockDuckApiClient.getEquipments.mockResolvedValue(createMockConnectRPCResponse({ equipments: mockEquipments }))

      const store = useEquipmentStore()

      // Make multiple concurrent calls
      await Promise.all([
        store.getEquipments(),
        store.getEquipments(),
        store.getEquipments()
      ])

      // Note: mockDuckApiClient.getEquipments is called 4 times:
      // 1 time during store initialization + 3 times (concurrent calls)
      expect(mockDuckApiClient.getEquipments).toHaveBeenCalledTimes(4)
      expect(store.equipments).toEqual(mockEquipments)
    })
  })

  describe('Business Logic and Equipment Processing', () => {
    it('should handle equipment associated with different LDCs', async () => {
      const equipmentWithDifferentLDCs = [
        { ...mockEquipments[0], ldcId: 1 },
        { ...mockEquipments[1], ldcId: 2 },
        { id: 3, type: 24, deviceIp: '192.168.1.102', hostIp: '192.168.1.22', hostPort: 6009, ldcId: 3, enabled: true }
      ]

      mockDuckApiClient.getEquipments.mockResolvedValue(createMockConnectRPCResponse({ equipments:equipmentWithDifferentLDCs }))

      const store = useEquipmentStore()
      await store.getEquipments()

      expect(store.equipments.map(eq => eq.ldcId)).toEqual([1, 2, 3])
    })
  })

  describe('Store Reset Behavior', () => {
    it('should maintain state after multiple calls', async () => {
      // First successful call
      mockDuckApiClient.getEquipments.mockResolvedValue(createMockConnectRPCResponse({ equipments: mockEquipments }))
      const store = useEquipmentStore()
      await store.getEquipments()

      expect(store.equipments).toEqual(mockEquipments)

      // Second call with error
      const errorMessage = 'Server error'
      mockDuckApiClient.getEquipments.mockRejectedValue(createMockConnectRPCError(errorMessage))
      await store.getEquipments()

      expect(store.equipments).toEqual(mockEquipments) // Should maintain previous value
      expect(store.errorEquipments).toBe(errorMessage)
    })
  })

  describe('Data Integrity', () => {
    it('should preserve equipment data structure', async () => {
      mockDuckApiClient.getEquipments.mockResolvedValue(createMockConnectRPCResponse({ equipments: mockEquipments }))

      const store = useEquipmentStore()
      await store.getEquipments()

      const equipment = store.equipments[0]
      expect(equipment).toHaveProperty('id')
      expect(equipment).toHaveProperty('type')
      expect(equipment).toHaveProperty('deviceIp')
      expect(equipment).toHaveProperty('hostIp')
      expect(equipment).toHaveProperty('hostPort')
      expect(equipment).toHaveProperty('ldcId')
      expect(equipment).toHaveProperty('enabled')
    })

    it('should handle partial equipment data', async () => {
      const partialEquipment = { id: 1, type: 22 }
      mockDuckApiClient.getEquipments.mockResolvedValue(createMockConnectRPCResponse({ equipments:[partialEquipment] }))

      const store = useEquipmentStore()
      await store.getEquipments()

      expect(store.equipments[0]).toEqual(partialEquipment)
    })
  })

  describe('Performance and Edge Cases', () => {
    it('should handle large equipment list', async () => {
      const largeEquipmentList = Array.from({ length: 100 }, (_, i) => ({
        id: i + 1,
        type: 22 + (i % 5),
        deviceIp: `192.168.1.${100 + (i % 100)}`,
        hostIp: `192.168.1.${20 + (i % 10)}`,
        hostPort: 6000 + (i % 100),
        ldcId: (i % 10) + 1,
        enabled: i % 2 === 0
      }))

      mockDuckApiClient.getEquipments.mockResolvedValue(createMockConnectRPCResponse({ equipments:largeEquipmentList }))

      const store = useEquipmentStore()

      const startTime = performance.now()
      await store.getEquipments()
      const endTime = performance.now()

      expect(store.equipments).toHaveLength(100)
      expect(endTime - startTime).toBeLessThan(100) // Should complete in less than 100ms
    })

    it('should handle configuration with invalid data types', async () => {
      const invalidConfig = {
        id: 'invalid',
        type: 'invalid',
        deviceIp: null,
        enabled: 'yes'
      }

      mockDuckApiClient.getEquipments.mockResolvedValue(createMockConnectRPCResponse({ equipments:[invalidConfig] }))

      const store = useEquipmentStore()
      await store.getEquipments()

      // Store should attempt to set the data even if types are wrong
      expect(store.equipments[0].id).toBe('invalid')
      expect(store.equipments[0].type).toBe('invalid')
      expect(store.equipments[0].deviceIp).toBe(null)
      expect(store.equipments[0].enabled).toBe('yes')
    })
  })
})