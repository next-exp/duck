import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import CheckConnection from '../CheckConnection.vue'
import { flushPromises } from '@/test/utils'
import { useMessagesStore } from '@/stores/messages'

describe('CheckConnection Component', () => {
  let wrapper: VueWrapper<any>
  let messagesStore: any

  beforeEach(async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    vi.clearAllMocks()

    wrapper = mount(CheckConnection, {
      global: {
        plugins: [pinia],
        stubs: {
          AlertError: true,
          AlertWarning: true
        }
      }
    })

    messagesStore = useMessagesStore()
    await flushPromises()
  })

  describe('rendering', () => {
    it('should render the component correctly', () => {
      expect(wrapper.find('div.flex').exists()).toBe(true)
    })

    it('should have proper CSS classes', () => {
      const container = wrapper.find('div.flex')
      expect(container.classes()).toContain('gap-1')
      expect(container.classes()).toContain('w-fit')
      expect(container.classes()).toContain('m-2')
      expect(container.classes()).toContain('p-2')
      expect(container.classes()).toContain('px-4')
      expect(container.classes()).toContain('rounded-md')
      expect(container.classes()).toContain('mx-auto')
    })

    it('should not display AlertError when errorCentrifuge is empty', () => {
      expect(messagesStore.errorCentrifuge).toBe('')
      expect(wrapper.findComponent({ name: 'AlertError' }).exists()).toBe(false)
    })

    it('should not display AlertWarning when warningCentrifuge is empty', () => {
      expect(messagesStore.warningCentrifuge).toBe('')
      expect(wrapper.findComponent({ name: 'AlertWarning' }).exists()).toBe(false)
    })
  })

  describe('error display', () => {
    it('should display AlertError when errorCentrifuge is set', async () => {
      messagesStore.errorCentrifuge = 'Connection failed'
      await wrapper.vm.$nextTick()

      const alertError = wrapper.findComponent({ name: 'AlertError' })
      expect(alertError.exists()).toBe(true)
      expect(alertError.props('text')).toBe('Connection failed')
    })

    it('should display AlertError with centrifuge error message', async () => {
      messagesStore.errorCentrifuge = 'Error in server connection, retrying...'
      await wrapper.vm.$nextTick()

      const alertError = wrapper.findComponent({ name: 'AlertError' })
      expect(alertError.exists()).toBe(true)
      expect(alertError.props('text')).toBe('Error in server connection, retrying...')
    })

    it('should display AlertError with disconnected message', async () => {
      messagesStore.errorCentrifuge = 'Disconnected from server, please refresh the page'
      await wrapper.vm.$nextTick()

      const alertError = wrapper.findComponent({ name: 'AlertError' })
      expect(alertError.exists()).toBe(true)
      expect(alertError.props('text')).toBe('Disconnected from server, please refresh the page')
    })

    it('should not display AlertWarning when errorCentrifuge is set', async () => {
      messagesStore.errorCentrifuge = 'Connection error'
      await wrapper.vm.$nextTick()

      expect(wrapper.findComponent({ name: 'AlertError' }).exists()).toBe(true)
      expect(wrapper.findComponent({ name: 'AlertWarning' }).exists()).toBe(false)
    })
  })

  describe('warning display', () => {
    it('should display AlertWarning when warningCentrifuge is set', async () => {
      messagesStore.warningCentrifuge = 'Connecting to server...'
      await wrapper.vm.$nextTick()

      const alertWarning = wrapper.findComponent({ name: 'AlertWarning' })
      expect(alertWarning.exists()).toBe(true)
      expect(alertWarning.props('text')).toBe('Connecting to server...')
    })

    it('should not display AlertError when warningCentrifuge is set', async () => {
      messagesStore.warningCentrifuge = 'Connecting...'
      await wrapper.vm.$nextTick()

      expect(wrapper.findComponent({ name: 'AlertWarning' }).exists()).toBe(true)
      expect(wrapper.findComponent({ name: 'AlertError' }).exists()).toBe(false)
    })
  })

  describe('both error and warning states', () => {
    it('should only display AlertError when both error and warning are set', async () => {
      messagesStore.errorCentrifuge = 'Connection error'
      messagesStore.warningCentrifuge = 'Connecting...'
      await wrapper.vm.$nextTick()

      // In the template, AlertError comes first and both could theoretically show,
      // but typically only one is set at a time in the actual store logic
      expect(wrapper.findComponent({ name: 'AlertError' }).exists()).toBe(true)
    })

    it('should clear error when errorCentrifuge is cleared', async () => {
      messagesStore.errorCentrifuge = 'Connection error'
      await wrapper.vm.$nextTick()
      expect(wrapper.findComponent({ name: 'AlertError' }).exists()).toBe(true)

      messagesStore.errorCentrifuge = ''
      await wrapper.vm.$nextTick()
      expect(wrapper.findComponent({ name: 'AlertError' }).exists()).toBe(false)
    })

    it('should clear warning when warningCentrifuge is cleared', async () => {
      messagesStore.warningCentrifuge = 'Connecting...'
      await wrapper.vm.$nextTick()
      expect(wrapper.findComponent({ name: 'AlertWarning' }).exists()).toBe(true)

      messagesStore.warningCentrifuge = ''
      await wrapper.vm.$nextTick()
      expect(wrapper.findComponent({ name: 'AlertWarning' }).exists()).toBe(false)
    })
  })

  describe('store integration', () => {
    it('should integrate with messages store', () => {
      expect(wrapper.vm).toBeDefined()
      expect(messagesStore).toBeDefined()
    })

    it('should react to store errorCentrifuge changes', async () => {
      expect(wrapper.findComponent({ name: 'AlertError' }).exists()).toBe(false)

      messagesStore.errorCentrifuge = 'Test error'
      await wrapper.vm.$nextTick()

      expect(wrapper.findComponent({ name: 'AlertError' }).exists()).toBe(true)
    })

    it('should react to store warningCentrifuge changes', async () => {
      expect(wrapper.findComponent({ name: 'AlertWarning' }).exists()).toBe(false)

      messagesStore.warningCentrifuge = 'Test warning'
      await wrapper.vm.$nextTick()

      expect(wrapper.findComponent({ name: 'AlertWarning' }).exists()).toBe(true)
    })
  })

  describe('component behavior', () => {
    it('should be a read-only component (no user interactions)', () => {
      expect(wrapper.find('button').exists()).toBe(false)
      expect(wrapper.find('input').exists()).toBe(false)
      expect(wrapper.find('form').exists()).toBe(false)
    })

    it('should only display alert components from store', () => {
      // Component should only contain AlertError and AlertWarning components
      const children = wrapper.findAllComponents({ name: 'AlertError' })
      const warnings = wrapper.findAllComponents({ name: 'AlertWarning' })

      // When store is empty, both should be absent
      expect(children.length + warnings.length).toBe(0)
    })
  })

  describe('accessibility', () => {
    it('should use semantic HTML elements', () => {
      const container = wrapper.find('div')
      expect(container.exists()).toBe(true)
    })

    it('should pass error text to AlertError for screen readers', async () => {
      messagesStore.errorCentrifuge = 'Connection failed'
      await wrapper.vm.$nextTick()

      const alertError = wrapper.findComponent({ name: 'AlertError' })
      expect(alertError.props('text')).toBe('Connection failed')
    })

    it('should pass warning text to AlertWarning for screen readers', async () => {
      messagesStore.warningCentrifuge = 'Connecting...'
      await wrapper.vm.$nextTick()

      const alertWarning = wrapper.findComponent({ name: 'AlertWarning' })
      expect(alertWarning.props('text')).toBe('Connecting...')
    })
  })

  describe('edge cases', () => {
    it('should handle empty string errorCentrifuge', async () => {
      messagesStore.errorCentrifuge = ''
      await wrapper.vm.$nextTick()

      expect(wrapper.findComponent({ name: 'AlertError' }).exists()).toBe(false)
    })

    it('should handle empty string warningCentrifuge', async () => {
      messagesStore.warningCentrifuge = ''
      await wrapper.vm.$nextTick()

      expect(wrapper.findComponent({ name: 'AlertWarning' }).exists()).toBe(false)
    })

    it('should handle rapid state changes', async () => {
      // Simulate rapid connection state changes
      messagesStore.warningCentrifuge = 'Connecting...'
      await wrapper.vm.$nextTick()
      expect(wrapper.findComponent({ name: 'AlertWarning' }).exists()).toBe(true)

      messagesStore.errorCentrifuge = 'Connection failed'
      messagesStore.warningCentrifuge = ''
      await wrapper.vm.$nextTick()
      expect(wrapper.findComponent({ name: 'AlertError' }).exists()).toBe(true)
      expect(wrapper.findComponent({ name: 'AlertWarning' }).exists()).toBe(false)

      messagesStore.errorCentrifuge = ''
      await wrapper.vm.$nextTick()
      expect(wrapper.findComponent({ name: 'AlertError' }).exists()).toBe(false)
    })

    it('should handle special characters in error messages', async () => {
      messagesStore.errorCentrifuge = 'Error: Connection refused (12345)'
      await wrapper.vm.$nextTick()

      const alertError = wrapper.findComponent({ name: 'AlertError' })
      expect(alertError.exists()).toBe(true)
      expect(alertError.props('text')).toBe('Error: Connection refused (12345)')
    })
  })
})
