import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import AlertError from '../AlertError.vue'

describe('AlertError Component', () => {
  describe('rendering', () => {
    it('should display the error text with period', () => {
      const text = 'Test error message'
      const wrapper = mount(AlertError, {
        props: {
          text
        }
      })

      expect(wrapper.text()).toContain(`${text}.`)
    })

    it('should have proper CSS classes', () => {
      const wrapper = mount(AlertError, {
        props: {
          text: 'Test error'
        }
      })

      const alertDiv = wrapper.find('div[data-testid="alert-error"]')
      expect(alertDiv.exists()).toBe(true)
      expect(alertDiv.classes()).toContain('alert')
      expect(alertDiv.classes()).toContain('alert-error')
      expect(alertDiv.classes()).toContain('my-2')
      expect(alertDiv.classes()).toContain('mx-2')
      expect(alertDiv.classes()).toContain('text-lg')
      expect(alertDiv.classes()).toContain('font-bold')
    })

    it('should render error icon with correct SVG path', () => {
      const wrapper = mount(AlertError, {
        props: {
          text: 'Test error'
        }
      })

      const svg = wrapper.find('svg')
      expect(svg.exists()).toBe(true)
      expect(svg.attributes('xmlns')).toBe('http://www.w3.org/2000/svg')
      expect(svg.attributes('class')).toContain('stroke-current')
      expect(svg.attributes('class')).toContain('shrink-0')
      expect(svg.attributes('class')).toContain('h-6')
      expect(svg.attributes('class')).toContain('w-6')

      const path = svg.find('path')
      expect(path.exists()).toBe(true)
      // Error icon should have specific path data
      expect(path.attributes('d')).toContain('M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z')
      expect(path.attributes('stroke-linecap')).toBe('round')
      expect(path.attributes('stroke-linejoin')).toBe('round')
    })

    it('should have proper icon size attributes', () => {
      const wrapper = mount(AlertError, {
        props: {
          text: 'Test error'
        }
      })

      const svg = wrapper.find('svg')
      expect(svg.attributes('viewBox')).toBe('0 0 24 24')
      expect(svg.attributes('fill')).toBe('none')
    })
  })

  describe('text display', () => {
    it('should render with long error messages', () => {
      const longText = 'This is a very long error message that contains a lot of information about what went wrong and should still be displayed properly'
      const wrapper = mount(AlertError, {
        props: {
          text: longText
        }
      })

      expect(wrapper.text()).toContain(longText)
      expect(wrapper.text()).toContain('.')
    })

    it('should render with special characters in text', () => {
      const specialText = 'Error: Failed to connect to 192.168.1.1:8080 (timeout)'
      const wrapper = mount(AlertError, {
        props: {
          text: specialText
        }
      })

      expect(wrapper.text()).toContain(specialText)
    })

    it('should render with newlines in text', () => {
      const multilineText = 'Error: Connection failed\nRetry attempt 1 failed\nPlease check your network'
      const wrapper = mount(AlertError, {
        props: {
          text: multilineText
        }
      })

      expect(wrapper.text()).toContain(multilineText)
    })

    it('should render with HTML entities in text', () => {
      const htmlText = 'Error: <configuration> & "settings" are invalid'
      const wrapper = mount(AlertError, {
        props: {
          text: htmlText
        }
      })

      // Vue should escape HTML by default
      expect(wrapper.text()).toContain(htmlText)
    })

    it('should render with unicode characters', () => {
      const unicodeText = 'Error: conexión fallida ❌'
      const wrapper = mount(AlertError, {
        props: {
          text: unicodeText
        }
      })

      expect(wrapper.text()).toContain(unicodeText)
    })
  })

  describe('component structure', () => {
    it('should have proper semantic HTML structure', () => {
      const wrapper = mount(AlertError, {
        props: {
          text: 'Test error'
        }
      })

      const alertDiv = wrapper.find('div[data-testid="alert-error"]')
      expect(alertDiv.exists()).toBe(true)
      expect(alertDiv.find('svg').exists()).toBe(true)
      expect(alertDiv.find('div').exists()).toBe(true)
      expect(alertDiv.find('span').exists()).toBe(true)
    })

    it('should use data-testid for easy selection', () => {
      const wrapper = mount(AlertError, {
        props: {
          text: 'Test error'
        }
      })

      expect(wrapper.find('div[data-testid="alert-error"]').exists()).toBe(true)
    })
  })

  describe('accessibility', () => {
    it('should have proper ARIA attributes through semantic HTML', () => {
      const wrapper = mount(AlertError, {
        props: {
          text: 'Test error'
        }
      })

      const alertDiv = wrapper.find('div[data-testid="alert-error"]')
      expect(alertDiv.classes()).toContain('alert')
      expect(alertDiv.classes()).toContain('alert-error')
      // DaisyUI alert classes provide proper accessibility
    })

    it('should have readable text with proper sizing', () => {
      const wrapper = mount(AlertError, {
        props: {
          text: 'Test error'
        }
      })

      const alertDiv = wrapper.find('div[data-testid="alert-error"]')
      expect(alertDiv.classes()).toContain('text-lg')
      expect(alertDiv.classes()).toContain('font-bold')
    })
  })

  describe('edge cases', () => {
    it('should render with empty string', () => {
      const wrapper = mount(AlertError, {
        props: {
          text: ''
        }
      })

      // Should render with just a period
      expect(wrapper.text()).toBe('.')
    })

    it('should render with single character', () => {
      const wrapper = mount(AlertError, {
        props: {
          text: 'E'
        }
      })

      expect(wrapper.text()).toBe('E.')
    })

    it('should render with whitespace only', () => {
      const wrapper = mount(AlertError, {
        props: {
          text: '   '
        }
      })

      // Whitespace is preserved and period is added
      expect(wrapper.text()).toBeTruthy()
      expect(wrapper.text()).toContain('.')
    })

    it('should handle rapidly changing text', async () => {
      const wrapper = mount(AlertError, {
        props: {
          text: 'Error 1'
        }
      })

      expect(wrapper.text()).toContain('Error 1.')

      await wrapper.setProps({ text: 'Error 2' })
      expect(wrapper.text()).toContain('Error 2.')

      await wrapper.setProps({ text: 'Error 3' })
      expect(wrapper.text()).toContain('Error 3.')
    })
  })
})
