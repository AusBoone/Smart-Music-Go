const { test, expect } = require('@playwright/test');

test('loads homepage', async ({ page }) => {
  await page.goto('/');
  await expect(page.locator('h1')).toHaveText('Smart Music Go');
});
