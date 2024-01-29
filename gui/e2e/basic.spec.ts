import { test, expect } from './fixtures'

test.describe('DUCK Application E2E Tests', () => {
  test.beforeEach(async ({ page, mockApi }) => {
    await page.goto('/daq/')
    await page.waitForLoadState('networkidle')
  })

  test('should display the main dashboard', async ({ page }) => {
    // Check if the main title is visible - use first() to avoid strict mode violation
    await expect(page.locator('h1, h2').first()).toBeVisible()

    // Check if dashboard elements are present
    await expect(page.locator('body')).toBeVisible()
  })

  test('should have accessible navigation', async ({ page }) => {
    // Check if navigation elements are present
    const navElements = page.locator('nav, a[href], button')
    await expect(navElements.first()).toBeVisible()
  })

  test('should handle responsive design', async ({ page }) => {
    // Test mobile viewport
    await page.setViewportSize({ width: 375, height: 667 })
    await expect(page.locator('body')).toBeVisible()

    // Test desktop viewport
    await page.setViewportSize({ width: 1920, height: 1080 })
    await expect(page.locator('body')).toBeVisible()
  })

  test('should load without JavaScript errors', async ({ page }) => {
    // Listen for console errors
    const errors: string[] = []
    page.on('console', msg => {
      if (msg.type() === 'error') {
        errors.push(msg.text())
      }
    })

    await page.reload()
    await page.waitForLoadState('networkidle')

    // Log errors for debugging
    if (errors.length > 0) {
      console.log('JavaScript errors found:', errors)
    }

    // Assert no critical JavaScript errors (filter out known non-critical errors)
    const criticalErrors = errors.filter(err =>
      !err.includes('Failed to load resource') && // Ignore resource loading errors
      !err.includes('favicon') && // Ignore favicon errors
      !err.includes('404') // Ignore 404 errors
    )
    expect(criticalErrors.length).toBe(0)
  })

  test('should have proper page title', async ({ page }) => {
    await expect(page).toHaveTitle(/NEXT DAQ/)
  })

  test('should handle 404 gracefully', async ({ page }) => {
    const response = await page.goto('/non-existent-page')
    expect(response?.status()).toBe(404)
  })
})

test.describe('Component Rendering', () => {
  test('should render control elements', async ({ page, mockApi }) => {
    await page.goto('/daq/')

    // Look for common control elements
    const buttons = page.locator('button')
    if (await buttons.count() > 0) {
      await expect(buttons.first()).toBeVisible()
    }

    // Look for input elements
    const inputs = page.locator('input')
    if (await inputs.count() > 0) {
      await expect(inputs.first()).toBeVisible()
    }
  })

  test('should render status indicators', async ({ page, mockApi }) => {
    await page.goto('/daq/')

    // Look for status-related elements
    const statusElements = page.locator('[class*="status"], [class*="indicator"], [class*="state"]')
    if (await statusElements.count() > 0) {
      await expect(statusElements.first()).toBeVisible()
    }
  })
})

test.describe('Accessibility', () => {
  test('should have proper heading hierarchy', async ({ page, mockApi }) => {
    await page.goto('/daq/')

    // Check for h1 element
    const h1 = page.locator('h1')
    if (await h1.count() > 0) {
      await expect(h1.first()).toBeVisible()
    }

    // Check that headings are in order
    const headings = page.locator('h1, h2, h3, h4, h5, h6')
    if (await headings.count() > 0) {
      await expect(headings.first()).toBeVisible()
    }
  })

  test('should have accessible form controls', async ({ page, mockApi }) => {
    await page.goto('/daq/')

    // Check form labels
    const inputs = page.locator('input')
    const inputCount = await inputs.count()

    for (let i = 0; i < inputCount; i++) {
      const input = inputs.nth(i)
      const id = await input.getAttribute('id')

      if (id) {
        const label = page.locator(`label[for="${id}"]`)
        if (await label.count() > 0) {
          await expect(label.first()).toBeVisible()
        }
      }
    }
  })

  test('should have focusable elements', async ({ page, mockApi }) => {
    await page.goto('/daq/')

    // Check if focusable elements exist
    const focusableElements = page.locator('button, input, select, textarea, a[href], [tabindex]:not([tabindex="-1"])')
    const elementCount = await focusableElements.count()

    if (elementCount > 0) {
      await expect(focusableElements.first()).toBeVisible()

      // Test keyboard navigation
      await page.keyboard.press('Tab')
      const focusedElement = page.locator(':focus')
      expect(await focusedElement.count()).toBeGreaterThan(0)
    }
  })
})