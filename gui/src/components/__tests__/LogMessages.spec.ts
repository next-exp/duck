import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import LogMessages from '../LogMessages.vue'
import { useMessagesStore } from '@/stores/messages'

const mockMessage = {
  host: 'gdc1',
  timestamp: '2025-02-10T12:00:00.000Z',
  type: 'info',
  value: 'Test message',
  run: 123
}

const mockErrorMessage = {
  host: 'ldc1',
  timestamp: '2025-02-10T12:01:00.000Z',
  type: 'error',
  value: 'Error occurred',
  run: 123
}

const mockDebugMessage = {
  host: 'gdc2',
  timestamp: '2025-02-10T12:02:00.000Z',
  type: 'debug',
  value: 'Debug info',
  run: 123
}

describe('LogMessages Component', () => {
  let wrapper: VueWrapper<any>
  let messagesStore: any

  beforeEach(async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    vi.clearAllMocks()

    wrapper = mount(LogMessages, {
      global: {
        plugins: [pinia]
      }
    })

    messagesStore = useMessagesStore()
    await wrapper.vm.$nextTick()
  })

  describe('rendering - Messages section', () => {
    it('should render the component correctly', () => {
      expect(wrapper.find('h2').exists()).toBe(true)
      expect(wrapper.find('h2').text()).toBe('Messages')
    })

    it('should have proper CSS classes for Messages section', () => {
      const container = wrapper.find('.bg-slate-300')
      expect(container.exists()).toBe(true)
      expect(container.classes()).toContain('rounded-lg')
      expect(container.classes()).toContain('md:w-2/3')
    })

    it('should render messages container with overflow scroll', () => {
      const messagesContainer = wrapper.findAll('.bg-gray-200')[0]
      expect(messagesContainer.classes()).toContain('overflow-scroll')
    })

    it('should render all messages from store', async () => {
      messagesStore.messages = [mockMessage, mockErrorMessage]
      await wrapper.vm.$nextTick()

      const messageParagraphs = wrapper.findAll('.bg-gray-200')[0].findAll('p')
      expect(messageParagraphs.length).toBe(2)
    })

    it('should render messages in reverse order (newest first)', async () => {
      messagesStore.messages = [mockMessage, mockErrorMessage]
      await wrapper.vm.$nextTick()

      const messageParagraphs = wrapper.findAll('.bg-gray-200')[0].findAll('p')
      // Messages are reversed, so error message (later) should appear first
      expect(messageParagraphs[0].text()).toContain('Error occurred')
      expect(messageParagraphs[1].text()).toContain('Test message')
    })

    it('should display empty state when no messages exist', () => {
      const messagesContainer = wrapper.findAll('.bg-gray-200')[0]
      expect(messagesContainer.findAll('p').length).toBe(0)
    })
  })

  describe('error message styling', () => {
    it('should apply red color to error messages', async () => {
      messagesStore.messages = [mockMessage, mockErrorMessage]
      await wrapper.vm.$nextTick()

      const messageParagraphs = wrapper.findAll('.bg-gray-200')[0].findAll('p')
      const errorParagraph = messageParagraphs.find(p => p.text().includes('Error occurred'))
      expect(errorParagraph?.classes()).toContain('text-red-600')
    })

    it('should not apply red color to non-error messages', async () => {
      messagesStore.messages = [mockMessage]
      await wrapper.vm.$nextTick()

      const messageParagraphs = wrapper.findAll('.bg-gray-200')[0].findAll('p')
      const infoParagraph = messageParagraphs.find(p => p.text().includes('Test message'))
      expect(infoParagraph?.classes()).not.toContain('text-red-600')
    })

    it('should handle multiple error messages', async () => {
      messagesStore.messages = [
        { ...mockMessage, type: 'error', value: 'Error 1' },
        { ...mockMessage, type: 'error', value: 'Error 2', host: 'ldc1' },
        { ...mockMessage, type: 'info', value: 'Info message' }
      ]
      await wrapper.vm.$nextTick()

      const messageParagraphs = wrapper.findAll('.bg-gray-200')[0].findAll('p')
      const errorParagraphs = messageParagraphs.filter(p => p.classes().includes('text-red-600'))
      expect(errorParagraphs.length).toBe(2)
    })
  })

  describe('message formatting', () => {
    it('should format timestamp correctly', async () => {
      messagesStore.messages = [mockMessage]
      await wrapper.vm.$nextTick()

      const messageParagraphs = wrapper.findAll('.bg-gray-200')[0].findAll('p')
      expect(messageParagraphs[0].text()).toMatch(/\d{2}\/\d{2}\/\d{2} - \d{2}:\d{2}:\d{2}\.\d{3}/)
    })

    it('should include message type in display', async () => {
      messagesStore.messages = [mockMessage]
      await wrapper.vm.$nextTick()

      const messageParagraphs = wrapper.findAll('.bg-gray-200')[0].findAll('p')
      expect(messageParagraphs[0].text()).toContain('info')
    })

    it('should include host in display', async () => {
      messagesStore.messages = [mockMessage]
      await wrapper.vm.$nextTick()

      const messageParagraphs = wrapper.findAll('.bg-gray-200')[0].findAll('p')
      expect(messageParagraphs[0].text()).toContain('gdc1')
    })

    it('should include message value in display', async () => {
      messagesStore.messages = [mockMessage]
      await wrapper.vm.$nextTick()

      const messageParagraphs = wrapper.findAll('.bg-gray-200')[0].findAll('p')
      expect(messageParagraphs[0].text()).toContain('Test message')
    })

    it('should handle empty timestamp gracefully', async () => {
      messagesStore.messages = [{ ...mockMessage, timestamp: '' }]
      await wrapper.vm.$nextTick()

      const messageParagraphs = wrapper.findAll('.bg-gray-200')[0].findAll('p')
      expect(messageParagraphs[0].text()).toContain('—')
    })
  })

  describe('rendering - Debugging section', () => {
    it('should render Debugging section header', () => {
      const headers = wrapper.findAll('h2')
      expect(headers[1].text()).toBe('Debugging')
    })

    it('should have checkbox for showing debug messages', () => {
      const checkbox = wrapper.find('#show-debugging')
      expect(checkbox.exists()).toBe(true)
      expect(checkbox.attributes('type')).toBe('checkbox')
    })

    it('should have label for checkbox', () => {
      const label = wrapper.find('label[for="show-debugging"]')
      expect(label.exists()).toBe(true)
      expect(label.text()).toBe('Show messages')
    })

    it('should not show debug messages when checkbox is unchecked', async () => {
      messagesStore.messagesDebug = [mockDebugMessage]
      await wrapper.vm.$nextTick()

      expect(wrapper.vm.showDebug).toBe(false)
      const containers = wrapper.findAll('.bg-gray-200')
      expect(containers.length).toBe(1) // Only messages container, no debug container
    })

    it('should show debug messages when checkbox is checked', async () => {
      messagesStore.messagesDebug = [mockDebugMessage]
      await wrapper.vm.$nextTick()

      const checkbox = wrapper.find('#show-debugging')
      await checkbox.setChecked(true)
      await wrapper.vm.$nextTick()

      expect(wrapper.vm.showDebug).toBe(true)
      const debugContainer = wrapper.findAll('.bg-gray-200')[1]
      expect(debugContainer.exists()).toBe(true)
    })
  })

  describe('debug messages display', () => {
    it('should render debug messages in reverse order', async () => {
      messagesStore.messagesDebug = [
        { ...mockDebugMessage, value: 'First' },
        { ...mockDebugMessage, value: 'Second', host: 'ldc1' }
      ]

      const checkbox = wrapper.find('#show-debugging')
      await checkbox.setChecked(true)
      await wrapper.vm.$nextTick()

      const debugParagraphs = wrapper.findAll('.bg-gray-200')[1].findAll('p')
      expect(debugParagraphs[0].text()).toContain('Second')
      expect(debugParagraphs[1].text()).toContain('First')
    })

    it('should format debug messages with timestamp, host, and value', async () => {
      messagesStore.messagesDebug = [mockDebugMessage]

      const checkbox = wrapper.find('#show-debugging')
      await checkbox.setChecked(true)
      await wrapper.vm.$nextTick()

      const debugParagraphs = wrapper.findAll('.bg-gray-200')[1].findAll('p')
      expect(debugParagraphs[0].text()).toContain('gdc2')
      expect(debugParagraphs[0].text()).toContain('Debug info')
    })

    it('should display empty state when no debug messages exist', async () => {
      const checkbox = wrapper.find('#show-debugging')
      await checkbox.setChecked(true)
      await wrapper.vm.$nextTick()

      const debugParagraphs = wrapper.findAll('.bg-gray-200')[1].findAll('p')
      expect(debugParagraphs.length).toBe(0)
    })

    it('should hide debug messages when checkbox is unchecked', async () => {
      messagesStore.messagesDebug = [mockDebugMessage]

      const checkbox = wrapper.find('#show-debugging')
      await checkbox.setChecked(true)
      await wrapper.vm.$nextTick()

      let containers = wrapper.findAll('.bg-gray-200')
      expect(containers.length).toBe(2) // Messages + Debug

      await checkbox.setChecked(false)
      await wrapper.vm.$nextTick()

      containers = wrapper.findAll('.bg-gray-200')
      expect(containers.length).toBe(1) // Only messages container
    })
  })

  describe('store integration', () => {
    it('should integrate with messages store', () => {
      expect(wrapper.vm).toBeDefined()
      expect(messagesStore).toBeDefined()
    })

    it('should react to store messages changes', async () => {
      expect(wrapper.findAll('.bg-gray-200')[0].findAll('p').length).toBe(0)

      messagesStore.messages = [mockMessage, mockErrorMessage]
      await wrapper.vm.$nextTick()

      expect(wrapper.findAll('.bg-gray-200')[0].findAll('p').length).toBe(2)
    })

    it('should react to store messagesDebug changes', async () => {
      const checkbox = wrapper.find('#show-debugging')
      await checkbox.setChecked(true)
      await wrapper.vm.$nextTick()

      expect(wrapper.findAll('.bg-gray-200')[1].findAll('p').length).toBe(0)

      messagesStore.messagesDebug = [mockDebugMessage]
      await wrapper.vm.$nextTick()

      expect(wrapper.findAll('.bg-gray-200')[1].findAll('p').length).toBe(1)
    })
  })

  describe('accessibility', () => {
    it('should use semantic HTML elements', () => {
      expect(wrapper.find('h2').exists()).toBe(true)
      expect(wrapper.find('div').exists()).toBe(true)
    })

    it('should have proper label association with checkbox', () => {
      const label = wrapper.find('label[for="show-debugging"]')
      const checkbox = wrapper.find('#show-debugging')
      expect(label.exists()).toBe(true)
      expect(checkbox.exists()).toBe(true)
    })

    it('should have proper for attribute on label', () => {
      const label = wrapper.find('label[for="show-debugging"]')
      expect(label.attributes('for')).toBe('show-debugging')
    })
  })

  describe('component behavior', () => {
    it('should initialize showDebug to false', () => {
      expect(wrapper.vm.showDebug).toBe(false)
    })

    it('should toggle showDebug when checkbox is clicked', async () => {
      const checkbox = wrapper.find('#show-debugging')

      await checkbox.setChecked(true)
      expect(wrapper.vm.showDebug).toBe(true)

      await checkbox.setChecked(false)
      expect(wrapper.vm.showDebug).toBe(false)
    })

    it('should handle whitespace-pre-wrap for message formatting', async () => {
      messagesStore.messages = [{ ...mockMessage, value: 'Line 1\nLine 2' }]
      await wrapper.vm.$nextTick()

      const messageParagraphs = wrapper.findAll('.bg-gray-200')[0].findAll('p')
      expect(messageParagraphs[0].classes()).toContain('whitespace-pre-wrap')
    })
  })

  describe('edge cases', () => {
    it('should handle large number of messages', async () => {
      const largeMessageList = Array.from({ length: 200 }, (_, i) => ({
        ...mockMessage,
        value: `Message ${i}`,
        host: `host${i}`
      }))

      messagesStore.messages = largeMessageList
      await wrapper.vm.$nextTick()

      const messageParagraphs = wrapper.findAll('.bg-gray-200')[0].findAll('p')
      expect(messageParagraphs.length).toBe(200)
    })

    it('should handle messages with special characters', async () => {
      messagesStore.messages = [{
        ...mockMessage,
        value: 'Message with <special> & "characters"'
      }]
      await wrapper.vm.$nextTick()

      const messageParagraphs = wrapper.findAll('.bg-gray-200')[0].findAll('p')
      expect(messageParagraphs[0].text()).toContain('Message with <special> & "characters"')
    })

    it('should handle messages with newlines', async () => {
      messagesStore.messages = [{
        ...mockMessage,
        value: 'Line 1\nLine 2\nLine 3'
      }]
      await wrapper.vm.$nextTick()

      const messageParagraphs = wrapper.findAll('.bg-gray-200')[0].findAll('p')
      expect(messageParagraphs[0].text()).toContain('Line 1')
      expect(messageParagraphs[0].text()).toContain('Line 2')
      expect(messageParagraphs[0].text()).toContain('Line 3')
    })

    it('should handle rapid message updates', async () => {
      messagesStore.messages = [mockMessage]
      await wrapper.vm.$nextTick()

      messagesStore.messages = [mockMessage, mockErrorMessage]
      await wrapper.vm.$nextTick()

      messagesStore.messages = []
      await wrapper.vm.$nextTick()

      const messageParagraphs = wrapper.findAll('.bg-gray-200')[0].findAll('p')
      expect(messageParagraphs.length).toBe(0)
    })
  })
})
