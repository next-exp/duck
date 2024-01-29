import { defineConfig, devices } from '@playwright/test'

/**
 * Playwright configuration for MOCKED E2E tests
 *
 * These tests use custom fixtures (MSW + Centrifuge) and run without a real backend.
 */
export default defineConfig({
  testDir: './e2e',
  // Only run tests that use the custom fixtures
  testMatch: '**/run-control-statistics.spec.ts',
  /* Run tests in files in parallel */
  fullyParallel: true,
  /* Fail the build on CI if you accidentally left test.only in the source code. */
  forbidOnly: !!process.env.CI,
  /* Retry on CI only */
  retries: process.env.CI ? 2 : 0,
  /* Opt out of parallel tests on CI. */
  workers: process.env.CI ? 1 : undefined,
  /* Reporter to use. */
  reporter: 'html',
  /* Shared settings for all the projects below. */
  use: {
    /* Base URL to use in actions like `await page.goto('/')`. */
    baseURL: 'http://localhost:4173',

    /* Collect trace when retrying the failed test. */
    trace: 'on-first-retry',

    /* Take a screenshot on failure */
    screenshot: 'only-on-failure',

    /* Record video on failure */
    video: 'retain-on-failure',

    /* Global timeout for each action */
    actionTimeout: 10000,

    /* Global timeout for navigation */
    navigationTimeout: 30000
  },

  /* Configure projects for major browsers */
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  /* Run your local dev server before starting the tests */
  webServer: {
    command: 'npm run build-only && npm run preview',
    url: 'http://localhost:4173',
    reuseExistingServer: !process.env.CI,
    timeout: 120000,
  },

  /* Global test timeout */
  globalTimeout: process.env.CI ? 600000 : 300000,
})
