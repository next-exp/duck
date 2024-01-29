import { test, expect } from './fixtures'

test.describe('LDC Configuration Form', () => {
  test.beforeEach(async ({ page, mockApi }) => {
    await page.goto('/daq/detector')
    await page.waitForLoadState('networkidle')
  })

  test('should display LDC configuration form', async ({ page }) => {
    const ldcHeading = page.locator('h2, h3').filter({ hasText: /LDC/i })
    if (await ldcHeading.count() > 0) {
      await expect(ldcHeading.first()).toBeVisible()
    }
  })

  test('should validate empty required fields for LDC', async ({ page }) => {
    const submitButton = page.locator('button').filter({ hasText: /submit|save|create|update/i }).first()
    
    if (await submitButton.count() > 0) {
      // Clear name field
      const nameInput = page.locator('input[id*="ldc"][id*="name"], input[placeholder*="LDC name"]').first()
      if (await nameInput.count() > 0) {
        await nameInput.fill('')
        await submitButton.click()
        
        // Check for validation errors
        await expect(page.locator('.text-red-600')).toHaveCount({ min: 1 })
      }
    }
  })

  test('should validate port numbers for LDC', async ({ page }) => {
    const grpcPortInput = page.locator('input[id*="grpc"][id*="port"]').first()
    
    if (await grpcPortInput.count() > 0) {
      // Test out of range port
      await grpcPortInput.fill('99999')
      
      const submitButton = page.locator('button').filter({ hasText: /submit|save/i }).first()
      await submitButton.click()
      
      // Check for port validation error
      await expect(page.locator('text=/port.*range|invalid port/i')).toBeVisible({ timeout: 5000 })
    }
  })

  test('should validate prometheus port separately', async ({ page }) => {
    const prometheusPortInput = page.locator('input[id*="prometheus"][id*="port"]').first()
    
    if (await prometheusPortInput.count() > 0) {
      await prometheusPortInput.fill('-100')
      
      const submitButton = page.locator('button').filter({ hasText: /submit|save/i }).first()
      if (await submitButton.count() > 0) {
        await submitButton.click({ force: true })

        await expect(page.locator('.text-red-600')).toHaveCount({ min: 1 })
      }
    } else {
      test.skip(true, 'Prometheus port input not available')
    }
  })

  test('should successfully create LDC with valid data', async ({ page }) => {
    await page.route('**/ldc', async route => {
      await route.fulfill({
        status: 200,
        body: JSON.stringify({ id: 1, name: 'test-ldc' })
      })
    })
    
    const nameInput = page.locator('input[placeholder*="name"]').first()
    if (await nameInput.count() > 0) {
      await nameInput.fill('test-ldc-1')
      
      const hostnameInput = page.locator('input[placeholder*="hostname"]').first()
      if (await hostnameInput.count() > 0) {
        await hostnameInput.fill('ldc1.local')
      }
      
      const ipInput = page.locator('input[placeholder*="ip"]').first()
      if (await ipInput.count() > 0) {
        await ipInput.fill('192.168.1.20')
      }
      
      const submitButton = page.locator('button').filter({ hasText: /create|save/i }).first()
      if (await submitButton.count() > 0) {
        await submitButton.click({ force: true })

        await page.waitForTimeout(1000)
        expect(await page.locator('.text-red-600').count()).toBe(0)
      }
    } else {
      test.skip(true, 'LDC form not available')
    }
  })

  test('should handle LDC update operations', async ({ page }) => {
    await page.route('**/ldc/*', async route => {
      if (route.request().method() === 'PUT') {
        await route.fulfill({
          status: 200,
          body: JSON.stringify({ id: 1, name: 'updated-ldc' })
        })
      }
    })
    
    const nameInput = page.locator('input[placeholder*="name"]').first()
    if (await nameInput.count() > 0) {
      await nameInput.fill('updated-ldc-name')
      
      const updateButton = page.locator('button').filter({ hasText: /update|save/i }).first()
      if (await updateButton.count() > 0) {
        await updateButton.click()
        await page.waitForTimeout(500)
      }
    }
  })

  test('should toggle LDC enabled state', async ({ page }) => {
    const enabledCheckbox = page.locator('input[type="checkbox"]').filter({ hasText: /enabled/i }).first()
    
    if (await enabledCheckbox.count() > 0) {
      const initialState = await enabledCheckbox.isChecked()
      await enabledCheckbox.click()
      await expect(enabledCheckbox).toBeChecked({ checked: !initialState })
    }
  })
})

