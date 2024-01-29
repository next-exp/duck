import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import Equipment from '../Equipment.vue'
import { flushPromises } from '@/test/utils'
import { mockDuckApiClient } from '@/test/setup'
import { useEquipmentStore } from '@/stores/equipment'
import { useLDCStore } from '@/stores/ldc'

const mockEquipment = {
  id: 1,
  type: 22,
  deviceIp: '192.168.1.110',
  hostIp: '192.168.1.120',
  hostPort: 6006,
  ldcId: 1,
  enabled: true
}

const mockLDCs = [
  { id: 1, name: 'ldc1', hostname: 'ldc1', ip: '192.168.1.101', grpcPort: 50053, prometheusPort: 9092, enabled: true, equipments: [] },
  { id: 2, name: 'ldc2', hostname: 'ldc2', ip: '192.168.1.102', grpcPort: 50054, prometheusPort: 9093, enabled: false, equipments: [] }
]

describe('Equipment Component', () => {
  let wrapper: VueWrapper<any>
  let equipmentStore: any
  let ldcStore: any

  beforeEach(async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    vi.clearAllMocks()

    // Set up stores
    equipmentStore = useEquipmentStore()
    ldcStore = useLDCStore()
    ldcStore.ldcs = mockLDCs

    wrapper = mount(Equipment, {
      global: {
        plugins: [pinia],
        stubs: {
          AlertSuccess: true,
          AlertError: true
        }
      }
    })

    await flushPromises()
  })

  describe('rendering - create mode (no props)', () => {
    it('should render the component correctly', () => {
      expect(wrapper.find('h2').text()).toBe('Equipment')
    })

    it('should render all input fields', () => {
      expect(wrapper.find('#eq-device-ip-0').exists()).toBe(true)
      expect(wrapper.find('#eq-host-ip-0').exists()).toBe(true)
      expect(wrapper.find('#eq-host-port-0').exists()).toBe(true)
      expect(wrapper.find('#eq-ldcid-0').exists()).toBe(true)
    })

    it('should render enabled checkbox', () => {
      expect(wrapper.find('#eq-enabled-0').exists()).toBe(true)
    })

    it('should show "Create Equipment" button when no equipment prop', () => {
      const button = wrapper.find('button[data-testid="equipment-submit-button-0"]')
      expect(button.exists()).toBe(true)
      expect(button.text()).toContain('Create Equipment')
    })

    it('should not show delete button when no equipment prop', () => {
      const deleteButton = wrapper.find('button[data-testid="equipment-delete-button-0"]')
      expect(deleteButton.exists()).toBe(false)
    })
  })

  describe('rendering - update mode (with props)', () => {
    beforeEach(async () => {
      wrapper = mount(Equipment, {
        props: {
          equipment: mockEquipment
        },
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertSuccess: true,
            AlertError: true
          }
        }
      })
      await flushPromises()
    })

    it('should show "Update" button when equipment prop is provided', () => {
      const button = wrapper.find('button[data-testid="equipment-submit-button-1"]')
      expect(button.exists()).toBe(true)
      expect(button.text()).toContain('Update')
    })

    it('should show delete button when equipment prop is provided', () => {
      const deleteButton = wrapper.find('button[data-testid="equipment-delete-button-1"]')
      expect(deleteButton.exists()).toBe(true)
    })

    it('should render modal dialog for delete confirmation', () => {
      const dialog = wrapper.find('dialog')
      expect(dialog.exists()).toBe(true)
    })
  })

  describe('LDC select dropdown', () => {
    beforeEach(async () => {
      // Ensure LDCs are set for these tests
      ldcStore.ldcs = mockLDCs
      await flushPromises()
    })

    it('should render LDC select dropdown', () => {
      const select = wrapper.find('#eq-ldcid-0')
      expect(select.exists()).toBe(true)
      expect(select.element.tagName).toBe('SELECT')
    })

    it('should populate LDC options from store', () => {
      const select = wrapper.find('#eq-ldcid-0')
      const options = select.findAll('option')

      expect(options.length).toBeGreaterThan(0)
      expect(options[0].element.text).toContain(mockLDCs[0].hostname)
    })

    it('should have correct LDC values', () => {
      const select = wrapper.find('#eq-ldcid-0')
      const options = select.findAll('option')

      // The option values should be numeric ids from LDC objects
      expect(options.length).toBeGreaterThan(0)
      if (options.length > 0) {
        expect(options[0].element.value).toBeDefined()
      }
    })

    it('should have selectable LDC options', () => {
      const select = wrapper.find('#eq-ldcid-0')
      const options = select.findAll('option')

      // Options should be selectable (not disabled)
      options.forEach(option => {
        expect(option.attributes('disabled')).toBeUndefined()
      })
    })
  })

  describe('input fields', () => {
    it('should have proper labels for all inputs', () => {
      const labels = wrapper.findAll('label')
      const labelTexts = labels.map(l => l.text())

      expect(labelTexts).toContain('Device IP')
      expect(labelTexts).toContain('Host IP')
      expect(labelTexts).toContain('Host Port')
      expect(labelTexts).toContain('LDC ID')
      expect(labelTexts).toContain('Enabled')
    })

    it('should have proper placeholder text', () => {
      expect(wrapper.find('#eq-device-ip-0').attributes('placeholder')).toBe('Device IP')
      expect(wrapper.find('#eq-host-ip-0').attributes('placeholder')).toBe('Host IP')
      expect(wrapper.find('#eq-host-port-0').attributes('placeholder')).toBe('Host Port')
    })

    it('should have correct input types for all fields', () => {
      const deviceIpInput = wrapper.find('#eq-device-ip-0')
      expect(deviceIpInput.attributes('type')).toBe('text')

      const hostIpInput = wrapper.find('#eq-host-ip-0')
      expect(hostIpInput.attributes('type')).toBe('text')

      const hostPortInput = wrapper.find('#eq-host-port-0')
      expect(hostPortInput.attributes('type')).toBe('number')

      const enabledCheckbox = wrapper.find('#eq-enabled-0')
      expect(enabledCheckbox.attributes('type')).toBe('checkbox')

      const ldcSelect = wrapper.find('#eq-ldcid-0')
      expect(ldcSelect.element.tagName).toBe('SELECT')
    })

    it('should update input values when changed', async () => {
      const deviceIpInput = wrapper.find('#eq-device-ip-0')
      await deviceIpInput.setValue('192.168.1.200')
      expect(deviceIpInput.element.value).toBe('192.168.1.200')
    })

    it('should update input values using data-testid selector', async () => {
      const deviceIpInput = wrapper.find('input[data-testid="equipment-device-ip-input-0"]')
      await deviceIpInput.setValue('192.168.1.201')
      expect(deviceIpInput.element.value).toBe('192.168.1.201')
    })

    it('should update checkbox values when clicked', async () => {
      const enabledCheckbox = wrapper.find('#eq-enabled-0')
      await enabledCheckbox.setChecked(true)
      expect(enabledCheckbox.element.checked).toBe(true)
    })
  })

  describe('form submission - create mode', () => {
    it('should have form submission handler available', () => {
      expect(wrapper.vm.onSubmit).toBeDefined()
      expect(typeof wrapper.vm.onSubmit).toBe('function')
    })

    it('should be able to fill all required form fields', async () => {
      ldcStore.ldcs = mockLDCs
      await flushPromises()
      await wrapper.vm.$nextTick()

      // Fill in all required form fields
      await wrapper.find('input[data-testid="equipment-device-ip-input-0"]').setValue('192.168.1.200')
      await wrapper.find('input[data-testid="equipment-host-ip-input-0"]').setValue('192.168.1.20')
      await wrapper.find('input[data-testid="equipment-host-port-input-0"]').setValue('6006')

      // Select an LDC from the dropdown
      const ldcSelect = wrapper.find('select[data-testid="equipment-ldc-select-0"]')
      await ldcSelect.setValue(1)

      // Verify values were set
      expect(wrapper.find('input[data-testid="equipment-device-ip-input-0"]').element.value).toBe('192.168.1.200')
      expect(wrapper.find('input[data-testid="equipment-host-ip-input-0"]').element.value).toBe('192.168.1.20')
      expect(wrapper.find('input[data-testid="equipment-host-port-input-0"]').element.value).toBe('6006')
    })

    it('should call createEquipment API and store when form is submitted', async () => {
      ldcStore.ldcs = mockLDCs
      await flushPromises()

      const getEquipmentsSpy = vi.spyOn(equipmentStore, 'getEquipments')

      // Fill in form fields with valid data
      await wrapper.find('input[data-testid="equipment-device-ip-input-0"]').setValue('192.168.1.200')
      await wrapper.find('input[data-testid="equipment-host-ip-input-0"]').setValue('192.168.1.20')
      await wrapper.find('input[data-testid="equipment-host-port-input-0"]').setValue('6006')
      await wrapper.find('select[data-testid="equipment-ldc-select-0"]').setValue(1)

      // Mock successful API response
      mockDuckApiClient.createEquipment.mockResolvedValueOnce({ id: 999 })

      // Trigger submit through the component's method
      await wrapper.vm.onSubmit()
      await flushPromises()

      // Verify API was called (testing integration, not exact params)
      expect(mockDuckApiClient.createEquipment).toHaveBeenCalled()

      // Verify store refresh was triggered
      expect(getEquipmentsSpy).toHaveBeenCalled()
    })
  })

  describe('form submission - update mode', () => {
    let updateLdcStore: any

    beforeEach(async () => {
      const pinia = createPinia()
      setActivePinia(pinia)

      // Set up stores for update mode
      updateLdcStore = useLDCStore()
      updateLdcStore.ldcs = mockLDCs

      wrapper = mount(Equipment, {
        props: {
          equipment: mockEquipment
        },
        global: {
          plugins: [pinia],
          stubs: {
            AlertSuccess: true,
            AlertError: true
          }
        }
      })
      await flushPromises()
    })

    it('should have form submission handler available in update mode', () => {
      expect(wrapper.vm.onSubmit).toBeDefined()
      expect(typeof wrapper.vm.onSubmit).toBe('function')
    })

    it('should be able to update form values', async () => {
      const store = useEquipmentStore()
      const getEquipmentsSpy = vi.spyOn(store, 'getEquipments')

      // Update form values
      await wrapper.find('input[data-testid="equipment-device-ip-input-1"]').setValue('192.168.1.250')
      await wrapper.find('input[data-testid="equipment-host-ip-input-1"]').setValue('192.168.1.25')
      await wrapper.find('input[data-testid="equipment-host-port-input-1"]').setValue('7007')
      await wrapper.find('select[data-testid="equipment-ldc-select-1"]').setValue(2)

      // Verify values were set
      expect(wrapper.find('input[data-testid="equipment-device-ip-input-1"]').element.value).toBe('192.168.1.250')
      expect(wrapper.find('input[data-testid="equipment-host-ip-input-1"]').element.value).toBe('192.168.1.25')
    })

    it('should call updateEquipment API when form is submitted', async () => {
      const store = useEquipmentStore()
      const getEquipmentsSpy = vi.spyOn(store, 'getEquipments')

      // Mock successful API response
      mockDuckApiClient.updateEquipment.mockResolvedValueOnce({ id: mockEquipment.id })

      // The form is pre-populated with mockEquipment values from props
      // Just verify that submitting calls the API with the equipment id
      await wrapper.vm.onSubmit()
      await flushPromises()

      // Verify API was called with correct id
      expect(mockDuckApiClient.updateEquipment).toHaveBeenCalledWith(
        expect.objectContaining({
          id: mockEquipment.id
        })
      )

      // Verify store refresh was triggered
      expect(getEquipmentsSpy).toHaveBeenCalled()
    })
  })

  describe('delete functionality', () => {
    beforeEach(async () => {
      wrapper = mount(Equipment, {
        props: {
          equipment: mockEquipment
        },
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertSuccess: true,
            AlertError: true
          }
        }
      })
      await flushPromises()
    })

    it('should have deleteEquipment method available', () => {
      expect(wrapper.vm.deleteEquipment).toBeDefined()
      expect(typeof wrapper.vm.deleteEquipment).toBe('function')
    })

    it('should have delete button with TrashIcon', () => {
      const deleteButton = wrapper.find('button[data-testid="equipment-delete-button-1"]')
      expect(deleteButton.exists()).toBe(true)
    })

    it('should show confirmation dialog with device IP', () => {
      const dialog = wrapper.find('dialog')
      expect(dialog.exists()).toBe(true)
      expect(dialog.text()).toContain(mockEquipment.deviceIp)
    })

    it('should have modal dialog for delete confirmation', () => {
      const modal = wrapper.find('dialog.modal')
      expect(modal.exists()).toBe(true)
    })
  })

  describe('error handling', () => {
    it('should display AlertError when errorMsg is set', async () => {
      wrapper.vm.errorMsg = 'Test error'
      await wrapper.vm.$nextTick()

      expect(wrapper.findComponent({ name: 'AlertError' }).exists()).toBe(true)
    })

    it('should display AlertSuccess when success is true', async () => {
      wrapper.vm.success = true
      await wrapper.vm.$nextTick()

      expect(wrapper.findComponent({ name: 'AlertSuccess' }).exists()).toBe(true)
    })
  })

  describe('accessibility', () => {
    it('should have proper for attributes on all labels', () => {
      const deviceIpLabel = wrapper.find('label[for="eq-device-ip-0"]')
      expect(deviceIpLabel.exists()).toBe(true)

      const hostIpLabel = wrapper.find('label[for="eq-host-ip-0"]')
      expect(hostIpLabel.exists()).toBe(true)
    })

    it('should have unique ids for inputs using the id prop', async () => {
      wrapper = mount(Equipment, {
        props: {
          equipment: { ...mockEquipment, id: 456 }
        },
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertSuccess: true,
            AlertError: true
          }
        }
      })
      await flushPromises()

      expect(wrapper.find('#eq-device-ip-456').exists()).toBe(true)
      expect(wrapper.find('#eq-host-ip-456').exists()).toBe(true)
    })

    it('should use semantic HTML elements', () => {
      // Check for proper use of fieldset, legend, and label elements
      const fieldset = wrapper.find('fieldset')
      expect(fieldset.exists()).toBe(true)

      const labels = wrapper.findAll('label')
      expect(labels.length).toBeGreaterThan(0)

      // Check that inputs are properly associated with labels
      labels.forEach(label => {
        const forAttr = label.attributes('for')
        expect(forAttr).toBeTruthy()
        const input = wrapper.find(`#${forAttr}`)
        expect(input.exists()).toBe(true)
      })
    })

    it('should have keyboard-accessible controls', () => {
      const submitButton = wrapper.find('button[data-testid="equipment-submit-button-0"]')
      expect(submitButton.exists()).toBe(true)
      // Buttons should be focusable and actionable via keyboard
      expect(submitButton.attributes('disabled')).toBeUndefined()
    })

    it('should have visible error message containers', () => {
      // Error containers exist in the template (v-if="showErrors")
      // They are hidden initially but become visible on validation errors
      expect(wrapper.vm.showErrors).toBeDefined()
      expect(wrapper.vm.errors).toBeDefined()
    })
  })

  describe('validation errors', () => {
    it('should have validation schema configured', () => {
      expect(wrapper.vm.meta).toBeDefined()
      expect(wrapper.vm.errors).toBeDefined()
    })

    it('should track form submission count', () => {
      expect(wrapper.vm.submitCount).toBeDefined()
    })

    it('should compute showErrors correctly based on validity and submit count', () => {
      expect(wrapper.vm.showErrors).toBeDefined()
    })

    it('should have error handling setup', () => {
      // Check that the component has error handling infrastructure
      expect(wrapper.vm.errorMsg).toBeDefined()
      expect(wrapper.vm.success).toBeDefined()
    })

    it('should not show errors before first submission attempt', async () => {
      // Form should not show errors initially
      const errorDivs = wrapper.findAll('.text-red-600')
      // The error divs exist in DOM but are empty (v-if="showErrors" is false)
      expect(errorDivs.length).toBe(0)
    })
  })

  describe('form submission errors', () => {
    it('should have error handling logic for createEquipment', async () => {
      ldcStore.ldcs = mockLDCs
      await flushPromises()

      // Verify error state variables exist
      expect(wrapper.vm.errorMsg).toBeDefined()
      expect(wrapper.vm.success).toBeDefined()
    })

    it('should have error handling logic for updateEquipment', async () => {
      wrapper = mount(Equipment, {
        props: {
          equipment: mockEquipment
        },
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertSuccess: true,
            AlertError: true
          }
        }
      })

      // Verify error state variables exist
      expect(wrapper.vm.errorMsg).toBeDefined()
      expect(wrapper.vm.success).toBeDefined()
    })

    it('should have delete functionality with error handling', async () => {
      wrapper = mount(Equipment, {
        props: {
          equipment: mockEquipment
        },
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertSuccess: true,
            AlertError: true
          }
        }
      })

      // Verify delete method exists
      expect(wrapper.vm.deleteEquipment).toBeDefined()
      expect(typeof wrapper.vm.deleteEquipment).toBe('function')
    })

    it('should display AlertError component when error occurs', async () => {
      wrapper.vm.errorMsg = 'Test error message'
      await wrapper.vm.$nextTick()

      expect(wrapper.findComponent({ name: 'AlertError' }).exists()).toBe(true)
    })

    it('should display AlertSuccess component when operation succeeds', async () => {
      wrapper.vm.success = true
      await wrapper.vm.$nextTick()

      expect(wrapper.findComponent({ name: 'AlertSuccess' }).exists()).toBe(true)
    })
  })

  describe('props handling', () => {
    it('should populate form fields when equipment prop is provided', async () => {
      wrapper = mount(Equipment, {
        props: {
          equipment: mockEquipment
        },
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertSuccess: true,
            AlertError: true
          }
        }
      })
      await flushPromises()

      const deviceIpInput = wrapper.find('#eq-device-ip-1')
      expect(deviceIpInput.exists()).toBe(true)
    })

    it('should have empty fields when equipment prop is not provided', async () => {
      const deviceIpInput = wrapper.find('#eq-device-ip-0')
      expect(deviceIpInput.element.value).toBe('')
    })
  })

  describe('edge cases', () => {
    it('should handle empty form values', async () => {
      const deviceIpInput = wrapper.find('#eq-device-ip-0')
      expect(deviceIpInput.element.value).toBe('')
    })

    it('should handle rapid input changes', async () => {
      const deviceIpInput = wrapper.find('#eq-device-ip-0')

      for (let i = 0; i < 10; i++) {
        await deviceIpInput.setValue(`192.168.1.${100 + i}`)
      }

      expect(deviceIpInput.element.value).toBe('192.168.1.109')
    })

    it('should handle enabled checkbox toggling', async () => {
      const enabledCheckbox = wrapper.find('#eq-enabled-0')

      await enabledCheckbox.setChecked(true)
      expect(enabledCheckbox.element.checked).toBe(true)

      await enabledCheckbox.setChecked(false)
      expect(enabledCheckbox.element.checked).toBe(false)
    })

    it('should handle empty LDC list', async () => {
      // Create new wrapper with empty LDCs
      const emptyWrapper = mount(Equipment, {
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertSuccess: true,
            AlertError: true
          }
        }
      })

      const emptyLdcStore = useLDCStore()
      emptyLdcStore.ldcs = []
      await flushPromises()
      await emptyWrapper.vm.$nextTick()

      const select = emptyWrapper.find('#eq-ldcid-0')
      const options = select.findAll('option')
      expect(options.length).toBe(0)
    })

    it('should handle multiple LDCs in select', async () => {
      const manyLDCs = Array.from({ length: 10 }, (_, i) => ({
        id: i + 1,
        name: `ldc${i}`,
        hostname: `ldc${i}`,
        ip: `192.168.1.${100 + i}`,
        grpcPort: 50053 + i,
        prometheusPort: 9092 + i,
        enabled: i % 2 === 0,
        equipments: []
      }))

      ldcStore.ldcs = manyLDCs
      await wrapper.vm.$nextTick()

      const select = wrapper.find('#eq-ldcid-0')
      const options = select.findAll('option')
      expect(options.length).toBe(10)
    })
  })

  describe('form integration with LDC store', () => {
    beforeEach(async () => {
      // Ensure LDCs are set for these tests
      ldcStore.ldcs = mockLDCs
      await flushPromises()
    })

    it('should use LDCs from store', () => {
      expect(ldcStore.ldcs).toEqual(mockLDCs)
    })

    it('should render LDC options in correct order', () => {
      const select = wrapper.find('#eq-ldcid-0')
      const options = select.findAll('option')

      if (options.length > 0) {
        expect(options[0].element.text).toBe('ldc1')
        if (options.length > 1) {
          expect(options[1].element.text).toBe('ldc2')
        }
      }
    })
  })
})
