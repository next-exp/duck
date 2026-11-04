import { test, expect } from '@playwright/test'

test('rate history renders, zooms with a detailed request, restores Auto and shows only the current run', async ({ page }) => {
  let fleetSize = 1
  const requests: Record<string, any>[] = []
  const errors: string[] = []
  page.on('pageerror', error => errors.push(error.message))
  const origin = Date.parse('2026-11-04T12:00:00Z')
  const end = origin + 3600000
  await page.route('**/apiService.DuckAPI/*', async route => {
    const method = route.request().url().split('/').pop()
    let body: any = {}
    if (method === 'GetRunNumber') body = { runNumber: 123 }
    else if (method === 'GetGDCs') body = { gdcs: Array.from({ length: fleetSize }, (_, i) => ({ id: i + 1, name: `gdc${i + 1}`, enabled: true })) }
    else if (method === 'GetLDCs') body = { ldcs: Array.from({ length: fleetSize }, (_, i) => ({ id: i + 1, name: `ldc${i + 1}`, enabled: true })) }
    else if (method === 'GetToken') body = { token: '' }
    else if (method === 'CheckDisabled') body = { warnings: {} }
    else if (method === 'GetRateHistory') {
      const request = route.request().postDataJSON()
      requests.push(request)
      if (request.run === 999) body = { run: 999, hasData: false, bins: [], gdcs: [] }
      else {
        const from = request.from ? Date.parse(request.from) : origin
        const to = request.to ? Date.parse(request.to) : end
        const width = Math.max(5, Math.ceil((to - from) / 1000 / Math.min(request.maxPoints ?? 1000, 300) / 5) * 5)
        const bins = []
        for (let start = from, i = 0; start < to; start += width * 1000, i++) {
          const complete = i !== 3
          bins.push({ start: new Date(start).toISOString(), end: new Date(Math.min(to, start + width * 1000)).toISOString(), ...(complete ? { triggerRateHz: i === 2 ? 0 : 120 + 20 * Math.sin(i / 10), byteRate: 50000 } : {}), complete, coverage: complete ? 1 : 0.5 })
        }
        body = { run: 123, hasData: true, from: new Date(from).toISOString(), to: new Date(to).toISOString(), latestSample: new Date(end).toISOString(), runStart: new Date(origin).toISOString(), binWidthSeconds: width, gdcs: ['gdc1', 'gdc2'], coverageBasis: 'observed_run_history', bins }
      }
    }
    await route.fulfill({ json: body, headers: { 'access-control-allow-origin': '*' } })
  })
  await page.goto('/daq/')
  const panel = page.getByTestId('rate-history-panel')
  await expect(panel.getByRole('heading', { name: 'Trigger rate history' })).toBeVisible()
  await expect(panel.locator('canvas')).toBeVisible()
  await expect(panel.getByText(/incomplete coverage/)).toBeVisible()
  await expect.poll(() => requests.length).toBeGreaterThan(0)
  await page.getByTestId('rate-history-chart').scrollIntoViewIfNeeded()
  const statisticsBox = await page.getByTestId('run-statistics-container').boundingBox()
  const historyBox = await panel.boundingBox()
  expect(statisticsBox!.x + statisticsBox!.width).toBeLessThan(historyBox!.x)
  expect(Math.abs(statisticsBox!.y - historyBox!.y)).toBeLessThan(16)
  await expect(panel.getByRole('button', { name: 'Back', exact: true })).toBeDisabled()
  await panel.getByRole('button', { name: 'Select range', exact: true }).click()
  await expect(panel.getByRole('button', { name: 'Cancel selection', exact: true })).toHaveAttribute('aria-pressed', 'true')
  const selectionBox = await page.getByTestId('rate-history-canvas').boundingBox()
  if (!selectionBox) throw new Error('Selection canvas not visible')
  await page.mouse.move(selectionBox.x + selectionBox.width * 0.25, selectionBox.y + 120)
  await page.mouse.down()
  await page.mouse.move(selectionBox.x + selectionBox.width * 0.65, selectionBox.y + 120, { steps: 12 })
  await page.mouse.up()
  await expect(panel.getByRole('button', { name: 'Manual', exact: true })).toHaveAttribute('aria-pressed', 'true')
  await expect.poll(() => requests.some(request => request.from && request.to)).toBe(true)
  await panel.getByRole('button', { name: 'Back', exact: true }).click()
  await expect(panel.getByRole('button', { name: 'Auto', exact: true })).toHaveAttribute('aria-pressed', 'true')
  await expect(panel.getByRole('button', { name: 'Back', exact: true })).toBeDisabled()
  await page.getByTestId('rate-history-canvas').scrollIntoViewIfNeeded()
  const beforeZoom = requests.length
  const box = await page.getByTestId('rate-history-chart').boundingBox()
  if (!box) throw new Error('Chart not visible')
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2)
  await page.keyboard.down('Control')
  await page.mouse.wheel(0, -400)
  await page.keyboard.up('Control')
  await expect(panel.getByRole('button', { name: 'Manual', exact: true })).toHaveAttribute('aria-pressed', 'true')
  await expect.poll(() => requests.slice(beforeZoom).some(request => request.from && request.to && Date.parse(request.to) - Date.parse(request.from) < end - origin)).toBe(true)
  await panel.getByLabel('From (UTC)').fill('2026-11-04T12:15')
  await panel.getByLabel('To (UTC)').fill('2026-11-04T12:30')
  await panel.getByRole('button', { name: 'Apply window', exact: true }).click()
  await expect.poll(() => requests.some(request => request.from === '2026-11-04T12:15:00Z' && request.to === '2026-11-04T12:30:00Z')).toBe(true)
  await panel.getByRole('button', { name: 'Auto', exact: true }).click()
  await expect.poll(() => requests[requests.length - 1]?.from).toBeUndefined()
  await expect(panel.getByRole('button', { name: 'Auto', exact: true })).toHaveAttribute('aria-pressed', 'true')
  await page.setViewportSize({ width: 390, height: 844 })
  await expect(panel.locator('canvas')).toBeVisible()
  const mobileStatistics = await page.getByTestId('run-statistics-container').boundingBox()
  const mobileHistory = await panel.boundingBox()
  expect(mobileHistory!.y).toBeGreaterThanOrEqual(mobileStatistics!.y + mobileStatistics!.height)
  expect((await panel.boundingBox())!.width).toBeLessThanOrEqual(390)
  await page.screenshot({ path: '/tmp/duck-rate-history-mobile.png', fullPage: true })
  await page.setViewportSize({ width: 1440, height: 1000 })
  await expect(panel.getByRole('spinbutton')).toHaveCount(0)
  await expect(panel.getByRole('button', { name: 'Load', exact: true })).toHaveCount(0)
  await expect(panel.getByRole('button', { name: 'Live run', exact: true })).toHaveCount(0)
  await expect(panel.getByText('Run 123', { exact: true })).toBeVisible()
  await page.screenshot({ path: '/tmp/duck-rate-history-desktop.png', fullPage: true })
  fleetSize = 2
  await page.reload()
  await expect(panel.locator('canvas')).toBeVisible()
  const hdStatistics = await page.getByTestId('run-statistics-container').boundingBox()
  const hdHistory = await panel.boundingBox()
  expect(hdStatistics!.x + hdStatistics!.width).toBeLessThan(hdHistory!.x)
  expect(hdHistory!.width).toBeGreaterThan(450)
  fleetSize = 7
  await page.reload()
  await expect(panel.locator('canvas')).toBeVisible()
  const largeStatistics = await page.getByTestId('run-statistics-container').boundingBox()
  const largeHistory = await panel.boundingBox()
  expect(largeHistory!.y).toBeGreaterThanOrEqual(largeStatistics!.y + largeStatistics!.height)
  expect(errors).toEqual([])
})