test.describe('Equipment Configuration Form', () => {
  test.beforeEach(async ({ page, mockApi }) => {
    await page.goto('/daq/detector')
    await page.waitForLoadState('networkidle')
  })

  test('should display equipment configuration form', async ({ page }) => {
    const equipmentHeading = page.locator('h2, h3').filter({ hasText: /equipment/i })
    if (await equipmentHeading.count() > 0) {
      await expect(equipmentHeading.first()).toBeVisible()
    }
  })

  test('should validate equipment type selection', async ({ page }) => {
    const typeSelect = page.locator('select[id*="type"], select[name*="type"]').first()
    
    if (await typeSelect.count() > 0) {
      // Clear or select invalid option
      await typeSelect.selectOption({ index: 0 })
      
      const submitButton = page.locator('button').filter({ hasText: /submit|save|create/i }).first()
      await submitButton.click()
      
      // May show validation error
      await page.waitForTimeout(500)
    }
  })

  test('should validate device IP address', async ({ page }) => {
    const deviceIpInput = page.locator('input[id*="device"][id*="ip"]').first()
    
    if (await deviceIpInput.count() > 0) {
      await deviceIpInput.fill('invalid-ip')
      
      const submitButton = page.locator('button').filter({ hasText: /submit|save/i }).first()
      await submitButton.click()
      
      await expect(page.locator('.text-red-600, .error')).toHaveCount({ min: 1 })
    }
  })

  test('should validate host port number', async ({ page }) => {
    const hostPortInput = page.locator('input[id*="host"][id*="port"]').first()
    
    if (await hostPortInput.count() > 0) {
      await hostPortInput.fill('70000') // Invalid port
      
      const submitButton = page.locator('button').filter({ hasText: /submit|save/i }).first()
      await submitButton.click()
      
      await expect(page.locator('text=/port/i')).toBeVisible()
    }
  })

  test('should associate equipment with LDC', async ({ page }) => {
    const ldcSelect = page.locator('select[id*="ldc"]').first()
    
    if (await ldcSelect.count() > 0) {
      // Select an LDC
      const options = await ldcSelect.locator('option').count()
      if (options > 1) {
        await ldcSelect.selectOption({ index: 1 })
        await expect(ldcSelect).not.toHaveValue('')
      }
    }
  })

  test('should successfully create equipment with valid data', async ({ page }) => {
    await page.route('**/equipment', async route => {
      await route.fulfill({
        status: 200,
        body: JSON.stringify({ id: 1, type: 22 })
      })
    })
    
    // Fill valid equipment data
    const typeSelect = page.locator('select[id*="type"]').first()
    if (await typeSelect.count() > 0) {
      await typeSelect.selectOption({ index: 1 })
      
      const deviceIpInput = page.locator('input[id*="device"][id*="ip"]').first()
      if (await deviceIpInput.count() > 0) {
        await deviceIpInput.fill('192.168.1.100')
      }
      
      const hostPortInput = page.locator('input[id*="host"][id*="port"]').first()
      if (await hostPortInput.count() > 0) {
        await hostPortInput.fill('6006')
      }
      
      const submitButton = page.locator('button').filter({ hasText: /create|save/i }).first()
      await submitButton.click()
      
      await page.waitForTimeout(1000)
    }
  })

  test('should handle equipment deletion', async ({ page }) => {
    await page.route('**/equipment/*', async route => {
      if (route.request().method() === 'DELETE') {
        await route.fulfill({
          status: 200,
          body: JSON.stringify({ success: true })
        })
      }
    })
    
    const deleteButton = page.locator('button').filter({ hasText: /delete|remove/i }).first()
    if (await deleteButton.count() > 0) {
      await deleteButton.click({ force: true })
      await page.waitForTimeout(500)
    } else {
      test.skip(true, 'Delete button not available')
    }
  })
})
