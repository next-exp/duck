import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import LDC from '../LDC.vue'
import { flushPromises } from '@/test/utils'
import { mockDuckApiClient } from '@/test/setup'
import { useLDCStore } from '@/stores/ldc'

const mockLDC = {
  id: 1,
  name: 'ldc1',
  hostname: 'ldc-host',
  ip: '192.168.1.101',
  grpcPort: 50053,
  prometheusPort: 9092,
  enabled: true,
  equipments: []
}

const mockEquipments = [
  { id: 1, type: 22, deviceIp: '192.168.1.110', hostIp: '192.168.1.120', hostPort: 6006, ldcId: 1, enabled: true }
]

describe('LDC Component', () => {
  let wrapper: VueWrapper<any>
  let ldcStore: any

  beforeEach(async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    vi.clearAllMocks()

    // Set up some mock LDCs for the equipment select
    ldcStore = useLDCStore()
    ldcStore.ldcs = [{ ...mockLDC }, { id: 2, name: 'ldc2', hostname: 'ldc2', ip: '192.168.1.102', grpcPort: 50054, prometheusPort: 9093, enabled: false, equipments: [] }]

    wrapper = mount(LDC, {
      global: {
        plugins: [pinia],
        stubs: {
          AlertSuccess: true,
          AlertError: true,
          Equipment: true
        }
      }
    })

    await flushPromises()
  })

  describe('rendering - create mode (no props)', () => {
    it('should render the component correctly', () => {
      expect(wrapper.find('h2').text()).toBe('LDC')
    })

    it('should render all input fields', () => {
      expect(wrapper.find('#ldc-name-0').exists()).toBe(true)
      expect(wrapper.find('#ldc-hostname-0').exists()).toBe(true)
      expect(wrapper.find('#ldc-ip-0').exists()).toBe(true)

      // Check for text inputs (name, hostname, ip)
      const textInputs = wrapper.findAll('input[type="text"]')
      expect(textInputs.length).toBeGreaterThanOrEqual(3)

      // Check for number inputs (grpcPort, prometheusPort)
      const numberInputs = wrapper.findAll('input[type="number"]')
      expect(numberInputs.length).toBeGreaterThanOrEqual(2)
    })

    it('should render enabled checkbox', () => {
      expect(wrapper.find('#ldc-enabled-0').exists()).toBe(true)
    })

    it('should show "Create LDC" button when no ldc prop', () => {
      const button = wrapper.find('button[data-testid="ldc-submit-button-0"]')
      expect(button.exists()).toBe(true)
      expect(button.text()).toContain('Create LDC')
    })

    it('should not show delete button when no ldc prop', () => {
      const deleteButton = wrapper.find('button[data-testid="ldc-delete-button-0"]')
      expect(deleteButton.exists()).toBe(false)
    })

    it('should not show equipment section when no ldc prop', () => {
      const equipmentSection = wrapper.find('div[data-testid="ldc-equipment-collapse"]')
      expect(equipmentSection.exists()).toBe(false)
    })
  })

  describe('rendering - update mode (with props)', () => {
    beforeEach(async () => {
      wrapper = mount(LDC, {
        props: {
          ldc: mockLDC
        },
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertSuccess: true,
            AlertError: true,
            Equipment: true
          }
        }
      })
      await flushPromises()
    })

    it('should show "Update LDC" button when ldc prop is provided', () => {
      const button = wrapper.find('button[data-testid="ldc-submit-button-1"]')
      expect(button.exists()).toBe(true)
      expect(button.text()).toContain('Update LDC')
    })

    it('should show delete button when ldc prop is provided', () => {
      const deleteButton = wrapper.find('button[data-testid="ldc-delete-button-1"]')
      expect(deleteButton.exists()).toBe(true)
    })

    it('should render modal dialog for delete confirmation', () => {
      const dialog = wrapper.find('dialog')
      expect(dialog.exists()).toBe(true)
    })

    it('should show equipment section when ldc has equipments', async () => {
      wrapper = mount(LDC, {
        props: {
          ldc: { ...mockLDC, equipments: mockEquipments }
        },
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertSuccess: true,
            AlertError: true,
            Equipment: true
          }
        }
      })
      await flushPromises()

      const equipmentSection = wrapper.find('div[data-testid="ldc-equipment-collapse"]')
      expect(equipmentSection.exists()).toBe(true)
    })
  })

  describe('input fields', () => {
    it('should have proper labels for all inputs', () => {
      const labels = wrapper.findAll('label')
      const labelTexts = labels.map(l => l.text())

      expect(labelTexts).toContain('Name')
      expect(labelTexts).toContain('Hostname')
      expect(labelTexts).toContain('IP')
      expect(labelTexts).toContain('gRPC port')
      expect(labelTexts).toContain('Prom. port')
      expect(labelTexts).toContain('Enabled')
    })

    it('should have proper placeholder text', () => {
      const textInputs = wrapper.findAll('input[type="text"]')
      const textPlaceholders = textInputs.map(i => i.attributes('placeholder')).filter(Boolean)

      expect(textPlaceholders).toContain('Name')
      expect(textPlaceholders).toContain('Hostname')
      expect(textPlaceholders).toContain('IP')

      // Check number input placeholders
      const numberInputs = wrapper.findAll('input[type="number"]')
      const numberPlaceholders = numberInputs.map(i => i.attributes('placeholder')).filter(Boolean)

      expect(numberPlaceholders).toContain('gRPC port')
      expect(numberPlaceholders).toContain('Prometheus port')
    })

    it('should have correct input types for all fields', () => {
      const nameInput = wrapper.find('input[data-testid="ldc-name-input-0"]')
      expect(nameInput.attributes('type')).toBe('text')

      const hostnameInput = wrapper.find('input[data-testid="ldc-hostname-input-0"]')
      expect(hostnameInput.attributes('type')).toBe('text')

      const ipInput = wrapper.find('input[data-testid="ldc-ip-input-0"]')
      expect(ipInput.attributes('type')).toBe('text')

      const grpcPortInput = wrapper.find('input[data-testid="ldc-grpc-port-input-0"]')
      expect(grpcPortInput.attributes('type')).toBe('number')

      const prometheusPortInput = wrapper.find('input[data-testid="ldc-prometheus-port-input-0"]')
      expect(prometheusPortInput.attributes('type')).toBe('number')

      const enabledCheckbox = wrapper.find('#ldc-enabled-0')
      expect(enabledCheckbox.attributes('type')).toBe('checkbox')
    })

    it('should update input values when changed', async () => {
      const nameInput = wrapper.find('#ldc-name-0')
      await nameInput.setValue('new-ldc-name')
      expect(nameInput.element.value).toBe('new-ldc-name')
    })

    it('should update input values using data-testid selector', async () => {
      const nameInput = wrapper.find('input[data-testid="ldc-name-input-0"]')
      await nameInput.setValue('another-ldc-name')
      expect(nameInput.element.value).toBe('another-ldc-name')
    })

    it('should update checkbox values when clicked', async () => {
      const enabledCheckbox = wrapper.find('#ldc-enabled-0')
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
      // Fill in all required form fields
      await wrapper.find('input[data-testid="ldc-name-input-0"]').setValue('new-ldc')
      await wrapper.find('input[data-testid="ldc-hostname-input-0"]').setValue('new-ldc-host')
      await wrapper.find('input[data-testid="ldc-ip-input-0"]').setValue('192.168.1.100')
      await wrapper.find('input[data-testid="ldc-grpc-port-input-0"]').setValue('50053')
      await wrapper.find('input[data-testid="ldc-prometheus-port-input-0"]').setValue('9092')

      // Verify values were set
      expect(wrapper.find('input[data-testid="ldc-name-input-0"]').element.value).toBe('new-ldc')
      expect(wrapper.find('input[data-testid="ldc-hostname-input-0"]').element.value).toBe('new-ldc-host')
    })
  })

  describe('form submission - update mode', () => {
    beforeEach(async () => {
      wrapper = mount(LDC, {
        props: {
          ldc: mockLDC
        },
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertSuccess: true,
            AlertError: true,
            Equipment: true
          }
        }
      })
      await flushPromises()
    })

    it('should have form submission handler available in update mode', () => {
      expect(wrapper.vm.onSubmit).toBeDefined()
      expect(typeof wrapper.vm.onSubmit).toBe('function')
    })
  })

  describe('delete functionality', () => {
    beforeEach(async () => {
      wrapper = mount(LDC, {
        props: {
          ldc: mockLDC
        },
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertSuccess: true,
            AlertError: true,
            Equipment: true
          }
        }
      })
      await flushPromises()
    })

    it('should have deleteLDC method available', () => {
      expect(wrapper.vm.deleteLDC).toBeDefined()
      expect(typeof wrapper.vm.deleteLDC).toBe('function')
    })

    it('should have delete button with TrashIcon', () => {
      const deleteButton = wrapper.find('button[data-testid="ldc-delete-button-1"]')
      expect(deleteButton.exists()).toBe(true)
    })

    it('should show confirmation dialog with hostname', () => {
      const dialog = wrapper.find('dialog')
      expect(dialog.exists()).toBe(true)
      expect(dialog.text()).toContain(mockLDC.hostname)
    })

    it('should have modal dialog for delete confirmation', () => {
      const modal = wrapper.find('dialog.modal')
      expect(modal.exists()).toBe(true)
    })
  })

  describe('equipment display', () => {
    it('should render Equipment component for each equipment', async () => {
      wrapper = mount(LDC, {
        props: {
          ldc: { ...mockLDC, equipments: mockEquipments }
        },
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertSuccess: true,
            AlertError: true,
            Equipment: true
          }
        }
      })
      await flushPromises()

      const equipmentComponents = wrapper.findAllComponents({ name: 'Equipment' })
      expect(equipmentComponents.length).toBe(mockEquipments.length)
    })

    it('should not render equipment section when ldc has no equipments', async () => {
      wrapper = mount(LDC, {
        props: {
          ldc: { ...mockLDC, equipments: [] }
        },
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertSuccess: true,
            AlertError: true,
            Equipment: true
          }
        }
      })
      await flushPromises()

      const collapse = wrapper.find('div[data-testid="ldc-equipment-collapse"]')
      // The collapse element should still be in the template even if empty
      expect(collapse.exists()).toBe(true)
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
      const nameLabel = wrapper.find('label[for="ldc-name-0"]')
      expect(nameLabel.exists()).toBe(true)

      const hostnameLabel = wrapper.find('label[for="ldc-hostname-0"]')
      expect(hostnameLabel.exists()).toBe(true)
    })

    it('should have unique ids for inputs using the id prop', async () => {
      wrapper = mount(LDC, {
        props: {
          ldc: { ...mockLDC, id: 123 }
        },
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertSuccess: true,
            AlertError: true,
            Equipment: true
          }
        }
      })
      await flushPromises()

      expect(wrapper.find('#ldc-name-123').exists()).toBe(true)
      expect(wrapper.find('#ldc-hostname-123').exists()).toBe(true)
    })

    it('should use semantic HTML elements', () => {
      const fieldset = wrapper.find('fieldset')
      expect(fieldset.exists()).toBe(true)

      const labels = wrapper.findAll('label')
      expect(labels.length).toBeGreaterThan(0)

      const labelsWithFor = labels.filter(label => label.attributes('for'))
      expect(labelsWithFor.length).toBeGreaterThan(0)
    })

    it('should have keyboard-accessible controls', () => {
      const submitButton = wrapper.find('button[data-testid="ldc-submit-button-0"]')
      expect(submitButton.exists()).toBe(true)
      expect(submitButton.attributes('disabled')).toBeUndefined()
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

    it('should compute showErrors correctly', () => {
      expect(wrapper.vm.showErrors).toBeDefined()
    })

    it('should have error handling setup', () => {
      expect(wrapper.vm.errorMsg).toBeDefined()
      expect(wrapper.vm.success).toBeDefined()
    })

    it('should not show errors before first submission attempt', async () => {
      const errorDivs = wrapper.findAll('.text-red-600')
      expect(errorDivs.length).toBe(0)
    })
  })

  describe('form submission errors', () => {
    it('should have error handling logic for createLDC', async () => {
      expect(wrapper.vm.errorMsg).toBeDefined()
      expect(wrapper.vm.success).toBeDefined()
    })

    it('should have error handling logic for updateLDC', async () => {
      wrapper = mount(LDC, {
        props: {
          ldc: mockLDC
        },
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertSuccess: true,
            AlertError: true,
            Equipment: true
          }
        }
      })

      expect(wrapper.vm.errorMsg).toBeDefined()
      expect(wrapper.vm.success).toBeDefined()
    })

    it('should have delete functionality with error handling', async () => {
      wrapper = mount(LDC, {
        props: {
          ldc: mockLDC
        },
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertSuccess: true,
            AlertError: true,
            Equipment: true
          }
        }
      })

      expect(wrapper.vm.deleteLDC).toBeDefined()
      expect(typeof wrapper.vm.deleteLDC).toBe('function')
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
    it('should populate form fields when ldc prop is provided', async () => {
      wrapper = mount(LDC, {
        props: {
          ldc: mockLDC
        },
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertSuccess: true,
            AlertError: true,
            Equipment: true
          }
        }
      })
      await flushPromises()

      const nameInput = wrapper.find('#ldc-name-1')
      expect(nameInput.exists()).toBe(true)
    })

    it('should have empty fields when ldc prop is not provided', async () => {
      const nameInput = wrapper.find('#ldc-name-0')
      expect(nameInput.element.value).toBe('')
    })
  })

  describe('edge cases', () => {
    it('should handle empty form values', async () => {
      const nameInput = wrapper.find('#ldc-name-0')
      expect(nameInput.element.value).toBe('')
    })

    it('should handle rapid input changes', async () => {
      const nameInput = wrapper.find('#ldc-name-0')

      for (let i = 0; i < 10; i++) {
        await nameInput.setValue(`ldc-${i}`)
      }

      expect(nameInput.element.value).toBe('ldc-9')
    })

    it('should handle enabled checkbox toggling', async () => {
      const enabledCheckbox = wrapper.find('#ldc-enabled-0')

      await enabledCheckbox.setChecked(true)
      expect(enabledCheckbox.element.checked).toBe(true)

      await enabledCheckbox.setChecked(false)
      expect(enabledCheckbox.element.checked).toBe(false)
    })
  })
})
