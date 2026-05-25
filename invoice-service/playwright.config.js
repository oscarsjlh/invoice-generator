const { defineConfig, devices } = require('@playwright/test');

const authBaseURL = process.env.PLAYWRIGHT_AUTH_BASE_URL || 'http://127.0.0.1:18080';
const noAuthBaseURLs = (process.env.PLAYWRIGHT_NOAUTH_BASE_URLS || 'http://127.0.0.1:18081,http://127.0.0.1:18082')
  .split(',')
  .map((url) => url.trim())
  .filter(Boolean);
const skipWebServer = process.env.PLAYWRIGHT_SKIP_WEBSERVER === '1';
const workers = Number(process.env.PLAYWRIGHT_WORKERS || (process.env.CI ? 2 : 2));

module.exports = defineConfig({
  testDir: './tests/e2e',
  timeout: 45_000,
  expect: {
    timeout: 5_000,
  },
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers,
  reporter: process.env.CI
    ? [['list'], ['html', { open: 'never', outputFolder: 'playwright-report' }], ['json', { outputFile: 'test-results/results.json' }]]
    : [['list'], ['html', { open: 'never', outputFolder: 'playwright-report' }]],
  use: {
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  outputDir: 'test-results',
  projects: [
    {
      name: 'auth-on-chromium',
      testMatch: /.*\.auth\.spec\.js/,
      use: { ...devices['Desktop Chrome'], baseURL: authBaseURL, mode: 'auth-on' },
    },
    {
      name: 'noauth-chromium',
      testMatch: /.*\.noauth\.spec\.js/,
      use: { ...devices['Desktop Chrome'], baseURL: noAuthBaseURLs[0], noAuthBaseURLs, mode: 'noauth' },
    },
  ],
  webServer: skipWebServer
    ? undefined
    : {
        command: 'docker compose -f ../docker-compose.e2e.yml up --build app-auth-on app-noauth-1 app-noauth-2',
        url: `${authBaseURL}/health`,
        reuseExistingServer: false,
        timeout: 120_000,
        stdout: 'pipe',
        stderr: 'pipe',
      },
});
