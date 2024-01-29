import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import CheckDisabled from '../CheckDisabled.vue'
import { flushPromises } from '@/test/utils'
import { mockDuckApiClient } from '@/test/setup'

describe('CheckDisabled Component', () => {
  let wrapper: VueWrapper<any>

  beforeEach(async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    vi.clearAllMocks()

    // Reset mock to default behavior
    mockDuckApiClient.checkDisabled.mockResolvedValue({
      warnings: {
        gdcs: [],
        ldcs: [],
        equipments: [],
        writing: []
      }
    })

    wrapper = mount(CheckDisabled, {
      global: {
        plugins: [pinia],
        stubs: {
          AlertError: true,
          AlertWarning: true
        }
      }
    })

    // Wait for nextTick and flush promises to ensure component is mounted and API call completes
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

    it('should initialize with empty disabled state', () => {
      expect(wrapper.vm.disabled).toBeDefined()
      expect(wrapper.vm.disabled.gdcs).toEqual([])
      expect(wrapper.vm.disabled.ldcs).toEqual([])
      expect(wrapper.vm.disabled.equipments).toEqual([])
      expect(wrapper.vm.disabled.writing).toEqual([])
    })

    it('should initialize with empty error state', () => {
      expect(wrapper.vm.errorCheckDisabled).toBeDefined()
      expect(wrapper.vm.errorCheckDisabled).toBe('')
    })
  })

  describe('API call on mount', () => {
    it('should call checkDisabled API on mount', () => {
      expect(mockDuckApiClient.checkDisabled).toHaveBeenCalled()
    })

    it('should call checkDisabled API only once on mount', () => {
      expect(mockDuckApiClient.checkDisabled).toHaveBeenCalledTimes(1)
    })

    it('should have checkDisabled method available', () => {
      expect(wrapper.vm.checkDisabled).toBeDefined()
      expect(typeof wrapper.vm.checkDisabled).toBe('function')
    })
  })

  describe('warning display - disabled GDCs', () => {
    it('should display AlertWarning when disabled GDCs exist', async () => {
      mockDuckApiClient.checkDisabled.mockReset()
      mockDuckApiClient.checkDisabled.mockResolvedValue({
        warnings: {
          gdcs: ['gdc1', 'gdc2'],
          ldcs: [],
          equipments: [],
          writing: []
        }
      })

      wrapper = mount(CheckDisabled, {
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertError: true,
            AlertWarning: true
          }
        }
      })

      await flushPromises()
      await flushPromises() // Double flush to ensure async operations complete

      const alertWarning = wrapper.findAllComponents({ name: 'AlertWarning' })
      expect(alertWarning.length).toBeGreaterThan(0)

      const gdcWarning = alertWarning.find(w => w.props('text').includes('Disabled GDCs'))
      expect(gdcWarning).toBeDefined()
      expect(gdcWarning?.props('text')).toContain('gdc1')
      expect(gdcWarning?.props('text')).toContain('gdc2')
    })

    it('should not display AlertWarning for GDCs when array is empty', () => {
      const alertWarning = wrapper.findAllComponents({ name: 'AlertWarning' })
      const gdcWarning = alertWarning.find(w => w.props('text')?.includes('Disabled GDCs'))
      expect(gdcWarning).toBeUndefined()
    })

    it('should handle single disabled GDC', async () => {
      mockDuckApiClient.checkDisabled.mockReset()
      mockDuckApiClient.checkDisabled.mockResolvedValue({
        warnings: {
          gdcs: ['gdc1'],
          ldcs: [],
          equipments: [],
          writing: []
        }
      })

      wrapper = mount(CheckDisabled, {
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertError: true,
            AlertWarning: true
          }
        }
      })

      await flushPromises()

      expect(wrapper.vm.disabled.gdcs).toEqual(['gdc1'])
    })
  })

  describe('warning display - disabled LDCs', () => {
    it('should display AlertWarning when disabled LDCs exist', async () => {
      mockDuckApiClient.checkDisabled.mockReset()
      mockDuckApiClient.checkDisabled.mockResolvedValue({
        warnings: {
          gdcs: [],
          ldcs: ['ldc1', 'ldc2', 'ldc3'],
          equipments: [],
          writing: []
        }
      })

      wrapper = mount(CheckDisabled, {
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertError: true,
            AlertWarning: true
          }
        }
      })

      await flushPromises()

      const alertWarning = wrapper.findAllComponents({ name: 'AlertWarning' })
      const ldcWarning = alertWarning.find(w => w.props('text').includes('Disabled LDCs'))
      expect(ldcWarning).toBeDefined()
      expect(ldcWarning?.props('text')).toContain('ldc1')
      expect(ldcWarning?.props('text')).toContain('ldc2')
      expect(ldcWarning?.props('text')).toContain('ldc3')
    })

    it('should not display AlertWarning for LDCs when array is empty', () => {
      const alertWarning = wrapper.findAllComponents({ name: 'AlertWarning' })
      const ldcWarning = alertWarning.find(w => w.props('text')?.includes('Disabled LDCs'))
      expect(ldcWarning).toBeUndefined()
    })
  })

  describe('warning display - disabled Equipments', () => {
    it('should display AlertWarning when disabled Equipments exist', async () => {
      mockDuckApiClient.checkDisabled.mockReset()
      mockDuckApiClient.checkDisabled.mockResolvedValue({
        warnings: {
          gdcs: [],
          ldcs: [],
          equipments: ['equipment1', 'equipment2'],
          writing: []
        }
      })

      wrapper = mount(CheckDisabled, {
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertError: true,
            AlertWarning: true
          }
        }
      })

      await flushPromises()

      const alertWarning = wrapper.findAllComponents({ name: 'AlertWarning' })
      const equipmentWarning = alertWarning.find(w => w.props('text').includes('Disabled Equipments'))
      expect(equipmentWarning).toBeDefined()
      expect(equipmentWarning?.props('text')).toContain('equipment1')
      expect(equipmentWarning?.props('text')).toContain('equipment2')
    })

    it('should not display AlertWarning for Equipments when array is empty', () => {
      const alertWarning = wrapper.findAllComponents({ name: 'AlertWarning' })
      const equipmentWarning = alertWarning.find(w => w.props('text')?.includes('Disabled Equipments'))
      expect(equipmentWarning).toBeUndefined()
    })
  })

  describe('warning display - writing disabled', () => {
    it('should display AlertWarning when writing is disabled on servers', async () => {
      mockDuckApiClient.checkDisabled.mockReset()
      mockDuckApiClient.checkDisabled.mockResolvedValue({
        warnings: {
          gdcs: [],
          ldcs: [],
          equipments: [],
          writing: ['gdc1', 'ldc1']
        }
      })

      wrapper = mount(CheckDisabled, {
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertError: true,
            AlertWarning: true
          }
        }
      })

      await flushPromises()

      const alertWarning = wrapper.findAllComponents({ name: 'AlertWarning' })
      const writingWarning = alertWarning.find(w => w.props('text').includes('File writing disabled'))
      expect(writingWarning).toBeDefined()
      expect(writingWarning?.props('text')).toContain('gdc1')
      expect(writingWarning?.props('text')).toContain('ldc1')
    })

    it('should not display AlertWarning for writing when array is empty', () => {
      const alertWarning = wrapper.findAllComponents({ name: 'AlertWarning' })
      const writingWarning = alertWarning.find(w => w.props('text')?.includes('File writing disabled'))
      expect(writingWarning).toBeUndefined()
    })
  })

  describe('multiple warnings', () => {
    it('should display multiple warnings when multiple types are disabled', async () => {
      mockDuckApiClient.checkDisabled.mockReset()
      mockDuckApiClient.checkDisabled.mockResolvedValue({
        warnings: {
          gdcs: ['gdc1'],
          ldcs: ['ldc1'],
          equipments: ['equipment1'],
          writing: ['gdc2']
        }
      })

      wrapper = mount(CheckDisabled, {
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertError: true,
            AlertWarning: true
          }
        }
      })

      await flushPromises()

      const alertWarnings = wrapper.findAllComponents({ name: 'AlertWarning' })
      expect(alertWarnings.length).toBe(4)
    })

    it('should display all four warning types when all have data', async () => {
      mockDuckApiClient.checkDisabled.mockReset()
      mockDuckApiClient.checkDisabled.mockResolvedValue({
        warnings: {
          gdcs: ['gdc1', 'gdc2'],
          ldcs: ['ldc1'],
          equipments: ['equipment1'],
          writing: ['gdc3']
        }
      })

      wrapper = mount(CheckDisabled, {
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertError: true,
            AlertWarning: true
          }
        }
      })

      await flushPromises()

      const alertWarnings = wrapper.findAllComponents({ name: 'AlertWarning' })

      const gdcWarning = alertWarnings.find(w => w.props('text').includes('Disabled GDCs'))
      const ldcWarning = alertWarnings.find(w => w.props('text').includes('Disabled LDCs'))
      const equipmentWarning = alertWarnings.find(w => w.props('text').includes('Disabled Equipments'))
      const writingWarning = alertWarnings.find(w => w.props('text').includes('File writing disabled'))

      expect(gdcWarning).toBeDefined()
      expect(ldcWarning).toBeDefined()
      expect(equipmentWarning).toBeDefined()
      expect(writingWarning).toBeDefined()
    })
  })

  describe('component behavior', () => {
    it('should be a read-only component (no user interactions)', () => {
      expect(wrapper.find('button').exists()).toBe(false)
      expect(wrapper.find('input').exists()).toBe(false)
      expect(wrapper.find('form').exists()).toBe(false)
    })

    it('should only display alert components based on API response', () => {
      // When API returns empty arrays, no warnings should be shown
      expect(wrapper.findAllComponents({ name: 'AlertWarning' }).length).toBe(0)
      expect(wrapper.findComponent({ name: 'AlertError' }).exists()).toBe(false)
    })

    it('should update disabled state when API returns data', async () => {
      mockDuckApiClient.checkDisabled.mockReset()
      mockDuckApiClient.checkDisabled.mockResolvedValue({
        warnings: {
          gdcs: ['gdc1'],
          ldcs: ['ldc1'],
          equipments: ['equipment1'],
          writing: ['gdc2']
        }
      })

      wrapper = mount(CheckDisabled, {
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertError: true,
            AlertWarning: true
          }
        }
      })

      await flushPromises()

      expect(wrapper.vm.disabled.gdcs).toEqual(['gdc1'])
      expect(wrapper.vm.disabled.ldcs).toEqual(['ldc1'])
      expect(wrapper.vm.disabled.equipments).toEqual(['equipment1'])
      expect(wrapper.vm.disabled.writing).toEqual(['gdc2'])
    })
  })

  describe('accessibility', () => {
    it('should use semantic HTML elements', () => {
      const container = wrapper.find('div')
      expect(container.exists()).toBe(true)
    })

    it('should pass warning text to AlertWarning for screen readers', async () => {
      mockDuckApiClient.checkDisabled.mockReset()
      mockDuckApiClient.checkDisabled.mockResolvedValue({
        warnings: {
          gdcs: ['gdc1'],
          ldcs: [],
          equipments: [],
          writing: []
        }
      })

      wrapper = mount(CheckDisabled, {
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertError: true,
            AlertWarning: true
          }
        }
      })

      await flushPromises()

      const alertWarning = wrapper.findAllComponents({ name: 'AlertWarning' })
      const gdcWarning = alertWarning.find(w => w.props('text').includes('Disabled GDCs'))
      expect(gdcWarning?.props('text')).toBe('Disabled GDCs: gdc1')
    })
  })

  describe('edge cases', () => {
    it('should handle empty arrays in response', () => {
      // The default mock returns empty arrays
      expect(wrapper.vm.disabled.gdcs).toEqual([])
      expect(wrapper.vm.disabled.ldcs).toEqual([])
      expect(wrapper.vm.disabled.equipments).toEqual([])
      expect(wrapper.vm.disabled.writing).toEqual([])
    })

    it('should handle special characters in server names', async () => {
      // Clear previous mock
      mockDuckApiClient.checkDisabled.mockReset()
      mockDuckApiClient.checkDisabled.mockResolvedValue({
        warnings: {
          gdcs: ['gdc-1.test', 'gdc_2'],
          ldcs: [],
          equipments: [],
          writing: []
        }
      })

      wrapper = mount(CheckDisabled, {
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertError: true,
            AlertWarning: true
          }
        }
      })

      await flushPromises()

      expect(wrapper.vm.disabled.gdcs).toEqual(['gdc-1.test', 'gdc_2'])
    })

    it('should handle response with null warnings', async () => {
      mockDuckApiClient.checkDisabled.mockReset()
      mockDuckApiClient.checkDisabled.mockResolvedValue({
        warnings: null
      })

      wrapper = mount(CheckDisabled, {
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertError: true,
            AlertWarning: true
          }
        }
      })

      await flushPromises()

      // Component should handle null warnings by using fallback || []
      expect(wrapper.vm.disabled).toBeDefined()
    })

    it('should handle long lists of disabled items', async () => {
      mockDuckApiClient.checkDisabled.mockReset()
      mockDuckApiClient.checkDisabled.mockResolvedValue({
        warnings: {
          gdcs: ['gdc1', 'gdc2', 'gdc3', 'gdc4', 'gdc5'],
          ldcs: [],
          equipments: [],
          writing: []
        }
      })

      wrapper = mount(CheckDisabled, {
        global: {
          plugins: [createPinia()],
          stubs: {
            AlertError: true,
            AlertWarning: true
          }
        }
      })

      await flushPromises()

      expect(wrapper.vm.disabled.gdcs.length).toBe(5)
    })
  })
})
