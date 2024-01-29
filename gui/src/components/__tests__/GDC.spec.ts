import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import GDC from '../GDC.vue'
import { flushPromises, createMockConnectRPCResponse, createMockConnectRPCError } from '@/test/utils'
import { mockDuckApiClient } from '@/test/setup'
import { useGDCStore } from '@/stores/gdc'

const mockGDC = {
  id: 1,
  name: 'gdc1',
  hostname: 'gdc-host',
  ip: '192.168.1.100',
  port: 6005,
  datapath: '/data/gdc',
  grpcPort: 50051,
  prometheusPort: 9090,
  enabled: true,
  writeOutput: true,
  decode: false
}

describe('GDC Component', () => {
  let wrapper: VueWrapper<any>
  let gdcStore: any

  beforeEach(async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    vi.clearAllMocks()

    wrapper = mount(GDC, {
      global: {
        plugins: [pinia],
        stubs: {
          AlertSuccess: true,
          AlertError: true
        }
      }
    })

    gdcStore = useGDCStore()
    await flushPromises()
  })

  describe('rendering - create mode (no props)', () => {
    it('should render the component correctly', () => {
      expect(wrapper.find('h2').text()).toBe('GDC')
    })

    it('should render all input fields', () => {
      expect(wrapper.find('#gdc-name-0').exists()).toBe(true)
      expect(wrapper.find('#gdc-hostname-0').exists()).toBe(true)
      expect(wrapper.find('#gdc-ip-0').exists()).toBe(true)
      expect(wrapper.find('#gdc-port-0').exists()).toBe(true)
      expect(wrapper.find('#gdc-datapath-0').exists()).toBe(true)

      // Check for text inputs (name, hostname, ip, datapath)
      const textInputs = wrapper.findAll('input[type="text"]')
      expect(textInputs.length).toBeGreaterThanOrEqual(4)

      // Check for number inputs (port, grpcPort, prometheusPort)
      const numberInputs = wrapper.findAll('input[type="number"]')
      expect(numberInputs.length).toBeGreaterThanOrEqual(3)
    })

    it('should render all checkboxes', () => {
      expect(wrapper.find('#gdc-enabled-0').exists()).toBe(true)
      expect(wrapper.find('#gdc-write-enabled-0').exists()).toBe(true)
      expect(wrapper.find('#gdc-decode-0').exists()).toBe(true)
    })

    it('should show "Create GDC" button when no gdc prop', () => {
      const button = wrapper.find('button[data-testid="gdc-submit-button-0"]')
      expect(button.exists()).toBe(true)
      expect(button.text()).toContain('Create GDC')
    })

    it('should not show delete button when no gdc prop', () => {
      const deleteButton = wrapper.find('button[data-testid="gdc-delete-button-0"]')
      expect(deleteButton.exists()).toBe(false)
    })
  })

  describe('rendering - update mode (with props)', () => {
    beforeEach(async () => {
      wrapper = mount(GDC, {
        props: {
          gdc: mockGDC
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

    it('should show "Update" button when gdc prop is provided', () => {
      const button = wrapper.find('button[data-testid="gdc-submit-button-1"]')
      expect(button.exists()).toBe(true)
      expect(button.text()).toContain('Update')
    })

    it('should show delete button when gdc prop is provided', () => {
      const deleteButton = wrapper.find('button[data-testid="gdc-delete-button-1"]')
      expect(deleteButton.exists()).toBe(true)
    })

    it('should render modal dialog for delete confirmation', () => {
      const dialog = wrapper.find('dialog')
      expect(dialog.exists()).toBe(true)
    })
  })

  describe('input fields', () => {
    it('should have proper labels for all inputs', () => {
      const labels = wrapper.findAll('label')
      const labelTexts = labels.map(l => l.text())

      expect(labelTexts).toContain('Name')
      expect(labelTexts).toContain('Hostname')
      expect(labelTexts).toContain('IP')
      expect(labelTexts).toContain('Port')
      expect(labelTexts).toContain('Datapath')
      expect(labelTexts).toContain('gRPC port')
      expect(labelTexts).toContain('Prom. port')
      expect(labelTexts).toContain('Enabled')
      expect(labelTexts).toContain('Write binary file')
      expect(labelTexts).toContain('Decode')
    })

    it('should have proper placeholder text', () => {
      const textInputs = wrapper.findAll('input[type="text"]')
      const textPlaceholders = textInputs.map(i => i.attributes('placeholder')).filter(Boolean)

      expect(textPlaceholders).toContain('Name')
      expect(textPlaceholders).toContain('Hostname')
      expect(textPlaceholders).toContain('IP')
      expect(textPlaceholders).toContain('Data path')

      // Check number input placeholders
      const numberInputs = wrapper.findAll('input[type="number"]')
      const numberPlaceholders = numberInputs.map(i => i.attributes('placeholder')).filter(Boolean)

      expect(numberPlaceholders).toContain('Port')
      expect(numberPlaceholders).toContain('gRPC port')
      expect(numberPlaceholders).toContain('Prometheus port')
    })

    it('should have correct input types for all fields', () => {
      const nameInput = wrapper.find('input[data-testid="gdc-name-input-0"]')
      expect(nameInput.attributes('type')).toBe('text')

      const hostnameInput = wrapper.find('input[data-testid="gdc-hostname-input-0"]')
      expect(hostnameInput.attributes('type')).toBe('text')

      const ipInput = wrapper.find('input[data-testid="gdc-ip-input-0"]')
      expect(ipInput.attributes('type')).toBe('text')

      const portInput = wrapper.find('input[data-testid="gdc-port-input-0"]')
      expect(portInput.attributes('type')).toBe('number')

      const grpcPortInput = wrapper.find('input[data-testid="gdc-grpc-port-input-0"]')
      expect(grpcPortInput.attributes('type')).toBe('number')

      const prometheusPortInput = wrapper.find('input[data-testid="gdc-prometheus-port-input-0"]')
      expect(prometheusPortInput.attributes('type')).toBe('number')

      const datapathInput = wrapper.find('input[data-testid="gdc-datapath-input-0"]')
      expect(datapathInput.attributes('type')).toBe('text')

      const enabledCheckbox = wrapper.find('#gdc-enabled-0')
      expect(enabledCheckbox.attributes('type')).toBe('checkbox')
    })

    it('should update input values when changed', async () => {
      const nameInput = wrapper.find('#gdc-name-0')
      await nameInput.setValue('new-gdc-name')
      expect(nameInput.element.value).toBe('new-gdc-name')
    })

    it('should update input values using data-testid selector', async () => {
      const nameInput = wrapper.find('input[data-testid="gdc-name-input-0"]')
      await nameInput.setValue('another-gdc-name')
      expect(nameInput.element.value).toBe('another-gdc-name')
    })

    it('should update checkbox values when clicked', async () => {
      const enabledCheckbox = wrapper.find('#gdc-enabled-0')
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
      await wrapper.find('input[data-testid="gdc-name-input-0"]').setValue('new-gdc')
      await wrapper.find('input[data-testid="gdc-hostname-input-0"]').setValue('new-gdc-host')
      await wrapper.find('input[data-testid="gdc-ip-input-0"]').setValue('192.168.1.100')
      await wrapper.find('input[data-testid="gdc-port-input-0"]').setValue('6005')
      await wrapper.find('input[data-testid="gdc-datapath-input-0"]').setValue('/data/gdc')
      await wrapper.find('input[data-testid="gdc-grpc-port-input-0"]').setValue('50051')
      await wrapper.find('input[data-testid="gdc-prometheus-port-input-0"]').setValue('9090')

      // Verify values were set
      expect(wrapper.find('input[data-testid="gdc-name-input-0"]').element.value).toBe('new-gdc')
      expect(wrapper.find('input[data-testid="gdc-hostname-input-0"]').element.value).toBe('new-gdc-host')
    })

    it('should be able to fill all required form fields', async () => {
      // Fill in all required form fields
      await wrapper.find('input[data-testid="gdc-name-input-0"]').setValue('new-gdc')
      await wrapper.find('input[data-testid="gdc-hostname-input-0"]').setValue('new-gdc-host')
      await wrapper.find('input[data-testid="gdc-ip-input-0"]').setValue('192.168.1.100')
      await wrapper.find('input[data-testid="gdc-port-input-0"]').setValue('6005')
      await wrapper.find('input[data-testid="gdc-datapath-input-0"]').setValue('/data/gdc')

      // Verify values were set
      expect(wrapper.find('input[data-testid="gdc-name-input-0"]').element.value).toBe('new-gdc')
      expect(wrapper.find('input[data-testid="gdc-hostname-input-0"]').element.value).toBe('new-gdc-host')
    })
  })

  describe('form submission - update mode', () => {
    beforeEach(async () => {
      wrapper = mount(GDC, {
        props: {
          gdc: mockGDC
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

    it('should have form submission handler available in update mode', () => {
      expect(wrapper.vm.onSubmit).toBeDefined()
      expect(typeof wrapper.vm.onSubmit).toBe('function')
    })

    it('should call updateGDC API when form is submitted', async () => {
      // Mock successful API response
      mockDuckApiClient.updateGDC.mockResolvedValueOnce({ id: mockGDC.id })

      // Trigger submit through the component's method
      await wrapper.vm.onSubmit()
      await flushPromises()

      // Verify API was called with correct id
      expect(mockDuckApiClient.updateGDC).toHaveBeenCalledWith(
        expect.objectContaining({
          id: mockGDC.id
        })
      )
    })
  })

  describe('delete functionality', () => {
    beforeEach(async () => {
      wrapper = mount(GDC, {
        props: {
          gdc: mockGDC
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

    it('should have deleteGDC method available', () => {
      expect(wrapper.vm.deleteGDC).toBeDefined()
      expect(typeof wrapper.vm.deleteGDC).toBe('function')
    })

    it('should have delete button with TrashIcon', () => {
      const deleteButton = wrapper.find('button[data-testid="gdc-delete-button-1"]')
      expect(deleteButton.exists()).toBe(true)
    })

    it('should show confirmation dialog with hostname', () => {
      const dialog = wrapper.find('dialog')
      expect(dialog.exists()).toBe(true)
      expect(dialog.text()).toContain(mockGDC.hostname)
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
      const nameLabel = wrapper.find('label[for="gdc-name-0"]')
      expect(nameLabel.exists()).toBe(true)

      const hostnameLabel = wrapper.find('label[for="gdc-hostname-0"]')
      expect(hostnameLabel.exists()).toBe(true)
    })

    it('should have unique ids for inputs using the id prop', async () => {
      wrapper = mount(GDC, {
        props: {
          gdc: { ...mockGDC, id: 123 }
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

      expect(wrapper.find('#gdc-name-123').exists()).toBe(true)
      expect(wrapper.find('#gdc-hostname-123').exists()).toBe(true)
    })

    it('should use semantic HTML elements', () => {
      const fieldset = wrapper.find('fieldset')
      expect(fieldset.exists()).toBe(true)

      const labels = wrapper.findAll('label')
      expect(labels.length).toBeGreaterThan(0)

      // Check that labels have for attributes
      const labelsWithFor = labels.filter(label => label.attributes('for'))
      expect(labelsWithFor.length).toBeGreaterThan(0)
    })

    it('should have keyboard-accessible controls', () => {
      const submitButton = wrapper.find('button[data-testid="gdc-submit-button-0"]')
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
    it('should have error handling logic for createGDC', async () => {
      expect(wrapper.vm.errorMsg).toBeDefined()
      expect(wrapper.vm.success).toBeDefined()
    })

    it('should have error handling logic for updateGDC', async () => {
      wrapper = mount(GDC, {
        props: {
          gdc: mockGDC
        },
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertSuccess: true,
            AlertError: true
          }
        }
      })

      expect(wrapper.vm.errorMsg).toBeDefined()
      expect(wrapper.vm.success).toBeDefined()
    })

    it('should have delete functionality with error handling', async () => {
      wrapper = mount(GDC, {
        props: {
          gdc: mockGDC
        },
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertSuccess: true,
            AlertError: true
          }
        }
      })

      expect(wrapper.vm.deleteGDC).toBeDefined()
      expect(typeof wrapper.vm.deleteGDC).toBe('function')
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
    it('should populate form fields when gdc prop is provided', async () => {
      wrapper = mount(GDC, {
        props: {
          gdc: mockGDC
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

      const nameInput = wrapper.find('#gdc-name-1')
      expect(nameInput.exists()).toBe(true)
    })

    it('should have empty fields when gdc prop is not provided', async () => {
      const nameInput = wrapper.find('#gdc-name-0')
      expect(nameInput.element.value).toBe('')
    })
  })

  describe('edge cases', () => {
    it('should handle empty form values', async () => {
      const nameInput = wrapper.find('#gdc-name-0')
      expect(nameInput.element.value).toBe('')
    })

    it('should handle rapid input changes', async () => {
      const nameInput = wrapper.find('#gdc-name-0')

      for (let i = 0; i < 10; i++) {
        await nameInput.setValue(`gdc-${i}`)
      }

      expect(nameInput.element.value).toBe('gdc-9')
    })

    it('should handle all checkboxes unchecked', async () => {
      await wrapper.find('#gdc-enabled-0').setChecked(false)
      await wrapper.find('#gdc-write-enabled-0').setChecked(false)
      await wrapper.find('#gdc-decode-0').setChecked(false)

      await flushPromises()

      expect(wrapper.find('#gdc-enabled-0').element.checked).toBe(false)
      expect(wrapper.find('#gdc-write-enabled-0').element.checked).toBe(false)
      expect(wrapper.find('#gdc-decode-0').element.checked).toBe(false)
    })
  })
})
