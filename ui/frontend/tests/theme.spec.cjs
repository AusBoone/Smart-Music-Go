// Tests for theme switching functionality using Playwright.
const { test, expect } = require('@playwright/test');

test('toggles dark mode', async ({ page }) => {
  await page.goto('/app/');
  await page.locator('button:has-text("Toggle Theme")').click();
  await expect(page.locator('.App.dark')).toBeVisible();
});
