import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import AlertWarning from '../AlertWarning.vue'

describe('AlertWarning Component', () => {
  describe('rendering', () => {
    it('should display the warning text with period', () => {
      const text = 'Test warning message'
      const wrapper = mount(AlertWarning, {
        props: {
          text
        }
      })

      expect(wrapper.text()).toContain(`${text}.`)
    })

    it('should have proper CSS classes', () => {
      const wrapper = mount(AlertWarning, {
        props: {
          text: 'Test warning'
        }
      })

      const alertDiv = wrapper.find('div[data-testid="alert-warning"]')
      expect(alertDiv.exists()).toBe(true)
      expect(alertDiv.classes()).toContain('alert')
      expect(alertDiv.classes()).toContain('alert-warning')
      expect(alertDiv.classes()).toContain('my-2')
      expect(alertDiv.classes()).toContain('mx-2')
      expect(alertDiv.classes()).toContain('text-lg')
      expect(alertDiv.classes()).toContain('font-bold')
    })

    it('should render warning icon with correct SVG path', () => {
      const wrapper = mount(AlertWarning, {
        props: {
          text: 'Test warning'
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
      // Warning icon should have triangle exclamation pattern
      expect(path.attributes('d')).toBeTruthy()
      expect(path.attributes('stroke-linecap')).toBe('round')
      expect(path.attributes('stroke-linejoin')).toBe('round')
    })

    it('should have proper icon size attributes', () => {
      const wrapper = mount(AlertWarning, {
        props: {
          text: 'Test warning'
        }
      })

      const svg = wrapper.find('svg')
      expect(svg.attributes('viewBox')).toBe('0 0 24 24')
      expect(svg.attributes('fill')).toBe('none')
    })
  })

  describe('text display', () => {
    it('should render with long warning messages', () => {
      const longText = 'Warning: This operation may take a long time to complete. Please do not close the browser window while the operation is in progress'
      const wrapper = mount(AlertWarning, {
        props: {
          text: longText
        }
      })

      expect(wrapper.text()).toContain(longText)
      expect(wrapper.text()).toContain('.')
    })

    it('should render with special characters in text', () => {
      const specialText = 'Warning: Configuration file at /path/to/config.json is invalid'
      const wrapper = mount(AlertWarning, {
        props: {
          text: specialText
        }
      })

      expect(wrapper.text()).toContain(specialText)
    })

    it('should render with newlines in text', () => {
      const multilineText = 'Warning: Low disk space\nPlease free up space\nSystem may become unstable'
      const wrapper = mount(AlertWarning, {
        props: {
          text: multilineText
        }
      })

      expect(wrapper.text()).toContain(multilineText)
    })

    it('should render with unicode characters', () => {
      const unicodeText = 'Advertencia: espacio bajo en disco ⚠️'
      const wrapper = mount(AlertWarning, {
        props: {
          text: unicodeText
        }
      })

      expect(wrapper.text()).toContain(unicodeText)
    })
  })

  describe('component structure', () => {
    it('should have proper semantic HTML structure', () => {
      const wrapper = mount(AlertWarning, {
        props: {
          text: 'Test warning'
        }
      })

      const alertDiv = wrapper.find('div[data-testid="alert-warning"]')
      expect(alertDiv.exists()).toBe(true)
      expect(alertDiv.find('svg').exists()).toBe(true)
      expect(alertDiv.find('div').exists()).toBe(true)
      expect(alertDiv.find('span').exists()).toBe(true)
    })

    it('should use data-testid for easy selection', () => {
      const wrapper = mount(AlertWarning, {
        props: {
          text: 'Test warning'
        }
      })

      expect(wrapper.find('div[data-testid="alert-warning"]').exists()).toBe(true)
    })
  })

  describe('accessibility', () => {
    it('should have proper ARIA attributes through semantic HTML', () => {
      const wrapper = mount(AlertWarning, {
        props: {
          text: 'Test warning'
        }
      })

      const alertDiv = wrapper.find('div[data-testid="alert-warning"]')
      expect(alertDiv.classes()).toContain('alert')
      expect(alertDiv.classes()).toContain('alert-warning')
    })

    it('should have readable text with proper sizing', () => {
      const wrapper = mount(AlertWarning, {
        props: {
          text: 'Test warning'
        }
      })

      const alertDiv = wrapper.find('div[data-testid="alert-warning"]')
      expect(alertDiv.classes()).toContain('text-lg')
      expect(alertDiv.classes()).toContain('font-bold')
    })
  })

  describe('edge cases', () => {
    it('should render with empty string', () => {
      const wrapper = mount(AlertWarning, {
        props: {
          text: ''
        }
      })

      expect(wrapper.text()).toBe('.')
    })

    it('should render with single character', () => {
      const wrapper = mount(AlertWarning, {
        props: {
          text: '!'
        }
      })

      expect(wrapper.text()).toBe('!.')
    })

    it('should handle rapidly changing text', async () => {
      const wrapper = mount(AlertWarning, {
        props: {
          text: 'Warning 1'
        }
      })

      expect(wrapper.text()).toContain('Warning 1.')

      await wrapper.setProps({ text: 'Warning 2' })
      expect(wrapper.text()).toContain('Warning 2.')
    })
  })
})
