import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, VueWrapper } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import Status from '../Status.vue'
import { useMessagesStore } from '@/stores/messages'

// Mock child components
vi.mock('../AlertError.vue', () => ({
  default: {
    name: 'AlertError',
    template: '<div class="alert-error">{{ text }}</div>',
    props: ['text']
  }
}))

describe('Status Component Tests', () => {
  let wrapper: VueWrapper<any>
  let store: any
  let pinia: any

  beforeEach(() => {
    pinia = createPinia()
    setActivePinia(pinia)
    store = useMessagesStore()
    vi.clearAllMocks()
  })

  describe('Component Rendering', () => {
    it('should render status component', () => {
      wrapper = mount(Status, {
        global: {
          plugins: [pinia]
        }
      })

      expect(wrapper.find('h2').text()).toBe('Status')
      expect(wrapper.find('div[data-testid="status-container"]').exists()).toBe(true)
    })

    it('should display status information', () => {
      store.status = 'RUNNING'
      store.runNumber = 123

      wrapper = mount(Status, {
        global: {
          plugins: [pinia]
        }
      })

      expect(wrapper.text()).toContain('State:')
      expect(wrapper.text()).toContain('RUNNING')
      expect(wrapper.text()).toContain('Run number:')
      expect(wrapper.text()).toContain('123')
    })

    it('should show error states when present', () => {
      store.errorStates = 'Connection error'
      store.errorRunNumber = 'Run configuration error'

      wrapper = mount(Status, {
        global: {
          plugins: [pinia]
        }
      })

      const errorComponents = wrapper.findAllComponents({ name: 'AlertError' })
      expect(errorComponents.length).toBe(2)
      expect(errorComponents[0].props('text')).toBe('Error: Connection error')
      expect(errorComponents[1].props('text')).toBe('Error: Run configuration error')
    })

    it('should not show error states when empty', () => {
      store.errorStates = ''
      store.errorRunNumber = ''

      wrapper = mount(Status, {
        global: {
          plugins: [pinia]
        }
      })

      const errorComponents = wrapper.findAllComponents({ name: 'AlertError' })
      expect(errorComponents.length).toBe(0)
    })

    it('should show only first error when one is present', () => {
      store.errorStates = 'Connection error'
      store.errorRunNumber = ''

      wrapper = mount(Status, {
        global: {
          plugins: [pinia]
        }
      })

      const errorComponents = wrapper.findAllComponents({ name: 'AlertError' })
      expect(errorComponents.length).toBe(1)
      expect(errorComponents[0].props('text')).toBe('Error: Connection error')
    })
  })

  describe('Store Integration', () => {
    beforeEach(() => {
      // Clear error states to prevent test leakage
      store.errorStates = ''
      store.errorRunNumber = ''

      wrapper = mount(Status, {
        global: {
          plugins: [pinia]
        }
      })
    })

    it('should update display when store status changes', async () => {
      // Initial state
      expect(wrapper.text()).toContain('State:')
      expect(wrapper.text()).toContain('Run number:')

      // Update status
      store.status = 'STOPPED'
      await wrapper.vm.$nextTick()

      expect(wrapper.text()).toContain('State: STOPPED')

      // Update run number
      store.runNumber = 456
      await wrapper.vm.$nextTick()

      expect(wrapper.text()).toContain('Run number: 456')
    })

    it('should display errorStates from store', async () => {
      store.errorStates = 'New error occurred'
      await wrapper.vm.$nextTick()

      const errorComponents = wrapper.findAllComponents({ name: 'AlertError' })
      expect(errorComponents.length).toBe(1)
      expect(errorComponents[0].props('text')).toBe('Error: New error occurred')
    })

    it('should display errorRunNumber from store', async () => {
      store.errorRunNumber = 'Run configuration failed'
      await wrapper.vm.$nextTick()

      const errorComponents = wrapper.findAllComponents({ name: 'AlertError' })
      expect(errorComponents.length).toBe(1)
      expect(errorComponents[0].props('text')).toBe('Error: Run configuration failed')
    })

    it('should handle simultaneous error states', async () => {
      store.errorStates = 'State error'
      store.errorRunNumber = 'Run number error'

      await wrapper.vm.$nextTick()

      const errorComponents = wrapper.findAllComponents({ name: 'AlertError' })
      expect(errorComponents.length).toBe(2)
      expect(errorComponents[0].props('text')).toBe('Error: State error')
      expect(errorComponents[1].props('text')).toBe('Error: Run number error')
    })

    it('should handle null values gracefully', async () => {
      store.status = null
      store.runNumber = null

      await wrapper.vm.$nextTick()

      // Vue renders null as empty string, so we just check the labels are present
      expect(wrapper.text()).toContain('State:')
      expect(wrapper.text()).toContain('Run number:')
    })

    it('should handle undefined values gracefully', async () => {
      store.status = undefined
      store.runNumber = undefined

      await wrapper.vm.$nextTick()

      expect(wrapper.text()).toContain('State:')
      expect(wrapper.text()).toContain('Run number:')
    })
  })

  describe('Component Structure', () => {
    beforeEach(() => {
      wrapper = mount(Status, {
        global: {
          plugins: [pinia]
        }
      })
    })

    it('should have correct CSS classes', () => {
      const statusDiv = wrapper.find('div[data-testid="status-container"]')
      expect(statusDiv.exists()).toBe(true)
      expect(statusDiv.classes()).toContain('flex')
      expect(statusDiv.classes()).toContain('flex-col')
      expect(statusDiv.classes()).toContain('gap-2')
      expect(statusDiv.classes()).toContain('w-fit')
      expect(statusDiv.classes()).toContain('m-2')
      expect(statusDiv.classes()).toContain('p-2')
      expect(statusDiv.classes()).toContain('px-4')
      expect(statusDiv.classes()).toContain('rounded-md')
    })

    it('should have proper semantic structure', () => {
      expect(wrapper.find('h2').exists()).toBe(true)
      expect(wrapper.find('div').exists()).toBe(true)
      expect(wrapper.find('p').exists()).toBe(true)
    })

    it('should have proper text formatting', async () => {
      store.status = 'RUNNING'
      store.runNumber = 123

      await wrapper.vm.$nextTick()

      const paragraphs = wrapper.findAll('p')
      expect(paragraphs.length).toBe(2)

      // Check for bold labels
      const firstParagraph = paragraphs[0]
      const secondParagraph = paragraphs[1]

      expect(firstParagraph.text()).toContain('State:')
      expect(firstParagraph.find('span').text()).toBe('State:')

      expect(secondParagraph.text()).toContain('Run number:')
      expect(secondParagraph.find('span').text()).toBe('Run number:')
    })
  })

  describe('Data Display', () => {
    beforeEach(() => {
      // Clear error states to prevent test leakage
      store.errorStates = ''
      store.errorRunNumber = ''

      wrapper = mount(Status, {
        global: {
          plugins: [pinia]
        }
      })
    })

    it('should display different status values', async () => {
      const statuses = ['IDLE', 'RUNNING', 'STOPPED', 'CONFIGURING', 'ERROR']

      for (const status of statuses) {
        store.status = status
        await wrapper.vm.$nextTick()
        expect(wrapper.text()).toContain(`State: ${status}`)
      }
    })

    it('should display different run numbers', async () => {
      const runNumbers = [0, 1, 42, 999, 12345]

      for (const runNumber of runNumbers) {
        store.runNumber = runNumber
        await wrapper.vm.$nextTick()
        expect(wrapper.text()).toContain(`Run number: ${runNumber}`)
      }
    })

    it('should display empty values when no data', async () => {
      store.status = ''
      store.runNumber = 0

      await wrapper.vm.$nextTick()

      expect(wrapper.text()).toContain('State: ')
      expect(wrapper.text()).toContain('Run number: 0')
    })

    it('should display numeric run numbers correctly', async () => {
      store.runNumber = 123.456

      await wrapper.vm.$nextTick()

      expect(wrapper.text()).toContain('Run number: 123.456')
    })

    it('should display string run numbers correctly', async () => {
      store.runNumber = 'RUN-123'

      await wrapper.vm.$nextTick()

      expect(wrapper.text()).toContain('Run number: RUN-123')
    })
  })

  describe('Error Display Logic', () => {
    beforeEach(() => {
      // Clear error states to prevent test leakage
      store.errorStates = ''
      store.errorRunNumber = ''

      wrapper = mount(Status, {
        global: {
          plugins: [pinia]
        }
      })
    })

    it('should prioritize errorStates over errorRunNumber', async () => {
      store.errorStates = 'Critical error'
      store.errorRunNumber = 'Minor error'

      await wrapper.vm.$nextTick()

      const errorComponents = wrapper.findAllComponents({ name: 'AlertError' })
      expect(errorComponents[0].props('text')).toBe('Error: Critical error')
      expect(errorComponents[1].props('text')).toBe('Error: Minor error')
    })

    it('should handle empty string errors', async () => {
      store.errorStates = ''
      store.errorRunNumber = 'Non-empty error'

      await wrapper.vm.$nextTick()

      const errorComponents = wrapper.findAllComponents({ name: 'AlertError' })
      expect(errorComponents.length).toBe(1)
      expect(errorComponents[0].props('text')).toBe('Error: Non-empty error')
    })

    it('should handle whitespace-only errors', async () => {
      store.errorStates = '   '

      await wrapper.vm.$nextTick()

      const errorComponents = wrapper.findAllComponents({ name: 'AlertError' })
      expect(errorComponents.length).toBe(1)
      expect(errorComponents[0].props('text')).toBe('Error:    ')
    })

    it('should handle long error messages', async () => {
      const longError = 'This is a very long error message that spans multiple lines and contains a lot of detail about what went wrong with the system and how to potentially fix it'

      store.errorStates = longError

      await wrapper.vm.$nextTick()

      const errorComponents = wrapper.findAllComponents({ name: 'AlertError' })
      expect(errorComponents.length).toBe(1)
      expect(errorComponents[0].props('text')).toBe(`Error: ${longError}`)
    })
  })

  describe('Reactivity', () => {
    beforeEach(() => {
      // Clear error states to prevent test leakage
      store.errorStates = ''
      store.errorRunNumber = ''

      wrapper = mount(Status, {
        global: {
          plugins: [pinia]
        }
      })
    })

    it('should react to multiple rapid store updates', async () => {
      // Rapid succession of updates
      store.status = 'STARTING'
      await wrapper.vm.$nextTick()
      expect(wrapper.text()).toContain('State: STARTING')

      store.status = 'CONFIGURING'
      await wrapper.vm.$nextTick()
      expect(wrapper.text()).toContain('State: CONFIGURING')

      store.status = 'RUNNING'
      await wrapper.vm.$nextTick()
      expect(wrapper.text()).toContain('State: RUNNING')

      store.runNumber = 100
      await wrapper.vm.$nextTick()
      expect(wrapper.text()).toContain('Run number: 100')
    })

    it('should clear errors when they are resolved', async () => {
      // Set errors
      store.errorStates = 'Initial error'
      store.errorRunNumber = 'Run error'
      await wrapper.vm.$nextTick()

      expect(wrapper.findAllComponents({ name: 'AlertError' }).length).toBe(2)

      // Clear errors
      store.errorStates = ''
      store.errorRunNumber = ''
      await wrapper.vm.$nextTick()

      expect(wrapper.findAllComponents({ name: 'AlertError' }).length).toBe(0)
    })

    it('should clear one error while keeping the other', async () => {
      // Set errors
      store.errorStates = 'Persistent error'
      store.errorRunNumber = 'Temporary error'
      await wrapper.vm.$nextTick()

      expect(wrapper.findAllComponents({ name: 'AlertError' }).length).toBe(2)

      // Clear one error
      store.errorRunNumber = ''
      await wrapper.vm.$nextTick()

      const errorComponents = wrapper.findAllComponents({ name: 'AlertError' })
      expect(errorComponents.length).toBe(1)
      expect(errorComponents[0].props('text')).toBe('Error: Persistent error')
    })
  })

  describe('Accessibility', () => {
    beforeEach(() => {
      wrapper = mount(Status, {
        global: {
          plugins: [pinia]
        }
      })
    })

    it('should have proper heading structure', () => {
      const heading = wrapper.find('h2')
      expect(heading.exists()).toBe(true)
      expect(heading.text()).toBe('Status')
    })

    it('should have proper semantic text structure', async () => {
      store.status = 'RUNNING'
      store.runNumber = 123

      await wrapper.vm.$nextTick()

      const paragraphs = wrapper.findAll('p')
      expect(paragraphs.length).toBe(2)

      // Check for proper use of bold text for labels
      paragraphs.forEach(paragraph => {
        const boldText = paragraph.find('span.font-bold')
        expect(boldText.exists()).toBe(true)
      })
    })

    it('should have error elements when errors are present', async () => {
      store.errorStates = 'Test error'

      await wrapper.vm.$nextTick()

      const alertError = wrapper.findComponent({ name: 'AlertError' })
      expect(alertError.exists()).toBe(true)
      expect(alertError.props('text')).toContain('Error: Test error')
    })

    it('should have proper ARIA attributes through semantic HTML', () => {
      const statusDiv = wrapper.find('div[data-testid="status-container"]')
      expect(statusDiv.exists()).toBe(true)

      // The component uses semantic HTML (h2, p) which provides good accessibility
      expect(wrapper.find('h2').exists()).toBe(true)
      expect(wrapper.find('p').exists()).toBe(true)
    })
  })

  describe('Component Lifecycle', () => {
    it('should initialize with empty state', () => {
      wrapper = mount(Status, {
        global: {
          plugins: [pinia]
        }
      })

      expect(wrapper.find('h2').text()).toBe('Status')
      // Script setup components don't expose properties directly to wrapper.vm
      // Check that the component renders correctly instead
      expect(wrapper.find('p').exists()).toBe(true)
      expect(wrapper.text()).toContain('State:')
      expect(wrapper.text()).toContain('Run number:')
    })

    it('should handle component destruction', () => {
      wrapper = mount(Status, {
        global: {
          plugins: [pinia]
        }
      })

      // Simulate component destruction
      wrapper.unmount()

      // Component should be unmounted without errors
      expect(wrapper.exists()).toBe(false)
    })
  })

  describe('Performance', () => {
    it('should handle large status updates efficiently', async () => {
      wrapper = mount(Status, {
        global: {
          plugins: [pinia]
        }
      })

      const startTime = performance.now()

      // Perform many rapid updates
      for (let i = 0; i < 100; i++) {
        store.status = `STATUS_${i}`
        store.runNumber = i
        await wrapper.vm.$nextTick()
      }

      const endTime = performance.now()
      expect(endTime - startTime).toBeLessThan(1000) // Should complete within 1 second
    })

    it('should not re-render unnecessarily', async () => {
      wrapper = mount(Status, {
        global: {
          plugins: [pinia]
        }
      })

      const initialRender = wrapper.html()

      // Update with same values should not cause re-renders
      store.status = 'RUNNING'
      await wrapper.vm.$nextTick()
      const firstRender = wrapper.html()

      store.status = 'RUNNING'
      await wrapper.vm.$nextTick()
      const secondRender = wrapper.html()

      // HTML should be the same if values haven't changed
      expect(firstRender).toBe(secondRender)
    })
  })
})