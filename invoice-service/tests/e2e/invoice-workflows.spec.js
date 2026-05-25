const { expect, test } = require('@playwright/test');

function uniqueCategory(prefix) {
  return `${prefix} ${Date.now()} ${Math.random().toString(16).slice(2, 8)}`;
}

async function waitForCSRF(page) {
  await expect(page.locator('input[name="csrf_token"]').first()).not.toHaveValue('');
}

async function createEntry(page, { date, category, hours, notes }) {
  await page.goto('/entries');
  await waitForCSRF(page);

  await page.getByLabel('Date').fill(date);
  await page.getByLabel('Category').fill(category);
  await page.getByLabel('Hours').fill(hours);
  await page.getByLabel('Notes').fill(notes);

  await Promise.all([
    page.waitForURL('**/entries?notice=*'),
    page.getByRole('button', { name: 'Save Entry' }).click(),
  ]);
}

async function createRate(page, { category, startDate, endDate, rate }) {
  await page.goto('/rates');
  await waitForCSRF(page);

  await page.getByLabel('Category').fill(category);
  await page.getByLabel('Start Date').fill(startDate);
  await page.getByLabel('End Date').fill(endDate);
  await page.getByLabel('Hourly Rate').fill(rate);

  await Promise.all([
    page.waitForURL('**/rates?notice=*'),
    page.getByRole('button', { name: 'Save Rate' }).click(),
  ]);
}

test('creates a time entry from the browser', async ({ page }) => {
  const category = uniqueCategory('Entry E2E');
  const notes = `Browser-created entry ${category}`;

  await createEntry(page, {
    date: '2026-05-04',
    category,
    hours: '7.5',
    notes,
  });

  await expect(page.getByText('Entry added')).toBeVisible();
  await expect(page.getByRole('cell', { name: category, exact: true })).toBeVisible();
  await expect(page.getByRole('cell', { name: notes, exact: true })).toBeVisible();
});

test('creates a rate from the browser', async ({ page }) => {
  const category = uniqueCategory('Rate E2E');

  await createRate(page, {
    category,
    startDate: '2026-05-01',
    endDate: '2026-05-31',
    rate: '125.50',
  });

  await expect(page.getByText('Rate added')).toBeVisible();
  const rateRow = page.getByRole('row').filter({ has: page.getByRole('cell', { name: category, exact: true }) });
  await expect(rateRow).toBeVisible();
  await expect(rateRow.getByRole('cell', { name: 'GBP 125.50', exact: true })).toBeVisible();
});

test('saves settings from the browser', async ({ page }) => {
  const businessName = `E2E Business ${Date.now()}`;
  const customerName = `E2E Customer ${Date.now()}`;

  await page.goto('/settings');
  await waitForCSRF(page);

  await page.getByLabel('Business Name').fill(businessName);
  await page.getByLabel('Bank Name').fill('E2E Bank');
  await page.getByLabel('Account Name').fill('E2E Accounts');
  await page.getByLabel('Account Number').fill('12345678');
  await page.getByLabel('Sort Code').fill('12-34-56');
  await page.getByLabel('Default Due Days').fill('21');
  await page.getByLabel('Business Address').fill('1 Test Street\nBrowser City');
  await page.getByLabel('Payment Terms').fill('Please pay within 21 days.');
  await page.getByLabel('Customer Name / Company').fill(customerName);
  await page.getByLabel('Customer Email').fill('customer@example.test');
  await page.getByLabel('Customer City').fill('Testville');
  await page.getByLabel('Customer Postal Code').fill('TE1 1ST');
  await page.getByLabel('Customer Address').fill('2 Customer Road');

  await Promise.all([
    page.waitForURL('**/settings?notice=*'),
    page.getByRole('button', { name: 'Save Settings' }).click(),
  ]);

  await expect(page.getByText('Settings saved')).toBeVisible();

  await page.reload();
  await expect(page.getByLabel('Business Name')).toHaveValue(businessName);
  await expect(page.getByLabel('Customer Name / Company')).toHaveValue(customerName);
  await expect(page.getByLabel('Default Due Days')).toHaveValue('21');
});

test('generates an invoice from browser-created data', async ({ page }) => {
  const category = uniqueCategory('Invoice E2E');
  const notes = `Invoiceable browser entry ${category}`;

  await createRate(page, {
    category,
    startDate: '2026-05-01',
    endDate: '2026-05-31',
    rate: '100.00',
  });
  await createEntry(page, {
    date: '2026-05-04',
    category,
    hours: '8',
    notes,
  });

  await page.goto('/invoices');
  await waitForCSRF(page);
  await page.getByLabel('Month').fill('2026-05');
  await page.getByLabel('Category').selectOption(category);
  await page.getByLabel('Invoice Date').fill('2026-05-31');
  await page.getByLabel('Due Days').fill('14');

  await Promise.all([
    page.waitForURL(/\/invoices\/\d+\?notice=/),
    page.getByRole('button', { name: 'Generate Invoice' }).click(),
  ]);

  await expect(page.getByText('Invoice created')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Invoice', exact: true })).toBeVisible();
  await expect(page.getByText(category, { exact: true }).first()).toBeVisible();
  await expect(page.getByText('2026-05', { exact: true })).toBeVisible();
  await expect(page.getByRole('cell', { name: 'GBP 800.00', exact: true })).toBeVisible();
});
