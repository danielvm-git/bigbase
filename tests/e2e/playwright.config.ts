import path from 'path';
import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: '.',
  timeout: 30000,
  retries: 1,
  // Tests share one isolated SQLite server and the auth rate limiter. A single
  // worker keeps registration deterministic and prevents cross-test 429s.
  workers: 1,
  use: {
    baseURL: process.env.BASE_URL || 'http://localhost:9999',
    trace: 'on-first-retry',
  },
  projects: [
    { name: 'chromium', use: { browserName: 'chromium' } },
  ],
  webServer: {
    command: 'BIGBASE_ENV=development BIGBASE_ALLOW_PLAINTEXT_SECRETS=true BIGBASE_ROOT_ENCRYPTION_KEY=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA= go run . serve --port 9999 --db /tmp/bigbase-e2e.db --jwt-access-expiry=30s --jwt-refresh-expiry=60s',
    url: 'http://localhost:9999/health',
    reuseExistingServer: !process.env.CI,
    timeout: 120000,
    cwd: path.resolve(__dirname, '../..'),
  },
});
