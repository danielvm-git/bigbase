import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: '.',
  timeout: 30000,
  retries: 1,
  use: {
    baseURL: process.env.BASE_URL || 'http://localhost:9999',
    trace: 'on-first-retry',
  },
  projects: [
    { name: 'chromium', use: { browserName: 'chromium' } },
  ],
  webServer: {
    command: 'go run . serve --port 9999 --db /tmp/bigbase-e2e.db',
    url: 'http://localhost:9999/health',
    reuseExistingServer: !process.env.CI,
    timeout: 120000,
    cwd: process.cwd(),
  },
});
