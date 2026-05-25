const { expect, test } = require('@playwright/test');

async function gotoOK(page, path) {
  const response = await page.goto(path);
  expect(response, `${path} should return a response`).not.toBeNull();
  expect(response.ok(), `${path} should respond successfully`).toBe(true);
}

test('dashboard renders', async ({ page }) => {
  await gotoOK(page, '/');

  await expect(page.getByRole('heading', { name: 'Dashboard', exact: true })).toBeVisible();
  await expect(page.getByRole('navigation')).toContainText('Invoice App');
});

const pages = [
  { path: '/entries', heading: 'Entries', content: 'Add Entry' },
  { path: '/rates', heading: 'Rates', content: 'Add Rate' },
  { path: '/invoices', heading: 'Invoices', content: 'Generate Invoice' },
  { path: '/settings', heading: 'Settings', content: 'Business Details' },
];

for (const pageSpec of pages) {
  test(`${pageSpec.heading} page renders`, async ({ page }) => {
    await gotoOK(page, pageSpec.path);

    await expect(page.getByRole('heading', { name: pageSpec.heading, exact: true })).toBeVisible();
    await expect(page.getByRole('heading', { name: pageSpec.content, exact: true })).toBeVisible();
  });
}
