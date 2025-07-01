// Basic end-to-end tests verifying key UI functionality. These use
// Playwright's browser automation to simulate real user interactions.
const { test, expect } = require('@playwright/test');

test('loads homepage', async ({ page }) => {
  await page.goto('/');
  await expect(page.locator('h1')).toHaveText('Smart Music Go');
});

// This test simulates marking a track as a favorite from the UI. It navigates
// to the app, performs a search using the mock API and clicks the Favorite
// button. The track should then appear in the Favorites list.
test('adds item to favorites', async ({ page }) => {
  await page.goto('/app/');
  await page.locator('input[name="query"]').fill('song');
  await page.locator('button[type="submit"]').click();
  await page.locator('button:has-text("Favorite")').first().click();
  await page.click('text=Favorites');
  await expect(page.locator('li')).toContainText('song');
});
