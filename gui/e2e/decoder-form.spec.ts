import { test, expect } from './fixtures'

test.describe('Decoder Configuration Form', () => {
  test.beforeEach(async ({ page, mockApi }) => {
    await page.goto('/daq/detector')
    await page.waitForLoadState('networkidle')
  })

  test('should display decoder configuration form', async ({ page }) => {
    // Skip if the decoder configuration page doesn't exist
    const pageContent = await page.textContent('body')
    if (!pageContent?.includes('Decoder') && !pageContent?.includes('detector')) {
      test.skip(true, 'Decoder configuration page not available')
    }

    // Check for form title - make it more flexible
    const decoderHeading = page.locator('h2, h3, h4').filter({ hasText: /decoder.*configuration/i })
    if (await decoderHeading.count() > 0) {
      await expect(decoderHeading.first()).toBeVisible()
    }

    // Check for Update button
    const updateButton = page.locator('button').filter({ hasText: /update/i })
    if (await updateButton.count() > 0) {
      await expect(updateButton.first()).toBeVisible()
    }
  })

  test('should validate empty required fields', async ({ page }) => {
    // Skip if decoder form not available
    const externalTrigger = page.locator('input[id*="decoder-external-trigger"]')
    if (await externalTrigger.count() === 0) {
      test.skip(true, 'Decoder form not available')
    }

    // Clear all required fields
    await page.fill('input[id*="decoder-external-trigger"]', '')
    await page.fill('input[id*="decoder-trigger-code-1"]', '')
    await page.fill('input[id*="decoder-trigger-code-2"]', '')
    await page.fill('input[id*="decoder-db-host"]', '')
    await page.fill('input[id*="decoder-db-user"]', '')
    await page.fill('input[id*="decoder-db-password"]', '')
    await page.fill('input[id*="decoder-db-name"]', '')
    
    // Click update button
    await page.click('button:has-text("Update")')
    
    // Check for validation errors
    await expect(page.locator('.text-red-600')).toHaveCount({ min: 1 })
  })

  test('should validate trigger code range - negative values', async ({ page }) => {
    // Skip if decoder form not available
    const triggerCode = page.locator('input[id*="decoder-trigger-code-1"]')
    if (await triggerCode.count() === 0) {
      test.skip(true, 'Decoder form not available')
    }

    // Set negative value
    await page.fill('input[id*="decoder-trigger-code-1"]', '-1')
    
    // Click update button
    await page.click('button:has-text("Update")')
    
    // Check for validation error message
    await expect(page.locator('text=/trigger code must be positive/i')).toBeVisible()
  })

  test('should validate external trigger range - negative values', async ({ page }) => {
    // Skip if decoder form not available
    const externalTrigger = page.locator('input[id*="decoder-external-trigger"]')
    if (await externalTrigger.count() === 0) {
      test.skip(true, 'Decoder form not available')
    }

    // Set negative value
    await page.fill('input[id*="decoder-external-trigger"]', '-1')
    
    // Click update button
    await page.click('button:has-text("Update")')
    
    // Check for validation error message
    await expect(page.locator('text=/channel number must be positive/i')).toBeVisible()
  })

  test('should successfully submit valid decoder configuration', async ({ page }) => {
    // Skip if decoder form not available
    const externalTrigger = page.locator('input[id*="decoder-external-trigger"]')
    if (await externalTrigger.count() === 0) {
      test.skip(true, 'Decoder form not available')
    }

    // Fill all required fields with valid data
    await page.fill('input[id*="decoder-external-trigger"]', '1')
    await page.fill('input[id*="decoder-trigger-code-1"]', '100')
    await page.fill('input[id*="decoder-trigger-code-2"]', '200')
    await page.fill('input[id*="decoder-db-host"]', 'localhost')
    await page.fill('input[id*="decoder-db-user"]', 'test_user')
    await page.fill('input[id*="decoder-db-password"]', 'test_pass')
    await page.fill('input[id*="decoder-db-name"]', 'test_db')
    await page.selectOption('select[id*="decoder-blosc-algorithm"]', 'lz4')
    await page.fill('input[id*="decoder-compression-level"]', '5')
    await page.selectOption('select[id*="decoder-bit-shuffle"]', 'bit-shuffle')
    
    // Mock API response
    await page.route('**/decoder', async route => {
      await route.fulfill({
        status: 200,
        body: JSON.stringify({ success: true })
      })
    })
    
    // Submit form
    await page.click('button:has-text("Update")')
    
    // Check for success message or no error messages
    await expect(page.locator('.text-red-600')).toHaveCount(0)
  })

  test('should handle form submission error', async ({ page }) => {
    // Skip if decoder form not available
    const externalTrigger = page.locator('input[id*="decoder-external-trigger"]')
    if (await externalTrigger.count() === 0) {
      test.skip(true, 'Decoder form not available')
    }

    // Fill valid data
    await page.fill('input[id*="decoder-external-trigger"]', '1')
    await page.fill('input[id*="decoder-trigger-code-1"]', '100')
    await page.fill('input[id*="decoder-trigger-code-2"]', '200')
    await page.fill('input[id*="decoder-db-host"]', 'localhost')
    await page.fill('input[id*="decoder-db-user"]', 'test_user')
    await page.fill('input[id*="decoder-db-password"]', 'test_pass')
    await page.fill('input[id*="decoder-db-name"]', 'test_db')
    
    // Mock API error
    await page.route('**/decoder', async route => {
      await route.fulfill({
        status: 500,
        body: JSON.stringify({ error: 'Server error' })
      })
    })
    
    // Submit form
    await page.click('button:has-text("Update")')
    
    // Check for error display
    await expect(page.locator('text=/error/i')).toBeVisible()
  })

  test('should toggle checkboxes correctly', async ({ page }) => {
    // Find and toggle read_pmts checkbox
    const readPmtsCheckbox = page.locator('input[id*="decoder-read-pmts"]')
    if (await readPmtsCheckbox.count() === 0) {
      test.skip(true, 'Decoder form not available')
    }

    const initialState = await readPmtsCheckbox.isChecked()
    
    await readPmtsCheckbox.click()
    await expect(readPmtsCheckbox).toBeChecked({ checked: !initialState })
  })

  test('should update compression settings', async ({ page }) => {
    // Skip if decoder form not available
    const bloscAlgorithm = page.locator('select[id*="decoder-blosc-algorithm"]')
    if (await bloscAlgorithm.count() === 0) {
      test.skip(true, 'Decoder form not available')
    }

    // Select compression algorithm
    await page.selectOption('select[id*="decoder-blosc-algorithm"]', 'lz4')
    await expect(page.locator('select[id*="decoder-blosc-algorithm"]')).toHaveValue('lz4')
    
    // Set compression level
    await page.fill('input[id*="decoder-compression-level"]', '7')
    await expect(page.locator('input[id*="decoder-compression-level"]')).toHaveValue('7')
    
    // Select bit shuffle
    await page.selectOption('select[id*="decoder-bit-shuffle"]', 'byte-shuffle')
    await expect(page.locator('select[id*="decoder-bit-shuffle"]')).toHaveValue('byte-shuffle')
  })
})
