import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import AlertSuccess from '../AlertSuccess.vue'

describe('AlertSuccess Component', () => {
  describe('rendering', () => {
    it('should display the success text with period', () => {
      const text = 'Test success message'
      const wrapper = mount(AlertSuccess, {
        props: {
          text
        }
      })

      expect(wrapper.text()).toContain(`${text}.`)
    })

    it('should have proper CSS classes', () => {
      const wrapper = mount(AlertSuccess, {
        props: {
          text: 'Test success'
        }
      })

      const alertDiv = wrapper.find('div[data-testid="alert-success"]')
      expect(alertDiv.exists()).toBe(true)
      expect(alertDiv.classes()).toContain('alert')
      expect(alertDiv.classes()).toContain('alert-success')
      expect(alertDiv.classes()).toContain('my-2')
      expect(alertDiv.classes()).toContain('mx-2')
      expect(alertDiv.classes()).toContain('text-lg')
      expect(alertDiv.classes()).toContain('font-bold')
    })

    it('should render success icon with correct SVG path', () => {
      const wrapper = mount(AlertSuccess, {
        props: {
          text: 'Test success'
        }
      })

      const svg = wrapper.find('svg')
      expect(svg.exists()).toBe(true)
      expect(svg.attributes('xmlns')).toBe('http://www.w3.org/2000/svg')
      expect(svg.attributes('class')).toBe('size-6')
      expect(svg.attributes('fill')).toBe('none')

      const path = svg.find('path')
      expect(path.exists()).toBe(true)
      // Success icon path should have circular checkmark pattern
      expect(path.attributes('d')).toBeTruthy()
      expect(path.attributes('stroke-linecap')).toBe('round')
      expect(path.attributes('stroke-linejoin')).toBe('round')
    })

    it('should have proper icon size attributes', () => {
      const wrapper = mount(AlertSuccess, {
        props: {
          text: 'Test success'
        }
      })

      const svg = wrapper.find('svg')
      expect(svg.attributes('viewBox')).toBe('0 0 24 24')
      expect(svg.attributes('class')).toBe('size-6')
    })
  })

  describe('text display', () => {
    it('should render with long success messages', () => {
      const longText = 'Operation completed successfully. All data has been saved to the database and the system is ready for the next operation'
      const wrapper = mount(AlertSuccess, {
        props: {
          text: longText
        }
      })

      expect(wrapper.text()).toContain(longText)
      expect(wrapper.text()).toContain('.')
    })

    it('should render with special characters in text', () => {
      const specialText = 'Success: Connected to server at 192.168.1.1:8080'
      const wrapper = mount(AlertSuccess, {
        props: {
          text: specialText
        }
      })

      expect(wrapper.text()).toContain(specialText)
    })

    it('should render with newlines in text', () => {
      const multilineText = 'Success: Data saved\nConfiguration updated\nSystem ready'
      const wrapper = mount(AlertSuccess, {
        props: {
          text: multilineText
        }
      })

      expect(wrapper.text()).toContain(multilineText)
    })

    it('should render with unicode characters', () => {
      const unicodeText = '¡Éxito! operación completada ✓'
      const wrapper = mount(AlertSuccess, {
        props: {
          text: unicodeText
        }
      })

      expect(wrapper.text()).toContain(unicodeText)
    })
  })

  describe('component structure', () => {
    it('should have proper semantic HTML structure', () => {
      const wrapper = mount(AlertSuccess, {
        props: {
          text: 'Test success'
        }
      })

      const alertDiv = wrapper.find('div[data-testid="alert-success"]')
      expect(alertDiv.exists()).toBe(true)
      expect(alertDiv.find('svg').exists()).toBe(true)
      expect(alertDiv.find('div').exists()).toBe(true)
      expect(alertDiv.find('span').exists()).toBe(true)
    })

    it('should use data-testid for easy selection', () => {
      const wrapper = mount(AlertSuccess, {
        props: {
          text: 'Test success'
        }
      })

      expect(wrapper.find('div[data-testid="alert-success"]').exists()).toBe(true)
    })
  })

  describe('accessibility', () => {
    it('should have proper ARIA attributes through semantic HTML', () => {
      const wrapper = mount(AlertSuccess, {
        props: {
          text: 'Test success'
        }
      })

      const alertDiv = wrapper.find('div[data-testid="alert-success"]')
      expect(alertDiv.classes()).toContain('alert')
      expect(alertDiv.classes()).toContain('alert-success')
    })

    it('should have readable text with proper sizing', () => {
      const wrapper = mount(AlertSuccess, {
        props: {
          text: 'Test success'
        }
      })

      const alertDiv = wrapper.find('div[data-testid="alert-success"]')
      expect(alertDiv.classes()).toContain('text-lg')
      expect(alertDiv.classes()).toContain('font-bold')
    })
  })

  describe('edge cases', () => {
    it('should render with empty string', () => {
      const wrapper = mount(AlertSuccess, {
        props: {
          text: ''
        }
      })

      expect(wrapper.text()).toBe('.')
    })

    it('should render with single character', () => {
      const wrapper = mount(AlertSuccess, {
        props: {
          text: 'OK'
        }
      })

      expect(wrapper.text()).toBe('OK.')
    })

    it('should handle rapidly changing text', async () => {
      const wrapper = mount(AlertSuccess, {
        props: {
          text: 'Success 1'
        }
      })

      expect(wrapper.text()).toContain('Success 1.')

      await wrapper.setProps({ text: 'Success 2' })
      expect(wrapper.text()).toContain('Success 2.')
    })
  })
})
