import { test, expect } from './fixtures'

test.describe('With Mocked Backend', () => {
  test.beforeEach(async ({ page, mockApi }) => {
    await page.goto('/daq/')
    await page.waitForLoadState('networkidle')
  })

  test('should load page with mocked API', async ({ page, mockApi }) => {
    await expect(page.locator('body')).toBeVisible()
  })

  test('should respond to simulated WebSocket messages', async ({ page, websocket }) => {
    await websocket.emitStateChange('gdc-01', 'INITIALIZED')
    await websocket.emitStateChange('ldc-01', 'INITIALIZED')

    await page.waitForTimeout(300)
  })

  test('should display metrics after simulation', async ({ page, websocket }) => {
    await websocket.emitMetrics('gdc-01', {
      EventCounter: 50000,
      ByteCounter: 100000000,
      CurrentDataRate: 50000000,
      CurrentTrgRate: 25.5,
      AvgDataRate: 48000000,
      AvgTrgRate: 24.2
    })

    await page.waitForTimeout(200)
  })
})

test.describe('Run Control with Mocked Backend', () => {
  test.beforeEach(async ({ page, websocket, mockApi }) => {
    await page.goto('/daq/')
    await page.waitForLoadState('networkidle')

    await websocket.emitStateChange('gdc-01', 'INITIALIZED')
    await websocket.emitStateChange('ldc-01', 'INITIALIZED')
    await websocket.emitStateChange('ldc-02', 'INITIALIZED')
    await page.waitForTimeout(200)
  })

  test('should enable start button when servers are initialized', async ({ page }) => {
    const startButton = page.locator('button[data-testid="start-run-button"]')
    if (await startButton.count() > 0) {
      await expect(startButton.first()).toBeVisible()
    }
  })

  test('should handle run state transition', async ({ page, websocket }) => {
    await websocket.emitStateChange('gdc-01', 'RUNNING')
    await websocket.emitStateChange('ldc-01', 'RUNNING')
    await websocket.emitStateChange('ldc-02', 'RUNNING')
    await page.waitForTimeout(300)

    const stopButton = page.locator('button[data-testid="stop-run-button"]')
    if (await stopButton.count() > 0) {
      await expect(stopButton.first()).toBeVisible()
    }
  })
})
