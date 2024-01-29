import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import LDCs from '../LDCs.vue'
import { flushPromises } from '@/test/utils'
import { useLDCStore } from '@/stores/ldc'

const mockLDCs = [
  { id: 1, name: 'ldc1', hostname: 'ldc-host1', ip: '192.168.1.150', grpcPort: 50053, prometheusPort: 9092, enabled: true, equipments: [] },
  { id: 2, name: 'ldc2', hostname: 'ldc-host2', ip: '192.168.1.151', grpcPort: 50054, prometheusPort: 9093, enabled: false, equipments: [] },
  { id: 3, name: 'ldc3', hostname: 'ldc-host3', ip: '192.168.1.152', grpcPort: 50055, prometheusPort: 9094, enabled: true, equipments: [] }
]

describe('LDCs Component', () => {
  let wrapper: VueWrapper<any>
  let ldcStore: any

  beforeEach(async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    vi.clearAllMocks()

    // Get the store first and clear state before mounting the component
    ldcStore = useLDCStore()
    ldcStore.ldcs = []
    ldcStore.errorLDCs = ''

    wrapper = mount(LDCs, {
      global: {
        plugins: [pinia],
        stubs: {
          LDC: true,
          PlusIcon: true
        }
      }
    })

    // Wait for any initial async operations (like getLDCs) to complete
    await flushPromises()
  })

  describe('rendering', () => {
    it('should render the component correctly', () => {
      expect(wrapper.find('div.bg-slate-300').exists()).toBe(true)
    })

    it('should have proper CSS classes', () => {
      const container = wrapper.find('div.bg-slate-300')
      expect(container.classes()).toContain('rounded-md')
      expect(container.classes()).toContain('p-2')
    })

    it('should display "Add LDC" button', () => {
      const button = wrapper.find('button')
      expect(button.exists()).toBe(true)
      expect(button.text()).toContain('Add LDC')
    })

    it('should not display LDC components when store is empty', async () => {
      ldcStore.ldcs = []
      await wrapper.vm.$nextTick()

      const ldcComponents = wrapper.findAllComponents({ name: 'LDC' })
      expect(ldcComponents.length).toBe(0)
    })
  })

  describe('LDC list display', () => {
    it('should render LDC components for each LDC in store', async () => {
      ldcStore.ldcs = mockLDCs
      await wrapper.vm.$nextTick()

      const ldcComponents = wrapper.findAllComponents({ name: 'LDC' })
      expect(ldcComponents.length).toBe(mockLDCs.length)
    })

    it('should pass correct props to each LDC component', async () => {
      ldcStore.ldcs = mockLDCs
      await wrapper.vm.$nextTick()

      const ldcComponents = wrapper.findAllComponents({ name: 'LDC' })

      ldcComponents.forEach((comp, index) => {
        expect(comp.props('ldc')).toEqual(mockLDCs[index])
      })
    })

    it('should react to store changes', async () => {
      // Start empty
      expect(wrapper.findAllComponents({ name: 'LDC' }).length).toBe(0)

      // Add LDCs
      ldcStore.ldcs = [mockLDCs[0]]
      await wrapper.vm.$nextTick()
      expect(wrapper.findAllComponents({ name: 'LDC' }).length).toBe(1)

      // Add more
      ldcStore.ldcs = [...mockLDCs]
      await wrapper.vm.$nextTick()
      expect(wrapper.findAllComponents({ name: 'LDC' }).length).toBe(3)
    })
  })

  describe('add button functionality', () => {
    it('should toggle showNew when button is clicked', async () => {
      const button = wrapper.find('button')

      expect(wrapper.vm.showNew).toBe(false)

      await button.trigger('click')
      expect(wrapper.vm.showNew).toBe(true)

      await button.trigger('click')
      expect(wrapper.vm.showNew).toBe(false)
    })

    it('should hide button when showNew is true', async () => {
      wrapper.vm.showNew = true
      await wrapper.vm.$nextTick()

      const button = wrapper.find('button')
      expect(button.exists()).toBe(false)
    })

    it('should show new LDC form when showNew is true', async () => {
      wrapper.vm.showNew = true
      await wrapper.vm.$nextTick()

      const ldcComponents = wrapper.findAllComponents({ name: 'LDC' })
      // Should show one LDC component for the new form (no ldc prop)
      expect(ldcComponents.length).toBe(1)
      expect(ldcComponents[0].props('ldc')).toBeUndefined()
    })

    it('should show existing LDCs plus new form when showNew is true', async () => {
      ldcStore.ldcs = mockLDCs
      await wrapper.vm.$nextTick()

      wrapper.vm.showNew = true
      await wrapper.vm.$nextTick()

      const ldcComponents = wrapper.findAllComponents({ name: 'LDC' })
      // Should show 3 existing + 1 new form = 4 total
      expect(ldcComponents.length).toBe(4)
    })
  })

  describe('error handling', () => {
    it('should display AlertError when errorLDCs is set', async () => {
      ldcStore.errorLDCs = 'Failed to load LDCs'
      await wrapper.vm.$nextTick()

      const alertError = wrapper.findComponent({ name: 'AlertError' })
      expect(alertError.exists()).toBe(true)
      expect(alertError.props('text')).toContain('Error loading LDCS')
      expect(alertError.props('text')).toContain('Failed to load LDCs')
    })

    it('should not display AlertError when errorLDCs is empty', async () => {
      ldcStore.errorLDCs = ''
      await wrapper.vm.$nextTick()

      const alertError = wrapper.findComponent({ name: 'AlertError' })
      expect(alertError.exists()).toBe(false)
    })

    it('should display LDCs even when there is an error', async () => {
      ldcStore.ldcs = [mockLDCs[0]]
      ldcStore.errorLDCs = 'Partial error'
      await wrapper.vm.$nextTick()

      const alertError = wrapper.findComponent({ name: 'AlertError' })
      expect(alertError.exists()).toBe(true)

      const ldcComponents = wrapper.findAllComponents({ name: 'LDC' })
      expect(ldcComponents.length).toBe(1)
    })
  })

  describe('new LDC form behavior', () => {
    it('should hide new form when created event is emitted', async () => {
      wrapper.vm.showNew = true
      await wrapper.vm.$nextTick()

      // Form should be visible
      let ldcComponents = wrapper.findAllComponents({ name: 'LDC' })
      expect(ldcComponents.length).toBe(1)

      // Simulate created event
      const newFormComponent = ldcComponents[0]
      await newFormComponent.vm.$emit('created')
      await wrapper.vm.$nextTick()

      // Form should be hidden, button should be visible
      expect(wrapper.vm.showNew).toBe(false)
      expect(wrapper.find('button').exists()).toBe(true)
    })

    it('should toggle showNew back to false when created event fires', async () => {
      wrapper.vm.showNew = true
      await wrapper.vm.$nextTick()

      const newFormComponent = wrapper.findAllComponents({ name: 'LDC' })[0]

      await newFormComponent.vm.$emit('created')
      await wrapper.vm.$nextTick()

      expect(wrapper.vm.showNew).toBe(false)
    })
  })

  describe('component structure', () => {
    it('should have proper button styling', () => {
      const button = wrapper.find('button')
      expect(button.classes()).toContain('btn')
      expect(button.classes()).toContain('btn-info')
      expect(button.classes()).toContain('w-auto')
      expect(button.classes()).toContain('text-xl')
    })

    it('should use semantic HTML structure', () => {
      expect(wrapper.find('div').exists()).toBe(true)
      expect(wrapper.find('button').exists()).toBe(true)
    })
  })

  describe('store integration', () => {
    it('should integrate with LDC store', () => {
      expect(wrapper.vm).toBeDefined()
      expect(ldcStore).toBeDefined()
    })

    it('should react to LDC store changes', async () => {
      expect(wrapper.findAllComponents({ name: 'LDC' }).length).toBe(0)

      ldcStore.ldcs = mockLDCs
      await wrapper.vm.$nextTick()

      expect(wrapper.findAllComponents({ name: 'LDC' }).length).toBe(3)
    })

    it('should react to errorLDCs store changes', async () => {
      // Clear any error from initial async operations and wait for reactivity
      ldcStore.errorLDCs = ''
      await flushPromises()

      expect(wrapper.findComponent({ name: 'AlertError' }).exists()).toBe(false)

      ldcStore.errorLDCs = 'Test error'
      await wrapper.vm.$nextTick()

      expect(wrapper.findComponent({ name: 'AlertError' }).exists()).toBe(true)
    })
  })

  describe('edge cases', () => {
    it('should handle empty LDC list', async () => {
      ldcStore.ldcs = []
      await wrapper.vm.$nextTick()

      expect(wrapper.findAllComponents({ name: 'LDC' }).length).toBe(0)
      expect(wrapper.find('button').exists()).toBe(true)
    })

    it('should handle single LDC', async () => {
      ldcStore.ldcs = [mockLDCs[0]]
      await wrapper.vm.$nextTick()

      const ldcComponents = wrapper.findAllComponents({ name: 'LDC' })
      expect(ldcComponents.length).toBe(1)
    })

    it('should handle large number of LDCs', async () => {
      const largeList = Array.from({ length: 50 }, (_, i) => ({
        id: i + 1,
        name: `ldc${i}`,
        hostname: `ldc-host${i}`,
        ip: `192.168.1.${150 + i}`,
        grpcPort: 50000 + i,
        prometheusPort: 9000 + i,
        enabled: true,
        equipments: []
      }))

      ldcStore.ldcs = largeList
      await wrapper.vm.$nextTick()

      const ldcComponents = wrapper.findAllComponents({ name: 'LDC' })
      expect(ldcComponents.length).toBe(50)
    })

    it('should handle rapid showNew toggles', async () => {
      const button = wrapper.find('button')

      for (let i = 0; i < 10; i++) {
        await button.trigger('click')
        expect(wrapper.vm.showNew).toBe(true)

        // Find and emit created event
        const newForm = wrapper.findAllComponents({ name: 'LDC' }).find(
          comp => comp.props('ldc') === undefined
        )
        if (newForm) {
          await newForm.vm.$emit('created')
          await wrapper.vm.$nextTick()
        }

        expect(wrapper.vm.showNew).toBe(false)
      }
    })
  })

  describe('accessibility', () => {
    it('should have accessible button text', () => {
      const button = wrapper.find('button')
      expect(button.text()).toContain('Add LDC')
    })

    it('should use semantic HTML elements', () => {
      expect(wrapper.find('div').exists()).toBe(true)
      expect(wrapper.find('button').exists()).toBe(true)
    })
  })
})
