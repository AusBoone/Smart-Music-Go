// Ensures view changes apply the fade-in animation class.
const { test, expect } = require('@playwright/test');

test('applies fade-in class on navigation', async ({ page }) => {
  await page.goto('/app/');
  await page.click('text=Playlists');
  await expect(page.locator('.App.fade-in')).toBeVisible();
});
