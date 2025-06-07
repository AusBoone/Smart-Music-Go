// @ts-check
const { defineConfig } = require('@playwright/test');

module.exports = defineConfig({
  testDir: './tests',
  webServer: {
    command: 'npm run preview -- --port=4173',
    port: 4173,
    cwd: __dirname,
    timeout: 120 * 1000,
    reuseExistingServer: true,
  },
  use: { baseURL: 'http://localhost:4173' },
});
