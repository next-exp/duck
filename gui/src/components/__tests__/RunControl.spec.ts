import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import RunControl from '../RunControl.vue'
import { flushPromises, createMockConnectRPCResponse, createMockConnectRPCError } from '@/test/utils'
import { mockDuckApiClient } from '@/test/setup'
import { useMessagesStore } from '@/stores/messages'

describe('RunControl Component', () => {
  let wrapper: VueWrapper<any>
  let messagesStore: any

  beforeEach(async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    
    messagesStore = useMessagesStore()
    // Set store state to enable buttons
    messagesStore.controlEnabled = true
    messagesStore.startRunEnabled = true
    messagesStore.stopRunEnabled = true
    messagesStore.waitForSummaries = false
    
    vi.clearAllMocks()

    // Create the component
    wrapper = mount(RunControl, {
      global: {
        plugins: [pinia],
        stubs: {
          AlertError: true,
          'font-awesome-icon': true
        }
      }
    })

    // Enable the control buttons by toggling the checkbox
    const toggle = wrapper.find('input[type="checkbox"]')
    await toggle.setValue(true)
    await wrapper.vm.$nextTick()
  })

  describe('rendering', () => {
    it('should render the component correctly', () => {
      expect(wrapper.find('h2').text()).toBe('Run control')
      expect(wrapper.find('button[data-testid="start-run-button"]').exists()).toBe(true)
      expect(wrapper.find('button[data-testid="stop-run-button"]').exists()).toBe(true)
      expect(wrapper.find('button[data-testid="force-stop-button"]').exists()).toBe(true)
      expect(wrapper.find('button[data-testid="restart-services-button"]').exists()).toBe(true)
    })

    it('should have proper button labels', () => {
      expect(wrapper.find('button[data-testid="start-run-button"]').text()).toBe('Start run')
      expect(wrapper.find('button[data-testid="stop-run-button"]').text()).toBe('Stop run')
      expect(wrapper.find('button[data-testid="force-stop-button"]').text()).toBe('Force stop')
      expect(wrapper.find('button[data-testid="restart-services-button"]').text()).toBe('Restart services')
    })
  })

  describe('button interactions', () => {
    it('should call start API when start button clicked', async () => {
      vi.clearAllMocks() // Clear mocks from initialization
      mockDuckApiClient.startRun.mockResolvedValue(createMockConnectRPCResponse({}))

      const startButton = wrapper.find('button[data-testid="start-run-button"]')
      await startButton.trigger('click')
      await flushPromises()

      expect(mockDuckApiClient.startRun).toHaveBeenCalledWith({})
    })

    it('should call stop API when stop button clicked', async () => {
      vi.clearAllMocks() // Clear mocks from initialization
      mockDuckApiClient.stopRun.mockResolvedValue(createMockConnectRPCResponse({}))

      const stopButton = wrapper.find('button[data-testid="stop-run-button"]')
      await stopButton.trigger('click')
      await flushPromises()

      expect(mockDuckApiClient.stopRun).toHaveBeenCalledWith({})
    })

    it('should call force stop API when force stop button clicked', async () => {
      vi.clearAllMocks() // Clear mocks from initialization
      mockDuckApiClient.forceStopRun.mockResolvedValue(createMockConnectRPCResponse({}))

      const forceStopButton = wrapper.find('button[data-testid="force-stop-button"]')
      await forceStopButton.trigger('click')
      await flushPromises()

      expect(mockDuckApiClient.forceStopRun).toHaveBeenCalledWith({})
    })

    it('should call restart services API when restart button clicked', async () => {
      vi.clearAllMocks() // Clear mocks from initialization
      mockDuckApiClient.restartServices.mockResolvedValue(createMockConnectRPCResponse({}))

      const restartButton = wrapper.find('button[data-testid="restart-services-button"]')
      await restartButton.trigger('click')
      await flushPromises()

      expect(mockDuckApiClient.restartServices).toHaveBeenCalledWith({})
    })
  })

  describe('error handling', () => {
    it('should handle start run error correctly', async () => {
      vi.clearAllMocks() // Clear mocks from initialization
      const errorMessage = 'Start failed'
      mockDuckApiClient.startRun.mockRejectedValue(createMockConnectRPCError(errorMessage))

      const startButton = wrapper.find('button[data-testid="start-run-button"]')
      await startButton.trigger('click')
      await flushPromises()
      await wrapper.vm.$nextTick()

      // Error should be stored in the component's errorMsg ref
      expect(wrapper.vm.errorMsg).toBe(errorMessage)
    })

    it('should handle stop run error correctly', async () => {
      vi.clearAllMocks() // Clear mocks from initialization
      const errorMessage = 'Stop failed'
      mockDuckApiClient.stopRun.mockRejectedValue(createMockConnectRPCError(errorMessage))

      const stopButton = wrapper.find('button[data-testid="stop-run-button"]')
      await stopButton.trigger('click')
      await flushPromises()
      await wrapper.vm.$nextTick()

      // Error should be stored in the component's errorMsg ref
      expect(wrapper.vm.errorMsg).toBe(errorMessage)
    })
  })

  describe('control toggle', () => {
    it('should toggle control buttons state', async () => {
      const toggle = wrapper.find('input[type="checkbox"]')

      // Initially enabled (from beforeEach)
      expect(toggle.element.checked).toBe(true)
      expect(wrapper.text()).toContain('Disable control')

      // Disable controls
      await toggle.setValue(false)
      await wrapper.vm.$nextTick()
      expect(toggle.element.checked).toBe(false)
      expect(wrapper.text()).toContain('Enable control')

      // Re-enable controls
      await toggle.setValue(true)
      await wrapper.vm.$nextTick()
      expect(toggle.element.checked).toBe(true)
      expect(wrapper.text()).toContain('Disable control')
    })
  })

  describe('button states', () => {
    it('should have data-test-id attributes on all buttons', () => {
      const buttons = wrapper.findAll('button')
      expect(buttons).toHaveLength(4)

      const startButton = wrapper.find('button[data-testid="start-run-button"]')
      const stopButton = wrapper.find('button[data-testid="stop-run-button"]')
      const forceStopButton = wrapper.find('button[data-testid="force-stop-button"]')
      const restartButton = wrapper.find('button[data-testid="restart-services-button"]')

      expect(startButton.exists()).toBe(true)
      expect(stopButton.exists()).toBe(true)
      expect(forceStopButton.exists()).toBe(true)
      expect(restartButton.exists()).toBe(true)
    })
  })

  describe('accessibility', () => {
    it('should have proper form labeling for toggle', () => {
      const toggle = wrapper.find('input[type="checkbox"]')
      expect(toggle.attributes('id')).toBe('lock-controls')

      const label = wrapper.find('label[for="lock-controls"]')
      expect(label.exists()).toBe(true)
    })

    it('should have descriptive button text', () => {
      const buttons = wrapper.findAll('button')
      const buttonLabels = buttons.map(button => button.text())

      expect(buttonLabels).toContain('Start run')
      expect(buttonLabels).toContain('Stop run')
      expect(buttonLabels).toContain('Force stop')
      expect(buttonLabels).toContain('Restart services')
    })
  })
})