import { DecoderConfiguration } from '@/gen/api_pb'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useDecoderStore } from '../decoder'
import { createMockConnectRPCResponse, createMockConnectRPCError } from '@/test/utils'
import { mockDuckApiClient } from '@/test/setup'

describe('Decoder Store Tests', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  describe('Initial State Management', () => {
    it('should have empty initial state', () => {
      const store = useDecoderStore()

      const expectedDefault: DecoderConfiguration = {
        extTrigger: 0,
        trgCode1: 0,
        trgCode2: 0,
        readPmts: false,
        readSipms: false,
        readTrigger: false,
        readFibers: false,
        splitTrigger: false,
        noDb: false,
        discard: false,
        host: '',
        user: '',
        password: '',
        dbName: '',
        writeData: false,
        useBlosc: false,
        bloscAlgorithm: '',
        compressionLevel: 0,
        bitShuffle: ''
      }

      expect(store.decoderConfig).toEqual(expectedDefault)
      expect(store.errorDecoder).toBe("")
    })

    it('should have correct reactive properties', () => {
      const store = useDecoderStore()

      expect(store.decoderConfig).toBeDefined()
      expect(store.errorDecoder).toBeDefined()
      expect(typeof store.getDecoderConfig).toBe('function')
    })
  })

  describe('API Integration - getDecoderConfig', () => {
    it('should fetch decoder configuration successfully', async () => {
      const mockConfig: DecoderConfiguration = {
        extTrigger: 1,
        trgCode1: 100,
        trgCode2: 200,
        readPmts: true,
        readSipms: true,
        readTrigger: true,
        splitTrigger: false,
        noDb: false,
        discard: false,
        host: 'localhost',
        user: 'decoder_user',
        password: 'decoder_pass',
        dbName: 'test_db',
        writeData: true,
        useBlosc: true,
        bloscAlgorithm: 'lz4',
        compressionLevel: 5,
        bitShuffle: 'true'
      }

      mockDuckApiClient.getDecoderConfiguration.mockResolvedValue(createMockConnectRPCResponse({ configuration: mockConfig }))

      const store = useDecoderStore()
      await store.getDecoderConfig()

      expect(mockDuckApiClient.getDecoderConfiguration).toHaveBeenCalledWith({})
      expect(store.decoderConfig).toEqual(mockConfig)
      expect(store.errorDecoder).toBe("")
    })

    it('should handle API errors correctly', async () => {
      const errorMessage = 'Failed to fetch decoder configuration'
      mockDuckApiClient.getDecoderConfiguration.mockRejectedValue(createMockConnectRPCError(errorMessage))

      const store = useDecoderStore()
      await store.getDecoderConfig()

      expect(mockDuckApiClient.getDecoderConfiguration).toHaveBeenCalledWith({})
      expect(store.errorDecoder).toBe(errorMessage)
    })

    it('should handle empty response', async () => {
      mockDuckApiClient.getDecoderConfiguration.mockResolvedValue(createMockConnectRPCResponse({ configuration: {} }))

      const store = useDecoderStore()
      await store.getDecoderConfig()

      expect(store.decoderConfig).toEqual({})
      expect(store.errorDecoder).toBe("")
    })
  })

  describe('Configuration Validation', () => {
    it('should handle configuration with all boolean flags enabled', async () => {
      const fullBoolConfig: Partial<DecoderConfigType> = {
        readPmts: true,
        readSipms: true,
        readTrigger: true,
        splitTrigger: true,
        noDb: true,
        discard: true,
        writeData: true,
        useBlosc: true
      }

      mockDuckApiClient.getDecoderConfiguration.mockResolvedValue(createMockConnectRPCResponse({ configuration: fullBoolConfig }))

      const store = useDecoderStore()
      await store.getDecoderConfig()

      Object.entries(fullBoolConfig).forEach(([key, value]) => {
        expect(store.decoderConfig[key as keyof DecoderConfigType]).toBe(value)
      })
    })

    it('should handle configuration with numeric values', async () => {
      const numericConfig = {
        extTrigger: 5,
        trgCode1: 150,
        trgCode2: 250
      }

      mockDuckApiClient.getDecoderConfiguration.mockResolvedValue(createMockConnectRPCResponse({ configuration: numericConfig }))

      const store = useDecoderStore()
      await store.getDecoderConfig()

      expect(store.decoderConfig.extTrigger).toBe(5)
      expect(store.decoderConfig.trgCode1).toBe(150)
      expect(store.decoderConfig.trgCode2).toBe(250)
    })

    it('should handle configuration with string values', async () => {
      const stringConfig = {
        host: 'test-hostname',
        user: 'test-user',
        password: 'test-password',
        dbName: 'test-database',
        bloscAlgorithm: 'zstd',
        compressionLevel: 9,
        bitShuffle: 'false'
      }

      mockDuckApiClient.getDecoderConfiguration.mockResolvedValue(createMockConnectRPCResponse({ configuration: stringConfig }))

      const store = useDecoderStore()
      await store.getDecoderConfig()

      Object.entries(stringConfig).forEach(([key, value]) => {
        expect(store.decoderConfig[key as keyof DecoderConfigType]).toBe(value)
      })
    })
  })

  describe('Database Configuration', () => {
    it('should handle complete database configuration', async () => {
      const dbConfig = {
        host: 'db.example.com',
        user: 'dbuser',
        password: 'dbpass123',
        dbName: 'experiment_db',
        writeData: true,
        noDb: false
      }

      mockDuckApiClient.getDecoderConfiguration.mockResolvedValue(createMockConnectRPCResponse({ configuration: dbConfig }))

      const store = useDecoderStore()
      await store.getDecoderConfig()

      expect(store.decoderConfig.host).toBe('db.example.com')
      expect(store.decoderConfig.user).toBe('dbuser')
      expect(store.decoderConfig.password).toBe('dbpass123')
      expect(store.decoderConfig.dbName).toBe('experiment_db')
      expect(store.decoderConfig.writeData).toBe(true)
      expect(store.decoderConfig.noDb).toBe(false)
    })

    it('should handle noDb mode configuration', async () => {
      const noDbConfig = {
        noDb: true,
        writeData: false,
        host: '',
        user: '',
        password: '',
        dbName: ''
      }

      mockDuckApiClient.getDecoderConfiguration.mockResolvedValue(createMockConnectRPCResponse({ configuration: noDbConfig }))

      const store = useDecoderStore()
      await store.getDecoderConfig()

      expect(store.decoderConfig.noDb).toBe(true)
      expect(store.decoderConfig.writeData).toBe(false)
      expect(store.decoderConfig.host).toBe('')
      expect(store.decoderConfig.user).toBe('')
      expect(store.decoderConfig.password).toBe('')
      expect(store.decoderConfig.dbName).toBe('')
    })
  })

  describe('Compression Configuration', () => {
    it('should handle compression settings', async () => {
      const compressionConfig = {
        useBlosc: true,
        bloscAlgorithm: 'lz4hc',
        compressionLevel: 9,
        bitShuffle: 'true'
      }

      mockDuckApiClient.getDecoderConfiguration.mockResolvedValue(createMockConnectRPCResponse({ configuration: compressionConfig }))

      const store = useDecoderStore()
      await store.getDecoderConfig()

      expect(store.decoderConfig.useBlosc).toBe(true)
      expect(store.decoderConfig.bloscAlgorithm).toBe('lz4hc')
      expect(store.decoderConfig.compressionLevel).toBe(9)
      expect(store.decoderConfig.bitShuffle).toBe('true')
    })

    it('should handle disabled compression', async () => {
      const noCompressionConfig = {
        useBlosc: false,
        bloscAlgorithm: '',
        compressionLevel: 0,
        bitShuffle: ''
      }

      mockDuckApiClient.getDecoderConfiguration.mockResolvedValue(createMockConnectRPCResponse({ configuration: noCompressionConfig }))

      const store = useDecoderStore()
      await store.getDecoderConfig()

      expect(store.decoderConfig.useBlosc).toBe(false)
      expect(store.decoderConfig.bloscAlgorithm).toBe('')
      expect(store.decoderConfig.compressionLevel).toBe(0)
      expect(store.decoderConfig.bitShuffle).toBe('')
    })
  })

  describe('Trigger Configuration', () => {
    it('should handle trigger settings', async () => {
      const triggerConfig = {
        extTrigger: 3,
        trgCode1: 42,
        trgCode2: 84,
        readTrigger: true,
        splitTrigger: true
      }

      mockDuckApiClient.getDecoderConfiguration.mockResolvedValue(createMockConnectRPCResponse({ configuration: triggerConfig }))

      const store = useDecoderStore()
      await store.getDecoderConfig()

      expect(store.decoderConfig.extTrigger).toBe(3)
      expect(store.decoderConfig.trgCode1).toBe(42)
      expect(store.decoderConfig.trgCode2).toBe(84)
      expect(store.decoderConfig.readTrigger).toBe(true)
      expect(store.decoderConfig.splitTrigger).toBe(true)
    })

    it('should handle disabled trigger reading', async () => {
      const noTriggerConfig = {
        extTrigger: 0,
        trgCode1: 0,
        trgCode2: 0,
        readTrigger: false,
        splitTrigger: false
      }

      mockDuckApiClient.getDecoderConfiguration.mockResolvedValue(createMockConnectRPCResponse({ configuration: noTriggerConfig }))

      const store = useDecoderStore()
      await store.getDecoderConfig()

      expect(store.decoderConfig.extTrigger).toBe(0)
      expect(store.decoderConfig.trgCode1).toBe(0)
      expect(store.decoderConfig.trgCode2).toBe(0)
      expect(store.decoderConfig.readTrigger).toBe(false)
      expect(store.decoderConfig.splitTrigger).toBe(false)
    })
  })

  describe('Detector Reading Configuration', () => {
    it('should handle PMT and SiPM reading settings', async () => {
      const detectorConfig = {
        readPmts: true,
        readSipms: true
      }

      mockDuckApiClient.getDecoderConfiguration.mockResolvedValue(createMockConnectRPCResponse({ configuration: detectorConfig }))

      const store = useDecoderStore()
      await store.getDecoderConfig()

      expect(store.decoderConfig.readPmts).toBe(true)
      expect(store.decoderConfig.readSipms).toBe(true)
    })

    it('should handle only PMT reading', async () => {
      const pmtOnlyConfig = {
        readPmts: true,
        readSipms: false
      }

      mockDuckApiClient.getDecoderConfiguration.mockResolvedValue(createMockConnectRPCResponse({ configuration: pmtOnlyConfig }))

      const store = useDecoderStore()
      await store.getDecoderConfig()

      expect(store.decoderConfig.readPmts).toBe(true)
      expect(store.decoderConfig.readSipms).toBe(false)
    })

    it('should handle only SiPM reading', async () => {
      const sipmOnlyConfig = {
        readPmts: false,
        readSipms: true
      }

      mockDuckApiClient.getDecoderConfiguration.mockResolvedValue(createMockConnectRPCResponse({ configuration: sipmOnlyConfig }))

      const store = useDecoderStore()
      await store.getDecoderConfig()

      expect(store.decoderConfig.readPmts).toBe(false)
      expect(store.decoderConfig.readSipms).toBe(true)
    })

    it('should handle neither detector reading', async () => {
      const noDetectorConfig = {
        readPmts: false,
        readSipms: false
      }

      mockDuckApiClient.getDecoderConfiguration.mockResolvedValue(createMockConnectRPCResponse({ configuration: noDetectorConfig }))

      const store = useDecoderStore()
      await store.getDecoderConfig()

      expect(store.decoderConfig.readPmts).toBe(false)
      expect(store.decoderConfig.readSipms).toBe(false)
    })
  })

  describe('Data Processing Configuration', () => {
    it('should handle discard mode', async () => {
      const discardConfig = {
        discard: true,
        writeData: false
      }

      mockDuckApiClient.getDecoderConfiguration.mockResolvedValue(createMockConnectRPCResponse({ configuration: discardConfig }))

      const store = useDecoderStore()
      await store.getDecoderConfig()

      expect(store.decoderConfig.discard).toBe(true)
      expect(store.decoderConfig.writeData).toBe(false)
    })

    it('should handle normal data processing', async () => {
      const normalConfig: DecoderConfiguration = {
        extTrigger: 0,
        trgCode1: 0,
        trgCode2: 0,
        readPmts: false,
        readSipms: false,
        readTrigger: false,
        splitTrigger: false,
        noDb: false,
        discard: false,
        host: '',
        user: '',
        password: '',
        dbName: '',
        writeData: true,
        useBlosc: false,
        bloscAlgorithm: '',
        compressionLevel: 0,
        bitShuffle: ''
      }

      mockDuckApiClient.getDecoderConfiguration.mockResolvedValue(createMockConnectRPCResponse({ configuration: normalConfig }))

      const store = useDecoderStore()
      await store.getDecoderConfig()

      expect(store.decoderConfig.discard).toBe(false)
      expect(store.decoderConfig.writeData).toBe(true)
    })
  })

  describe('State Management and Reactivity', () => {
    it('should update configuration reactively', async () => {
      const initialConfig: DecoderConfiguration = {
        extTrigger: 1,
        trgCode1: 0,
        trgCode2: 0,
        readPmts: true,
        readSipms: false,
        readTrigger: false,
        splitTrigger: false,
        noDb: false,
        discard: false,
        host: '',
        user: '',
        password: '',
        dbName: '',
        writeData: false,
        useBlosc: false,
        bloscAlgorithm: '',
        compressionLevel: 0,
        bitShuffle: ''
      }
      const updatedConfig: DecoderConfiguration = {
        extTrigger: 2,
        trgCode1: 0,
        trgCode2: 0,
        readPmts: false,
        readSipms: false,
        readTrigger: false,
        splitTrigger: false,
        noDb: false,
        discard: false,
        host: 'new-host',
        user: '',
        password: '',
        dbName: '',
        writeData: false,
        useBlosc: false,
        bloscAlgorithm: '',
        compressionLevel: 0,
        bitShuffle: ''
      }

      // Initial load
      mockDuckApiClient.getDecoderConfiguration.mockResolvedValue(createMockConnectRPCResponse({ configuration: initialConfig }))
      const store = useDecoderStore()
      await store.getDecoderConfig()

      expect(store.decoderConfig.extTrigger).toBe(1)
      expect(store.decoderConfig.readPmts).toBe(true)

      // Update with new configuration
      mockDuckApiClient.getDecoderConfiguration.mockResolvedValue(createMockConnectRPCResponse({ configuration: updatedConfig }))
      await store.getDecoderConfig()

      expect(store.decoderConfig.extTrigger).toBe(2)
      expect(store.decoderConfig.readPmts).toBe(false)
      expect(store.decoderConfig.host).toBe('new-host')
    })

    it('should clear error state on successful fetch', async () => {
      // Set initial error state
      mockDuckApiClient.getDecoderConfiguration.mockRejectedValue(createMockConnectRPCError('Initial decoder error'))
      const store = useDecoderStore()
      await store.getDecoderConfig()

      expect(store.errorDecoder).toBe('Initial decoder error')

      // Successful fetch should clear error
      const fullConfig: DecoderConfiguration = {
        extTrigger: 1,
        trgCode1: 0,
        trgCode2: 0,
        readPmts: false,
        readSipms: false,
        readTrigger: false,
        splitTrigger: false,
        noDb: false,
        discard: false,
        host: '',
        user: '',
        password: '',
        dbName: '',
        writeData: false,
        useBlosc: false,
        bloscAlgorithm: '',
        compressionLevel: 0,
        bitShuffle: ''
      }
      mockDuckApiClient.getDecoderConfiguration.mockResolvedValue(createMockConnectRPCResponse({ configuration: fullConfig }))
      await store.getDecoderConfig()

      expect(store.errorDecoder).toBe("")
    })

    it('should handle concurrent getDecoderConfig calls', async () => {
      const fullConfig: DecoderConfiguration = {
        extTrigger: 5,
        trgCode1: 0,
        trgCode2: 0,
        readPmts: false,
        readSipms: false,
        readTrigger: false,
        splitTrigger: false,
        noDb: false,
        discard: false,
        host: '',
        user: '',
        password: '',
        dbName: '',
        writeData: false,
        useBlosc: false,
        bloscAlgorithm: '',
        compressionLevel: 0,
        bitShuffle: ''
      }
      mockDuckApiClient.getDecoderConfiguration.mockResolvedValue(createMockConnectRPCResponse({ configuration: fullConfig }))

      const store = useDecoderStore()

      // Make multiple concurrent calls
      await Promise.all([
        store.getDecoderConfig(),
        store.getDecoderConfig(),
        store.getDecoderConfig()
      ])

      // Note: mockDuckApiClient.getDecoderConfiguration is called 4 times:
      // 1 time during store initialization + 3 times (concurrent calls)
      expect(mockDuckApiClient.getDecoderConfiguration).toHaveBeenCalledTimes(4)
      expect(store.decoderConfig.extTrigger).toBe(5)
    })
  })

  describe('Data Integrity', () => {
    it('should preserve decoder configuration structure', async () => {
      const fullConfig: DecoderConfiguration = {
        extTrigger: 1,
        trgCode1: 100,
        trgCode2: 200,
        readPmts: true,
        readSipms: false,
        readTrigger: true,
        splitTrigger: false,
        noDb: false,
        discard: false,
        host: 'localhost',
        user: 'user',
        password: 'pass',
        dbName: 'db',
        writeData: true,
        useBlosc: false,
        bloscAlgorithm: '',
        compressionLevel: 0,
        bitShuffle: ''
      }

      mockDuckApiClient.getDecoderConfiguration.mockResolvedValue(createMockConnectRPCResponse({ configuration: fullConfig }))

      const store = useDecoderStore()
      await store.getDecoderConfig()

      expect(store.decoderConfig).toEqual(fullConfig)
    })

    it('should handle partial decoder data', async () => {
      const partialConfig = { extTrigger: 1, readPmts: true }
      mockDuckApiClient.getDecoderConfiguration.mockResolvedValue(createMockConnectRPCResponse({ configuration: partialConfig }))

      const store = useDecoderStore()
      await store.getDecoderConfig()

      expect(store.decoderConfig.extTrigger).toBe(1)
      expect(store.decoderConfig.readPmts).toBe(true)
    })
  })

  describe('Performance and Edge Cases', () => {
    it('should handle rapid configuration updates', async () => {
      const configs = [
        { extTrigger: 1 },
        { extTrigger: 2 },
        { extTrigger: 3 },
        { extTrigger: 4 },
        { extTrigger: 5 }
      ]

      const store = useDecoderStore()

      const startTime = performance.now()

      for (const config of configs) {
        mockDuckApiClient.getDecoderConfiguration.mockResolvedValue(createMockConnectRPCResponse({ configuration: config }))
        await store.getDecoderConfig()
      }

      const endTime = performance.now()

      expect(endTime - startTime).toBeLessThan(100) // Should complete quickly
      expect(store.decoderConfig.extTrigger).toBe(5)
    })

    it('should handle configuration with invalid data types', async () => {
      const invalidConfig = {
        extTrigger: 'invalid',
        readPmts: 'true',
        host: 123,
        useBlosc: 'yes'
      }

      mockDuckApiClient.getDecoderConfiguration.mockResolvedValue(createMockConnectRPCResponse({ configuration: invalidConfig }))

      const store = useDecoderStore()
      await store.getDecoderConfig()

      // Store should attempt to set the data even if types are wrong
      expect(store.decoderConfig.extTrigger).toBe('invalid')
      expect(store.decoderConfig.readPmts).toBe('true')
      expect(store.decoderConfig.host).toBe(123)
      expect(store.decoderConfig.useBlosc).toBe('yes')
    })

    it('should handle null response', async () => {
      mockDuckApiClient.getDecoderConfiguration.mockResolvedValue(createMockConnectRPCResponse({ configuration: null }))

      const store = useDecoderStore()
      await store.getDecoderConfig()

      expect(store.decoderConfig).toEqual(new DecoderConfiguration())
      expect(store.errorDecoder).toBe("")
    })
  })
})
