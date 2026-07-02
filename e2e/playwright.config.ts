import { defineConfig, devices } from '@playwright/test'

const baseURL = process.env.E2E_BASE_URL ?? 'http://localhost'
const isCI = Boolean(process.env.CI)
const parallelMode = process.env.E2E_PARALLEL === '1'

export default defineConfig({
  testDir: './tests',
  fullyParallel: parallelMode,
  forbidOnly: isCI,
  retries: isCI ? 2 : 0,
  workers: parallelMode ? 3 : 1,
  timeout: 60_000,
  expect: { timeout: 15_000 },
  reporter: isCI
    ? [['github'], ['html', { open: 'never' }], ['list']]
    : [['list'], ['html', { open: 'never' }]],
  use: {
    baseURL,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    locale: 'ja-JP',
    actionTimeout: 15_000,
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
})
