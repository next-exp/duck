import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { ref, computed } from 'vue'
import { useMessagesStore } from '../messages'
import { mockGDCs, mockLDCs, mockWebSocketMessages } from '@/test/fixtures/stores'
import { flushPromises, createMockConnectRPCResponse, createMockConnectRPCError } from '@/test/utils'
import { mockDuckApiClient } from '@/test/setup'

// Mock the stores that messages store depends on
vi.mock('../gdc', () => ({
  useGDCStore: () => ({
    getGDCs: vi.fn(),
    gdcs: ref(mockGDCs),
    errorGDCs: ref(''),
    nActiveGDCs: computed(() => 1)
  })
}))

vi.mock('../ldc', () => ({
  useLDCStore: () => ({
    getLDCs: vi.fn(),
    ldcs: ref(mockLDCs),
    errorLDCs: ref(''),
    nActiveLDCs: computed(() => 1)
  })
}))

describe('Messages Store', () => {
  let store: ReturnType<typeof useMessagesStore>
  let publicationHandler: Function | null = null
  let mockCentrifugeInstance: any = null
  let centrifugeEventHandlers: Record<string, Function> = {}

  // Helper to simulate publication events
  const simulatePublication = (message: any) => {
    if (publicationHandler) {
      publicationHandler({ data: message })
    }
  }

  beforeEach(async () => {
    setActivePinia(createPinia())
    vi.clearAllMocks()

    // Reset publication handler and event handlers
    publicationHandler = null
    centrifugeEventHandlers = {}
    mockCentrifugeInstance = null

    // Mock fetch for token - called by the store
    global.fetch = vi.fn(() =>
      Promise.resolve({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ token: 'mock-token' })
      }) as any
    )

    // Get the mocked Centrifuge from the global setup
    const { Centrifuge } = await import('centrifuge')
    const mockCentrifugeConstructor = vi.mocked(Centrifuge)

    // Clear previous mock implementation
    mockCentrifugeConstructor.mockReset()

    // Capture the subscription's publication handler when it's set up
    mockCentrifugeConstructor.mockImplementation((() => {
      // Create event handlers map for this instance
      const instanceEventHandlers: Record<string, Function> = {}

      const mockSubscription = {
        subscribe: vi.fn(),
        on: vi.fn((event: string, handler: Function) => {
          if (event === 'publication') {
            publicationHandler = handler
          }
        }),
        unsubscribe: vi.fn()
      }

      const instance = {
        on: vi.fn((event: string, handler: Function) => {
          instanceEventHandlers[event] = handler
          centrifugeEventHandlers[event] = handler
        }),
        newSubscription: vi.fn(() => mockSubscription),
        connect: vi.fn(),
        disconnect: vi.fn()
      }

      // Store reference to instance for test access
      mockCentrifugeInstance = instance

      return instance
    }) as any)

    store = useMessagesStore()
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  describe('initial state', () => {
    it('should have correct initial state', () => {
      expect(store.messages).toEqual([])
      expect(store.messagesDebug).toEqual([])
      expect(store.evts).toEqual({})
      expect(store.bytes).toEqual({})
      expect(store.states).toEqual({})
      expect(store.status).toBe('')
      expect(store.controlEnabled).toBe(false)
      expect(store.startRunEnabled).toBe(false)
      expect(store.stopRunEnabled).toBe(false)
      expect(store.runNumber).toBe(0)
      expect(store.errorCentrifuge).toBe('')
      expect(store.warningCentrifuge).toBe('')
      expect(store.waitForSummaries).toBe(false)
      expect(store.nSummariesReceived).toBe(0)
    })

    it('should initialize total counters to zero', () => {
      expect(store.evtsTotal).toEqual({ gdc: 0, ldc: 0 })
      expect(store.bytesTotal).toEqual({ gdc: 0, ldc: 0 })
      expect(store.evtRateAvgTotal).toEqual({ gdc: 0, ldc: 0 })
      expect(store.evtRateCurrentTotal).toEqual({ gdc: 0, ldc: 0 })
      expect(store.byteRateAvgTotal).toEqual({ gdc: 0, ldc: 0 })
      expect(store.byteRateCurrentTotal).toEqual({ gdc: 0, ldc: 0 })
    })
  })

  describe('WebSocket initialization', () => {
    it('should initialize Centrifuge connection', async () => {
      const { Centrifuge } = await import('centrifuge')
      expect(Centrifuge).toHaveBeenCalledWith(
        expect.stringContaining('/connection/websocket'),
        expect.objectContaining({
          getToken: expect.any(Function)
        })
      )
    })

    it('should subscribe to duck channel', async () => {
      expect(mockCentrifugeInstance).toBeDefined()
      expect(mockCentrifugeInstance.newSubscription).toHaveBeenCalledWith('duck')
      const mockSubscription = mockCentrifugeInstance.newSubscription.mock.results[0]?.value
      expect(mockSubscription).toBeDefined()
      expect(mockSubscription.subscribe).toHaveBeenCalled()
    })

    it('should connect to Centrifuge', async () => {
      expect(mockCentrifugeInstance).toBeDefined()
      expect(mockCentrifugeInstance.connect).toHaveBeenCalled()
    })

    it('should set up connection event handlers', async () => {
      expect(mockCentrifugeInstance).toBeDefined()
      expect(mockCentrifugeInstance.on).toHaveBeenCalledWith('connected', expect.any(Function))
      expect(mockCentrifugeInstance.on).toHaveBeenCalledWith('connecting', expect.any(Function))
      expect(mockCentrifugeInstance.on).toHaveBeenCalledWith('disconnected', expect.any(Function))
      expect(mockCentrifugeInstance.on).toHaveBeenCalledWith('error', expect.any(Function))
    })
  })

  describe('message processing', () => {
    describe('metric messages', () => {
      it('should process metric messages correctly', () => {
        const metricData = {
          EventCounter: 1000,
          ByteCounter: 5000000,
          CurrentDataRate: 1000,
          CurrentTrgRate: 10,
          AvgDataRate: 950,
          AvgTrgRate: 9.5
        }

        simulatePublication({
          ...mockWebSocketMessages.metricMessage,
          value: JSON.stringify(metricData)
        })

        expect(store.evts['gdc1']).toBe(1000)
        expect(store.bytes['gdc1']).toBe(5000000)
        expect(store.byteRateCurrent['gdc1']).toBe(1000)
        expect(store.evtRateCurrent['gdc1']).toBe(10)
        expect(store.byteRateAvg['gdc1']).toBe(950)
        expect(store.evtRateAvg['gdc1']).toBe(9.5)
      })

      it('should update total counters for GDC metrics', () => {
        const metricData = {
          EventCounter: 1000,
          ByteCounter: 5000000,
          CurrentDataRate: 1000,
          CurrentTrgRate: 10,
          AvgDataRate: 950,
          AvgTrgRate: 9.5
        }

        simulatePublication({
          ...mockWebSocketMessages.metricMessage,
          value: JSON.stringify(metricData)
        })

        expect(store.evtsTotal['gdc']).toBe(1000)
        expect(store.bytesTotal['gdc']).toBe(5000000)
        expect(store.evtRateCurrentTotal['gdc']).toBe(10)
        expect(store.evtRateAvgTotal['gdc']).toBe(9.5)
        expect(store.byteRateCurrentTotal['gdc']).toBe(1000)
        expect(store.byteRateAvgTotal['gdc']).toBe(950)
      })

      it('should update total counters for LDC metrics', () => {
        const metricData = {
          EventCounter: 500,
          ByteCounter: 2500000,
          CurrentDataRate: 500,
          CurrentTrgRate: 5,
          AvgDataRate: 475,
          AvgTrgRate: 4.75
        }

        simulatePublication({
          ...mockWebSocketMessages.metricMessage,
          host: 'ldc1',
          value: JSON.stringify(metricData)
        })

        expect(store.evtsTotal['ldc']).toBe(500)
        expect(store.bytesTotal['ldc']).toBe(2500000)
        expect(store.evtRateCurrentTotal['ldc']).toBe(5)
        expect(store.evtRateAvgTotal['ldc']).toBe(4.75)
        expect(store.byteRateCurrentTotal['ldc']).toBe(500)
        expect(store.byteRateAvgTotal['ldc']).toBe(475)
      })
    })

    describe('state messages', () => {
      it('should process RUNNING state and update control states', () => {
        simulatePublication({
          ...mockWebSocketMessages.stateMessage,
          value: JSON.stringify({ state: 'RUNNING' })
        })

        expect(store.states['ldc1']).toBe('RUNNING')
        expect(store.status).toBe('RUNNING')
        expect(store.controlEnabled).toBe(true)
        expect(store.startRunEnabled).toBe(false)
        expect(store.stopRunEnabled).toBe(true)
      })

      it('should process INITIALIZED state correctly', () => {
        simulatePublication({
          ...mockWebSocketMessages.stateMessage,
          value: JSON.stringify({ state: 'INITIALIZED' })
        })

        expect(store.states['ldc1']).toBe('INITIALIZED')
        expect(store.status).toBe('INITIALIZED')
        expect(store.startRunEnabled).toBe(true)
        expect(store.stopRunEnabled).toBe(false)
      })

      it('should handle mixed server states correctly', () => {
        // First server reports RUNNING
        simulatePublication({
          host: 'ldc1',
          timestamp: '2025-10-16T11:30:00Z',
          type: 'state',
          value: JSON.stringify({ state: 'RUNNING' }),
          run: 123
        })

        expect(store.status).toBe('RUNNING')

        // Second server reports INITIALIZED (mixed states)
        simulatePublication({
          host: 'ldc2',
          timestamp: '2025-10-16T11:30:00Z',
          type: 'state',
          value: JSON.stringify({ state: 'INITIALIZED' }),
          run: 123
        })

        expect(store.status).toBe('Changing state...')
        expect(store.controlEnabled).toBe(false)
        expect(store.startRunEnabled).toBe(false)
        expect(store.stopRunEnabled).toBe(false)
      })

      it('should handle state transitions back to synchronized state', () => {
        // Start with mixed states
        simulatePublication({
          host: 'ldc1',
          timestamp: '2025-10-16T11:30:00Z',
          type: 'state',
          value: JSON.stringify({ state: 'RUNNING' }),
          run: 123
        })

        simulatePublication({
          host: 'ldc2',
          timestamp: '2025-10-16T11:30:00Z',
          type: 'state',
          value: JSON.stringify({ state: 'INITIALIZED' }),
          run: 123
        })

        expect(store.status).toBe('Changing state...')

        // Bring them back to synchronized state
        simulatePublication({
          host: 'ldc2',
          timestamp: '2025-10-16T11:30:00Z',
          type: 'state',
          value: JSON.stringify({ state: 'RUNNING' }),
          run: 123
        })

        expect(store.status).toBe('RUNNING')
        expect(store.controlEnabled).toBe(true)
      })
    })

    describe('output file messages', () => {
      it('should process output file messages', () => {
        simulatePublication({
          ...mockWebSocketMessages.outputMessage,
          value: JSON.stringify({
            server: 'gdc1',
            subrun: 1
          })
        })

        expect(store.fileNumbers['gdc1']).toBe(1)
      })

      it('should handle multiple output file messages', () => {
        simulatePublication({
          ...mockWebSocketMessages.outputMessage,
          value: JSON.stringify({
            server: 'gdc1',
            subrun: 1
          })
        })

        simulatePublication({
          host: 'gdc1',
          timestamp: '2025-10-16T11:31:00Z',
          type: 'output',
          value: JSON.stringify({
            server: 'gdc1',
            subrun: 2
          }),
          run: 123
        })

        expect(store.fileNumbers['gdc1']).toBe(2)
      })
    })

    describe('summary messages', () => {
      it('should process summary messages correctly', () => {
        simulatePublication({
          ...mockWebSocketMessages.summaryMessage,
          value: JSON.stringify({
            events: 500,
            bytes: 2500000,
            errors: 2
          })
        })

        expect(store.nSummariesReceived).toBe(1)
        expect(store.messages).toHaveLength(1)
        expect(store.messages[0].type).toBe('summary')
        expect(store.messages[0].value).toContain('received 500 and 2500000 bytes')
      })

      it('should handle summary waiting correctly', () => {
        expect(store.waitForSummaries).toBe(false)
        expect(store.nSummariesReceived).toBe(0)

        // Start waiting for summaries (simulating stop run)
        store.restartRun()
        expect(store.waitForSummaries).toBe(false) // Should be reset by restartRun

        // Manually set to test summary counting
        store.waitForSummaries = true
        store.nSummariesReceived = 0

        simulatePublication({
          ...mockWebSocketMessages.summaryMessage
        })

        expect(store.nSummariesReceived).toBe(1)
        // Should remain true until all summaries received
        expect(store.waitForSummaries).toBe(true) // 1 out of expected 2 (1 GDC + 1 LDC)
      })

      it('should format summary messages correctly', () => {
        simulatePublication({
          host: 'summary-server',
          timestamp: '2025-10-16T11:30:00Z',
          type: 'summary',
          value: JSON.stringify({
            events: 1000,
            bytes: 5000000,
            errors: 5
          }),
          run: 123
        })

        expect(store.messages[0].host).toBe('summary-server')
        expect(store.messages[0].type).toBe('summary')
        expect(store.messages[0].value).toContain('received 1000 and 5000000 bytes')
      })
    })

    describe('debug messages', () => {
      it('should process debug messages', () => {
        const debugMessage = {
          host: 'gdc1',
          timestamp: '2025-10-16T11:30:00Z',
          type: 'debug',
          value: 'Debug information',
          run: 123
        }

        simulatePublication(debugMessage)

        expect(store.messagesDebug).toHaveLength(1)
        expect(store.messagesDebug[0]).toEqual(debugMessage)
      })

      it('should limit debug messages buffer', () => {
        const numberOfMessages = 200 // This is the limit in the store

        // Add messages up to the limit
        for (let i = 0; i < numberOfMessages; i++) {
          simulatePublication({
            host: 'gdc1',
            timestamp: '2025-10-16T11:30:00Z',
            type: 'debug',
            value: `Debug message ${i}`,
            run: 123
          })
        }

        expect(store.messagesDebug).toHaveLength(numberOfMessages)

        // Add one more message
        simulatePublication({
          host: 'gdc1',
          timestamp: '2025-10-16T11:30:00Z',
          type: 'debug',
          value: 'Latest message',
          run: 123
        })

        expect(store.messagesDebug).toHaveLength(numberOfMessages)
        expect(store.messagesDebug[numberOfMessages - 1].value).toBe('Latest message')
      })
    })

    describe('regular messages', () => {
      it('should process regular messages', () => {
        const regularMessage = {
          host: 'gdc1',
          timestamp: '2025-10-16T11:30:00Z',
          type: 'info',
          value: 'Regular message',
          run: 123
        }

        simulatePublication(regularMessage)

        expect(store.messages).toHaveLength(1)
        expect(store.messages[0]).toEqual(regularMessage)
      })

      it('should limit regular messages buffer', () => {
        const numberOfMessages = 200 // This is the limit in the store

        // Add messages up to the limit
        for (let i = 0; i < numberOfMessages; i++) {
          simulatePublication({
            host: 'gdc1',
            timestamp: '2025-10-16T11:30:00Z',
            type: 'info',
            value: `Message ${i}`,
            run: 123
          })
        }

        expect(store.messages).toHaveLength(numberOfMessages)

        // Add one more message
        simulatePublication({
          host: 'gdc1',
          timestamp: '2025-10-16T11:30:00Z',
          type: 'info',
          value: 'Latest message',
          run: 123
        })

        expect(store.messages).toHaveLength(numberOfMessages)
        expect(store.messages[numberOfMessages - 1].value).toBe('Latest message')
      })
    })

    describe('error handling in message processing', () => {
      it('should handle malformed JSON gracefully', () => {
        const badMessage = {
          host: 'gdc1',
          timestamp: '2025-10-16T11:30:00Z',
          type: 'metric',
          value: 'invalid json',
          run: 123
        }

        // Should not throw error
        expect(() => simulatePublication(badMessage)).not.toThrow()
      })

      it('should handle empty messages', () => {
        const emptyMessage = {
          host: 'gdc1',
          timestamp: '2025-10-16T11:30:00Z',
          type: 'info',
          value: '',
          run: 123
        }

        simulatePublication(emptyMessage)

        expect(store.messages).toHaveLength(1)
        expect(store.messages[0].value).toBe('')
      })
    })
  })

  describe('WebSocket event handlers', () => {
    const getEventHandler = async (eventName: string) => {
      // Use the tracked event handlers from the mock instance
      return centrifugeEventHandlers[eventName] || null
    }

    it('should handle connected event', async () => {
      const connectedHandler = await getEventHandler('connected')

      expect(connectedHandler).toBeDefined()
      expect(typeof connectedHandler).toBe('function')

      connectedHandler({})

      expect(store.errorCentrifuge).toBe('')
      expect(store.warningCentrifuge).toBe('')
      // Should also call getStates()
      expect(mockDuckApiClient.getProcessStates).toHaveBeenCalledWith({})
    })

    it('should handle connecting event', async () => {
      const connectingHandler = await getEventHandler('connecting')

      expect(connectingHandler).toBeDefined()

      connectingHandler({})

      expect(store.warningCentrifuge).toBe('Connecting to server...')
      expect(store.errorCentrifuge).toBe('')
    })

    it('should handle disconnected event', async () => {
      const disconnectedHandler = await getEventHandler('disconnected')

      expect(disconnectedHandler).toBeDefined()

      disconnectedHandler({})

      expect(store.errorCentrifuge).toBe('Disconnected from server, please refresh the page')
      expect(store.warningCentrifuge).toBe('')
    })

    it('should handle error event', async () => {
      const errorHandler = await getEventHandler('error')

      expect(errorHandler).toBeDefined()

      errorHandler({ code: 'connection_error', message: 'Failed to connect' })

      expect(store.errorCentrifuge).toBe('Error in server connection, retrying...')
      expect(store.warningCentrifuge).toBe('')
    })

    it('should call getStates on connected', async () => {
      mockDuckApiClient.getProcessStates.mockResolvedValue(createMockConnectRPCResponse({ states: { state: 'OK' } }))

      const connectedHandler = await getEventHandler('connected')
      connectedHandler({})

      expect(mockDuckApiClient.getProcessStates).toHaveBeenCalledWith({})
    })

    it('should handle getStates error gracefully', async () => {
      const errorMessage = 'States API error'
      mockDuckApiClient.getProcessStates.mockRejectedValue(createMockConnectRPCError(errorMessage))

      const connectedHandler = await getEventHandler('connected')
      connectedHandler({})

      await flushPromises()

      expect(store.errorStates).toBe(errorMessage)
    })
  })

  describe('counter management', () => {
    it('should reset total counters correctly', () => {
      // Set some initial values manually for testing
      store.evtsTotal['gdc'] = 100
      store.evtsTotal['ldc'] = 50
      store.bytesTotal['gdc'] = 1000000
      store.bytesTotal['ldc'] = 500000

      store.restartRun()

      expect(store.evtsTotal['gdc']).toBe(0)
      expect(store.evtsTotal['ldc']).toBe(0)
      expect(store.bytesTotal['gdc']).toBe(0)
      expect(store.bytesTotal['ldc']).toBe(0)
    })

    it('should reset server-specific counters correctly', () => {
      // Set some initial values
      store.evts['gdc1'] = 1000
      store.evts['ldc1'] = 500
      store.bytes['gdc1'] = 5000000
      store.bytes['ldc1'] = 2500000

      store.restartRun()

      expect(store.evts['gdc1']).toBe(0)
      expect(store.evts['ldc1']).toBe(0)
      expect(store.bytes['gdc1']).toBe(0)
      expect(store.bytes['ldc1']).toBe(0)
    })

    it('should reset summary counters', () => {
      store.nSummariesReceived = 5
      store.waitForSummaries = true

      store.restartRun()

      expect(store.nSummariesReceived).toBe(0)
      expect(store.waitForSummaries).toBe(false)
    })
  })

  describe('API calls', () => {
    it('should get run number successfully', async () => {
      const mockRunNumber = 123
      mockDuckApiClient.getRunNumber.mockResolvedValue(createMockConnectRPCResponse({ runNumber: mockRunNumber }))

      await store.getRunNumber()

      expect(mockDuckApiClient.getRunNumber).toHaveBeenCalledWith({})
      expect(store.runNumber).toBe(mockRunNumber)
      expect(store.errorRunNumber).toBe('')
    })

    it('should handle run number API error', async () => {
      const errorMessage = 'API Error'
      mockDuckApiClient.getRunNumber.mockRejectedValue(createMockConnectRPCError(errorMessage))

      await store.getRunNumber()

      expect(store.errorRunNumber).toBe(errorMessage)
    })

    it('should handle run number API timeout', async () => {
      const timeoutError = new Error('Request timeout')
      timeoutError.name = 'TimeoutError'
      mockDuckApiClient.getRunNumber.mockRejectedValue(timeoutError)

      await store.getRunNumber()

      expect(store.errorRunNumber).toBe('Request timeout')
    })

    it('should handle network error in run number API', async () => {
      const networkError = new Error('Network Error')
      networkError.name = 'NetworkError'
      mockDuckApiClient.getRunNumber.mockRejectedValue(networkError)

      await store.getRunNumber()

      expect(store.errorRunNumber).toBe('Network Error')
    })
  })

  describe('total counters calculation', () => {
    it('should calculate GDC total counters correctly', async () => {
      const metricData = {
        EventCounter: 1000,
        ByteCounter: 5000000,
        CurrentDataRate: 1000,
        CurrentTrgRate: 10,
        AvgDataRate: 950,
        AvgTrgRate: 9.5
      }

      // Simulate multiple GDC metrics
      simulatePublication({
        host: 'gdc1',
        timestamp: '2025-10-16T11:30:00Z',
        type: 'metric',
        value: JSON.stringify(metricData)
      })

      simulatePublication({
        host: 'gdc2',
        timestamp: '2025-10-16T11:30:00Z',
        type: 'metric',
        value: JSON.stringify({
          EventCounter: 500,
          ByteCounter: 2500000,
          CurrentDataRate: 500,
          CurrentTrgRate: 5,
          AvgDataRate: 475,
          AvgTrgRate: 4.75
        })
      })

      await flushPromises()

      expect(store.evtsTotal['gdc']).toBe(1500) // 1000 + 500
      expect(store.bytesTotal['gdc']).toBe(7500000) // 5000000 + 2500000
      expect(store.evtRateCurrentTotal['gdc']).toBe(15) // 10 + 5
      expect(store.evtRateAvgTotal['gdc']).toBe(14.25) // 9.5 + 4.75
      expect(store.byteRateCurrentTotal['gdc']).toBe(1500) // 1000 + 500
      expect(store.byteRateAvgTotal['gdc']).toBe(1425) // 950 + 475
    })

    it('should calculate LDC total counters correctly', async () => {
      const metricData = {
        EventCounter: 500,
        ByteCounter: 2500000,
        CurrentDataRate: 500,
        CurrentTrgRate: 5,
        AvgDataRate: 475,
        AvgTrgRate: 4.75
      }

      // Simulate LDC metrics
      simulatePublication({
        host: 'ldc1',
        timestamp: '2025-10-16T11:30:00Z',
        type: 'metric',
        value: JSON.stringify(metricData)
      })

      simulatePublication({
        host: 'ldc2',
        timestamp: '2025-10-16T11:30:00Z',
        type: 'metric',
        value: JSON.stringify({
          EventCounter: 300,
          ByteCounter: 1500000,
          CurrentDataRate: 300,
          CurrentTrgRate: 3,
          AvgDataRate: 285,
          AvgTrgRate: 2.85
        })
      })

      await flushPromises()

      expect(store.evtsTotal['ldc']).toBe(800) // 500 + 300
      expect(store.bytesTotal['ldc']).toBe(4000000) // 2500000 + 1500000
      expect(store.evtRateCurrentTotal['ldc']).toBe(8) // 5 + 3
      expect(store.evtRateAvgTotal['ldc']).toBe(7.6) // 4.75 + 2.85
      expect(store.byteRateCurrentTotal['ldc']).toBe(800) // 500 + 300
      expect(store.byteRateAvgTotal['ldc']).toBe(760) // 475 + 285
    })

    it('should separate GDC and LDC counters correctly', async () => {
      const gdcMetric = {
        EventCounter: 1000,
        ByteCounter: 5000000,
        CurrentDataRate: 1000,
        CurrentTrgRate: 10,
        AvgDataRate: 950,
        AvgTrgRate: 9.5
      }

      const ldcMetric = {
        EventCounter: 500,
        ByteCounter: 2500000,
        CurrentDataRate: 500,
        CurrentTrgRate: 5,
        AvgDataRate: 475,
        AvgTrgRate: 4.75
      }

      // Simulate GDC metrics
      simulatePublication({
        host: 'gdc1',
        timestamp: '2025-10-16T11:30:00Z',
        type: 'metric',
        value: JSON.stringify(gdcMetric)
      })

      // Simulate LDC metrics
      simulatePublication({
        host: 'ldc1',
        timestamp: '2025-10-16T11:30:00Z',
        type: 'metric',
        value: JSON.stringify(ldcMetric)
      })

      await flushPromises()

      // Verify GDC counters
      expect(store.evtsTotal['gdc']).toBe(1000)
      expect(store.bytesTotal['gdc']).toBe(5000000)

      // Verify LDC counters
      expect(store.evtsTotal['ldc']).toBe(500)
      expect(store.bytesTotal['ldc']).toBe(2500000)

      // Verify they are separate
      expect(store.evtsTotal['gdc']).not.toBe(store.evtsTotal['ldc'])
      expect(store.bytesTotal['gdc']).not.toBe(store.bytesTotal['ldc'])
    })
  })

  describe('edge cases and error scenarios', () => {
    it('should handle empty message data', () => {
      simulatePublication({
        host: '',
        timestamp: '',
        type: '',
        value: '',
        run: 0
      })

      expect(store.messages).toHaveLength(1)
      expect(store.messages[0].host).toBe('')
    })

    it('should handle missing message fields', () => {
      const incompleteMessage = {
        host: 'gdc1',
        // Missing other fields
      }

      simulatePublication(incompleteMessage as any)

      // Should not throw
      expect(store.messages).toHaveLength(1)
    })

    it('should handle null/undefined message values', () => {
      const nullMessage = {
        host: null,
        timestamp: null,
        type: null,
        value: null,
        run: null
      }

      simulatePublication(nullMessage as any)

      expect(store.messages).toHaveLength(1)
    })

    it('should handle very large metric values', () => {
      const largeMetricData = {
        EventCounter: Number.MAX_SAFE_INTEGER,
        ByteCounter: Number.MAX_SAFE_INTEGER,
        CurrentDataRate: Number.MAX_SAFE_INTEGER,
        CurrentTrgRate: Number.MAX_SAFE_INTEGER,
        AvgDataRate: Number.MAX_SAFE_INTEGER,
        AvgTrgRate: Number.MAX_SAFE_INTEGER
      }

      simulatePublication({
        ...mockWebSocketMessages.metricMessage,
        value: JSON.stringify(largeMetricData)
      })

      expect(store.evts['gdc1']).toBe(Number.MAX_SAFE_INTEGER)
      expect(store.bytes['gdc1']).toBe(Number.MAX_SAFE_INTEGER)
    })

    it('should handle rapid message processing', async () => {
      const rapidMessageCount = 100
      const startTime = Date.now()

      for (let i = 0; i < rapidMessageCount; i++) {
        simulatePublication({
          host: 'gdc1',
          timestamp: '2025-10-16T11:30:00Z',
          type: 'metric',
          value: JSON.stringify({
            EventCounter: i,
            ByteCounter: i * 1000,
            CurrentDataRate: i * 10,
            CurrentTrgRate: i,
            AvgDataRate: i * 9.5,
            AvgTrgRate: i * 0.95
          }),
          run: 123
        })
      }

      const endTime = Date.now()
      const processingTime = endTime - startTime

      // Should process quickly (less than 500ms for 100 messages)
      expect(processingTime).toBeLessThan(500)
      expect(store.evts['gdc1']).toBe(rapidMessageCount - 1) // Last message value
    })
  })
})