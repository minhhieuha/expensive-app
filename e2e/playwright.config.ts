import { defineConfig } from '@playwright/test';

const PORT = process.env['PORT'] || '8088';

export default defineConfig({
  testDir: './tests',
  timeout: 30000,
  use: {
    baseURL: process.env['BASE_URL'] || `http://127.0.0.1:${PORT}`,
  },
  webServer: {
    command: 'go run ./cmd/server',
    cwd: '../api',
    env: {
      PORT: PORT,
    },
    url: `http://127.0.0.1:${PORT}/api/health`,
    reuseExistingServer: false,
    timeout: 30000,
  },
});
