import { defineConfig, devices } from '@playwright/test'

/**
 * Playwright configuration for INTEGRATION tests
 *
 * These tests run against a REAL backend using Docker Compose.
 * The backend must be started before running these tests.
 *
 * See: e2e-integration/README.md
 */
export default defineConfig({
  testDir: './e2e-integration',
  /* Run integration tests sequentially - they share real backend state */
  fullyParallel: false,
  /* Fail the build on CI if you accidentally left test.only in the source code. */
  forbidOnly: !!process.env.CI,
  /* Retry on CI only */
  retries: process.env.CI ? 1 : 0,
  /* Integration tests run one at a time */
  workers: 1,
  /* Reporter to use. */
  reporter: [
    ['html', { outputFolder: 'playwright-integration-report' }],
    ['list']
  ],
  /* Shared settings for all the projects below. */
  use: {
    /* Base URL to use in actions like `await page.goto('/')`. */
    baseURL: 'http://localhost:4173',

    /* Collect trace on failure for integration tests */
    trace: 'retain-on-failure',

    /* Take a screenshot on failure */
    screenshot: 'only-on-failure',

    /* Record video on failure */
    video: 'retain-on-failure',

    /* Longer timeouts for integration tests (real backend) */
    actionTimeout: 15000,
    navigationTimeout: 60000,
  },

  /* Run your local dev server before starting the tests */
  /* Backend is started by docker-compose, this only serves the frontend */
  webServer: {
    command: 'npm run preview',
    url: 'http://localhost:4173',
    reuseExistingServer: !process.env.CI,
    timeout: 60000,
  },

  /* Configure projects for major browsers - integration tests run on Chromium by default */
  projects: [
    {
      name: 'chromium-integration',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  /* Do NOT run webServer for integration tests - backend must be started manually */
  /* This ensures tests run against the real Docker stack */

  /* Global test timeout - integration tests can be slower */
  globalTimeout: process.env.CI ? 900000 : 600000,
})
