import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import Equipments from '../Equipments.vue'
import { flushPromises } from '@/test/utils'
import { useEquipmentStore } from '@/stores/equipment'

const mockEquipments = [
  { id: 1, type: 22, deviceIp: '192.168.1.110', hostIp: '192.168.1.120', hostPort: 6006, ldcId: 1, enabled: true },
  { id: 2, type: 23, deviceIp: '192.168.1.111', hostIp: '192.168.1.121', hostPort: 6007, ldcId: 1, enabled: false },
  { id: 3, type: 24, deviceIp: '192.168.1.112', hostIp: '192.168.1.122', hostPort: 6008, ldcId: 2, enabled: true }
]

describe('Equipments Component', () => {
  let wrapper: VueWrapper<any>
  let equipmentStore: any

  beforeEach(async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    vi.clearAllMocks()

    // Get the store first and clear state before mounting the component
    equipmentStore = useEquipmentStore()
    equipmentStore.equipments = []
    equipmentStore.errorEquipments = ''

    wrapper = mount(Equipments, {
      global: {
        plugins: [pinia],
        stubs: {
          Equipment: true,
          PlusIcon: true
        }
      }
    })

    // Wait for any initial async operations (like getEquipments) to complete
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

    it('should display "Add Equipment" button', () => {
      const button = wrapper.find('button')
      expect(button.exists()).toBe(true)
      expect(button.text()).toContain('Add Equipment')
    })

    it('should not display Equipment components when store is empty', async () => {
      equipmentStore.equipments = []
      await wrapper.vm.$nextTick()

      const equipmentComponents = wrapper.findAllComponents({ name: 'Equipment' })
      expect(equipmentComponents.length).toBe(0)
    })
  })

  describe('equipment list display', () => {
    it('should render Equipment components for each equipment in store', async () => {
      equipmentStore.equipments = mockEquipments
      await wrapper.vm.$nextTick()

      const equipmentComponents = wrapper.findAllComponents({ name: 'Equipment' })
      expect(equipmentComponents.length).toBe(mockEquipments.length)
    })

    it('should pass correct props to each Equipment component', async () => {
      equipmentStore.equipments = mockEquipments
      await wrapper.vm.$nextTick()

      const equipmentComponents = wrapper.findAllComponents({ name: 'Equipment' })

      equipmentComponents.forEach((comp, index) => {
        expect(comp.props('equipment')).toEqual(mockEquipments[index])
      })
    })

    it('should react to store changes', async () => {
      // Start empty
      expect(wrapper.findAllComponents({ name: 'Equipment' }).length).toBe(0)

      // Add equipments
      equipmentStore.equipments = [mockEquipments[0]]
      await wrapper.vm.$nextTick()
      expect(wrapper.findAllComponents({ name: 'Equipment' }).length).toBe(1)

      // Add more
      equipmentStore.equipments = [...mockEquipments]
      await wrapper.vm.$nextTick()
      expect(wrapper.findAllComponents({ name: 'Equipment' }).length).toBe(3)
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

    it('should show new Equipment form when showNew is true', async () => {
      wrapper.vm.showNew = true
      await wrapper.vm.$nextTick()

      const equipmentComponents = wrapper.findAllComponents({ name: 'Equipment' })
      // Should show one Equipment component for the new form (no equipment prop)
      expect(equipmentComponents.length).toBe(1)
      expect(equipmentComponents[0].props('equipment')).toBeUndefined()
    })

    it('should show existing equipments plus new form when showNew is true', async () => {
      equipmentStore.equipments = mockEquipments
      await wrapper.vm.$nextTick()

      wrapper.vm.showNew = true
      await wrapper.vm.$nextTick()

      const equipmentComponents = wrapper.findAllComponents({ name: 'Equipment' })
      // Should show 3 existing + 1 new form = 4 total
      expect(equipmentComponents.length).toBe(4)
    })
  })

  describe('error handling', () => {
    it('should display AlertError when errorEquipments is set', async () => {
      equipmentStore.errorEquipments = 'Failed to load equipments'
      await wrapper.vm.$nextTick()

      const alertError = wrapper.findComponent({ name: 'AlertError' })
      expect(alertError.exists()).toBe(true)
      expect(alertError.props('text')).toContain('Error loading Equipments')
      expect(alertError.props('text')).toContain('Failed to load equipments')
    })

    it('should not display AlertError when errorEquipments is empty', async () => {
      equipmentStore.errorEquipments = ''
      await wrapper.vm.$nextTick()

      const alertError = wrapper.findComponent({ name: 'AlertError' })
      expect(alertError.exists()).toBe(false)
    })

    it('should display equipments even when there is an error', async () => {
      equipmentStore.equipments = [mockEquipments[0]]
      equipmentStore.errorEquipments = 'Partial error'
      await wrapper.vm.$nextTick()

      const alertError = wrapper.findComponent({ name: 'AlertError' })
      expect(alertError.exists()).toBe(true)

      const equipmentComponents = wrapper.findAllComponents({ name: 'Equipment' })
      expect(equipmentComponents.length).toBe(1)
    })
  })

  describe('new equipment form behavior', () => {
    it('should hide new form when created event is emitted', async () => {
      wrapper.vm.showNew = true
      await wrapper.vm.$nextTick()

      // Form should be visible
      let equipmentComponents = wrapper.findAllComponents({ name: 'Equipment' })
      expect(equipmentComponents.length).toBe(1)

      // Simulate created event
      const newFormComponent = equipmentComponents[0]
      await newFormComponent.vm.$emit('created')
      await wrapper.vm.$nextTick()

      // Form should be hidden, button should be visible
      expect(wrapper.vm.showNew).toBe(false)
      expect(wrapper.find('button').exists()).toBe(true)
    })

    it('should toggle showNew back to false when created event fires', async () => {
      wrapper.vm.showNew = true
      await wrapper.vm.$nextTick()

      const newFormComponent = wrapper.findAllComponents({ name: 'Equipment' })[0]

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
    it('should integrate with equipment store', () => {
      expect(wrapper.vm).toBeDefined()
      expect(equipmentStore).toBeDefined()
    })

    it('should react to equipment store changes', async () => {
      expect(wrapper.findAllComponents({ name: 'Equipment' }).length).toBe(0)

      equipmentStore.equipments = mockEquipments
      await wrapper.vm.$nextTick()

      expect(wrapper.findAllComponents({ name: 'Equipment' }).length).toBe(3)
    })

    it('should react to errorEquipments store changes', async () => {
      // Clear any error from initial async operations and wait for reactivity
      equipmentStore.errorEquipments = ''
      await flushPromises()

      expect(wrapper.findComponent({ name: 'AlertError' }).exists()).toBe(false)

      equipmentStore.errorEquipments = 'Test error'
      await wrapper.vm.$nextTick()

      expect(wrapper.findComponent({ name: 'AlertError' }).exists()).toBe(true)
    })
  })

  describe('edge cases', () => {
    it('should handle empty equipment list', async () => {
      equipmentStore.equipments = []
      await wrapper.vm.$nextTick()

      expect(wrapper.findAllComponents({ name: 'Equipment' }).length).toBe(0)
      expect(wrapper.find('button').exists()).toBe(true)
    })

    it('should handle single equipment', async () => {
      equipmentStore.equipments = [mockEquipments[0]]
      await wrapper.vm.$nextTick()

      const equipmentComponents = wrapper.findAllComponents({ name: 'Equipment' })
      expect(equipmentComponents.length).toBe(1)
    })

    it('should handle large number of equipments', async () => {
      const largeList = Array.from({ length: 50 }, (_, i) => ({
        id: i + 1,
        type: 22,
        deviceIp: `192.168.1.${100 + i}`,
        hostIp: `192.168.1.${200 + i}`,
        hostPort: 6000 + i,
        ldcId: 1,
        enabled: true
      }))

      equipmentStore.equipments = largeList
      await wrapper.vm.$nextTick()

      const equipmentComponents = wrapper.findAllComponents({ name: 'Equipment' })
      expect(equipmentComponents.length).toBe(50)
    })

    it('should handle rapid showNew toggles', async () => {
      const button = wrapper.find('button')

      for (let i = 0; i < 10; i++) {
        await button.trigger('click')
        expect(wrapper.vm.showNew).toBe(true)

        // Find and emit created event
        const newForm = wrapper.findAllComponents({ name: 'Equipment' }).find(
          comp => comp.props('equipment') === undefined
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
      expect(button.text()).toContain('Add Equipment')
    })

    it('should use semantic HTML elements', () => {
      expect(wrapper.find('div').exists()).toBe(true)
      expect(wrapper.find('button').exists()).toBe(true)
    })
  })
})
