import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import GDCs from '../GDCs.vue'
import { flushPromises } from '@/test/utils'
import { useGDCStore } from '@/stores/gdc'

const mockGDCs = [
  { id: 1, name: 'gdc1', hostname: 'gdc-host1', ip: '192.168.1.100', port: 6005, datapath: '/data/gdc', grpcPort: 50051, prometheusPort: 9090, enabled: true, writeOutput: true, decode: false, equipments: [] },
  { id: 2, name: 'gdc2', hostname: 'gdc-host2', ip: '192.168.1.101', port: 6006, datapath: '/data/gdc2', grpcPort: 50052, prometheusPort: 9091, enabled: false, writeOutput: false, decode: true, equipments: [] },
  { id: 3, name: 'gdc3', hostname: 'gdc-host3', ip: '192.168.1.102', port: 6007, datapath: '/data/gdc3', grpcPort: 50053, prometheusPort: 9092, enabled: true, writeOutput: true, decode: false, equipments: [] }
]

describe('GDCs Component', () => {
  let wrapper: VueWrapper<any>
  let gdcStore: any

  beforeEach(async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    vi.clearAllMocks()

    // Get the store first and clear state before mounting the component
    gdcStore = useGDCStore()
    gdcStore.gdcs = []
    gdcStore.errorGDCs = ''

    wrapper = mount(GDCs, {
      global: {
        plugins: [pinia],
        stubs: {
          GDC: true,
          PlusIcon: true
        }
      }
    })

    // Wait for any initial async operations (like getGDCs) to complete
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

    it('should display "Add GDC" button', () => {
      const button = wrapper.find('button')
      expect(button.exists()).toBe(true)
      expect(button.text()).toContain('Add GDC')
    })

    it('should not display GDC components when store is empty', async () => {
      gdcStore.gdcs = []
      await wrapper.vm.$nextTick()

      const gdcComponents = wrapper.findAllComponents({ name: 'GDC' })
      expect(gdcComponents.length).toBe(0)
    })
  })

  describe('GDC list display', () => {
    it('should render GDC components for each GDC in store', async () => {
      gdcStore.gdcs = mockGDCs
      await wrapper.vm.$nextTick()

      const gdcComponents = wrapper.findAllComponents({ name: 'GDC' })
      expect(gdcComponents.length).toBe(mockGDCs.length)
    })

    it('should pass correct props to each GDC component', async () => {
      gdcStore.gdcs = mockGDCs
      await wrapper.vm.$nextTick()

      const gdcComponents = wrapper.findAllComponents({ name: 'GDC' })

      gdcComponents.forEach((comp, index) => {
        expect(comp.props('gdc')).toEqual(mockGDCs[index])
      })
    })

    it('should react to store changes', async () => {
      // Start empty
      expect(wrapper.findAllComponents({ name: 'GDC' }).length).toBe(0)

      // Add GDCs
      gdcStore.gdcs = [mockGDCs[0]]
      await wrapper.vm.$nextTick()
      expect(wrapper.findAllComponents({ name: 'GDC' }).length).toBe(1)

      // Add more
      gdcStore.gdcs = [...mockGDCs]
      await wrapper.vm.$nextTick()
      expect(wrapper.findAllComponents({ name: 'GDC' }).length).toBe(3)
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

    it('should show new GDC form when showNew is true', async () => {
      wrapper.vm.showNew = true
      await wrapper.vm.$nextTick()

      const gdcComponents = wrapper.findAllComponents({ name: 'GDC' })
      // Should show one GDC component for the new form (no gdc prop)
      expect(gdcComponents.length).toBe(1)
      expect(gdcComponents[0].props('gdc')).toBeUndefined()
    })

    it('should show existing GDCs plus new form when showNew is true', async () => {
      gdcStore.gdcs = mockGDCs
      await wrapper.vm.$nextTick()

      wrapper.vm.showNew = true
      await wrapper.vm.$nextTick()

      const gdcComponents = wrapper.findAllComponents({ name: 'GDC' })
      // Should show 3 existing + 1 new form = 4 total
      expect(gdcComponents.length).toBe(4)
    })
  })

  describe('error handling', () => {
    it('should display AlertError when errorGDCs is set', async () => {
      gdcStore.errorGDCs = 'Failed to load GDCs'
      await wrapper.vm.$nextTick()

      const alertError = wrapper.findComponent({ name: 'AlertError' })
      expect(alertError.exists()).toBe(true)
      expect(alertError.props('text')).toContain('Error loading GDCS')
      expect(alertError.props('text')).toContain('Failed to load GDCs')
    })

    it('should not display AlertError when errorGDCs is empty', async () => {
      gdcStore.errorGDCs = ''
      await wrapper.vm.$nextTick()

      const alertError = wrapper.findComponent({ name: 'AlertError' })
      expect(alertError.exists()).toBe(false)
    })

    it('should display GDCs even when there is an error', async () => {
      gdcStore.gdcs = [mockGDCs[0]]
      gdcStore.errorGDCs = 'Partial error'
      await wrapper.vm.$nextTick()

      const alertError = wrapper.findComponent({ name: 'AlertError' })
      expect(alertError.exists()).toBe(true)

      const gdcComponents = wrapper.findAllComponents({ name: 'GDC' })
      expect(gdcComponents.length).toBe(1)
    })
  })

  describe('new GDC form behavior', () => {
    it('should hide new form when created event is emitted', async () => {
      wrapper.vm.showNew = true
      await wrapper.vm.$nextTick()

      // Form should be visible
      let gdcComponents = wrapper.findAllComponents({ name: 'GDC' })
      expect(gdcComponents.length).toBe(1)

      // Simulate created event
      const newFormComponent = gdcComponents[0]
      await newFormComponent.vm.$emit('created')
      await wrapper.vm.$nextTick()

      // Form should be hidden, button should be visible
      expect(wrapper.vm.showNew).toBe(false)
      expect(wrapper.find('button').exists()).toBe(true)
    })

    it('should toggle showNew back to false when created event fires', async () => {
      wrapper.vm.showNew = true
      await wrapper.vm.$nextTick()

      const newFormComponent = wrapper.findAllComponents({ name: 'GDC' })[0]

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
    it('should integrate with GDC store', () => {
      expect(wrapper.vm).toBeDefined()
      expect(gdcStore).toBeDefined()
    })

    it('should react to GDC store changes', async () => {
      expect(wrapper.findAllComponents({ name: 'GDC' }).length).toBe(0)

      gdcStore.gdcs = mockGDCs
      await wrapper.vm.$nextTick()

      expect(wrapper.findAllComponents({ name: 'GDC' }).length).toBe(3)
    })

    it('should react to errorGDCs store changes', async () => {
      // Clear any error from initial async operations and wait for reactivity
      gdcStore.errorGDCs = ''
      await flushPromises()

      expect(wrapper.findComponent({ name: 'AlertError' }).exists()).toBe(false)

      gdcStore.errorGDCs = 'Test error'
      await wrapper.vm.$nextTick()

      expect(wrapper.findComponent({ name: 'AlertError' }).exists()).toBe(true)
    })
  })

  describe('edge cases', () => {
    it('should handle empty GDC list', async () => {
      gdcStore.gdcs = []
      await wrapper.vm.$nextTick()

      expect(wrapper.findAllComponents({ name: 'GDC' }).length).toBe(0)
      expect(wrapper.find('button').exists()).toBe(true)
    })

    it('should handle single GDC', async () => {
      gdcStore.gdcs = [mockGDCs[0]]
      await wrapper.vm.$nextTick()

      const gdcComponents = wrapper.findAllComponents({ name: 'GDC' })
      expect(gdcComponents.length).toBe(1)
    })

    it('should handle large number of GDCs', async () => {
      const largeList = Array.from({ length: 50 }, (_, i) => ({
        id: i + 1,
        name: `gdc${i}`,
        hostname: `gdc-host${i}`,
        ip: `192.168.1.${100 + i}`,
        port: 6000 + i,
        datapath: `/data/gdc${i}`,
        grpcPort: 50000 + i,
        prometheusPort: 9000 + i,
        enabled: true,
        writeOutput: true,
        decode: false,
        equipments: []
      }))

      gdcStore.gdcs = largeList
      await wrapper.vm.$nextTick()

      const gdcComponents = wrapper.findAllComponents({ name: 'GDC' })
      expect(gdcComponents.length).toBe(50)
    })

    it('should handle rapid showNew toggles', async () => {
      const button = wrapper.find('button')

      for (let i = 0; i < 10; i++) {
        await button.trigger('click')
        expect(wrapper.vm.showNew).toBe(true)

        // Find and emit created event
        const newForm = wrapper.findAllComponents({ name: 'GDC' }).find(
          comp => comp.props('gdc') === undefined
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
      expect(button.text()).toContain('Add GDC')
    })

    it('should use semantic HTML elements', () => {
      expect(wrapper.find('div').exists()).toBe(true)
      expect(wrapper.find('button').exists()).toBe(true)
    })
  })
})
