const { expect } = require('./fixtures');

function uniqueCategory(prefix) {
  return `${prefix} ${Date.now()} ${Math.random().toString(16).slice(2, 8)}`;
}

async function gotoOK(page, path) {
  const response = await page.goto(path);
  expect(response, `${path} should return a response`).not.toBeNull();
  expect(response.ok(), `${path} should respond successfully`).toBe(true);
}

async function waitForCSRF(page) {
  await expect(page.locator('input[name="csrf_token"]').first()).not.toHaveValue('');
}

async function smokeNavigate(page) {
  await gotoOK(page, '/');
  await expect(page.getByRole('heading', { name: 'Dashboard', exact: true })).toBeVisible();

  const pages = [
    { path: '/entries', heading: 'Entries', content: 'Add Entry' },
    { path: '/rates', heading: 'Rates', content: 'Add Rate' },
    { path: '/invoices', heading: 'Invoices', content: 'Generate Invoice' },
    { path: '/settings', heading: 'Settings', content: 'Business Details' },
  ];
  for (const pageSpec of pages) {
    await gotoOK(page, pageSpec.path);
    await expect(page.getByRole('heading', { name: pageSpec.heading, exact: true })).toBeVisible();
    await expect(page.getByRole('heading', { name: pageSpec.content, exact: true })).toBeVisible();
  }
}

async function createEntry(page, { date, category, hours, notes }) {
  await page.goto('/entries');
  await waitForCSRF(page);
  await page.getByLabel('Date').fill(date);
  await page.getByLabel('Category').fill(category);
  await page.getByLabel('Hours').fill(hours);
  await page.getByLabel('Notes').fill(notes || '');
  await Promise.all([
    page.waitForURL('**/entries?notice=*'),
    page.getByRole('button', { name: 'Save Entry' }).click(),
  ]);
  await expect(page.getByRole('cell', { name: category, exact: true })).toBeVisible();
}

async function editEntry(page, { category, newCategory, newHours, newNotes }) {
  await page.goto('/entries');
  const row = page.getByRole('row').filter({ has: page.getByRole('cell', { name: category, exact: true }) });
  await row.getByRole('button', { name: 'Edit' }).click();
  const form = page.locator('form[hx-post^="/entries/"]').first();
  await form.getByLabel('Category').fill(newCategory);
  await form.getByLabel('Hours').fill(newHours);
  await form.getByLabel('Notes').fill(newNotes);
  await form.getByRole('button', { name: 'Save' }).click();
  await expect(page.getByRole('cell', { name: newCategory, exact: true })).toBeVisible();
  await expect(page.getByRole('cell', { name: newNotes, exact: true })).toBeVisible();
}

async function deleteEntry(page, category) {
  page.once('dialog', (dialog) => dialog.accept());
  await page.goto('/entries');
  const row = page.getByRole('row').filter({ has: page.getByRole('cell', { name: category, exact: true }) });
  await row.getByRole('button', { name: 'Delete' }).click();
  await expect(page.getByRole('cell', { name: category, exact: true })).toHaveCount(0);
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
  await expect(page.getByRole('cell', { name: category, exact: true })).toBeVisible();
}

async function editRate(page, { category, newCategory, newRate }) {
  await page.goto('/rates');
  const row = page.getByRole('row').filter({ has: page.getByRole('cell', { name: category, exact: true }) });
  await row.getByRole('button', { name: 'Edit' }).click();
  const form = page.locator('form[hx-post^="/rates/"]').first();
  await form.getByLabel('Category').fill(newCategory);
  await form.getByLabel('Rate').fill(newRate);
  await form.getByRole('button', { name: 'Save' }).click();
  await expect(page.getByRole('cell', { name: newCategory, exact: true })).toBeVisible();
  await expect(page.getByRole('cell', { name: `GBP ${Number(newRate).toFixed(2)}`, exact: true })).toBeVisible();
}

async function deleteRate(page, category) {
  page.once('dialog', (dialog) => dialog.accept());
  await page.goto('/rates');
  const row = page.getByRole('row').filter({ has: page.getByRole('cell', { name: category, exact: true }) });
  await row.getByRole('button', { name: 'Delete' }).click();
  await expect(page.getByRole('cell', { name: category, exact: true })).toHaveCount(0);
}

async function saveSettings(page, values) {
  await page.goto('/settings');
  await waitForCSRF(page);
  await page.getByLabel('Business Name').fill(values.businessName);
  await page.getByLabel('Bank Name').fill(values.bankName);
  await page.getByLabel('Account Name').fill(values.accountName);
  await page.getByLabel('Account Number').fill(values.accountNumber);
  await page.getByLabel('Sort Code').fill(values.sortCode);
  await page.getByLabel('Default Due Days').fill(values.defaultDueDays);
  await page.getByLabel('Business Address').fill(values.businessAddress);
  await page.getByLabel('Payment Terms').fill(values.paymentTerms);
  await page.getByLabel('Customer Name / Company').fill(values.customerName);
  await page.getByLabel('Customer Email').fill(values.customerEmail);
  await page.getByLabel('Customer City').fill(values.customerCity);
  await page.getByLabel('Customer Postal Code').fill(values.customerPostalCode);
  await page.getByLabel('Customer Address').fill(values.customerAddress);
  await Promise.all([
    page.waitForURL('**/settings?notice=*'),
    page.getByRole('button', { name: 'Save Settings' }).click(),
  ]);
  await expect(page.getByText('Settings saved')).toBeVisible();
  await page.reload();
  await expect(page.getByLabel('Business Name')).toHaveValue(values.businessName);
  await expect(page.getByLabel('Customer Name / Company')).toHaveValue(values.customerName);
}

async function generateInvoice(page, { month, category, invoiceDate, dueDays }) {
  await page.goto('/invoices');
  await waitForCSRF(page);
  await page.getByLabel('Month').fill(month);
  await page.getByLabel('Category').selectOption(category);
  await page.getByLabel('Invoice Date').fill(invoiceDate);
  await page.getByLabel('Due Days').fill(dueDays);
  await Promise.all([
    page.waitForURL(/\/invoices\/\d+\?notice=/),
    page.getByRole('button', { name: 'Generate Invoice' }).click(),
  ]);
  const match = page.url().match(/\/invoices\/(\d+)/);
  expect(match).not.toBeNull();
  return match[1];
}

async function assertInvoicePreview(page, { invoiceID, month, category, expectedLines, settings }) {
  await gotoOK(page, `/invoices/${invoiceID}`);
  await expect(page.getByRole('heading', { name: 'Invoice', exact: true })).toBeVisible();
  await expect(page.getByText(month, { exact: true })).toBeVisible();
  await expect(page.getByText(category, { exact: true }).first()).toBeVisible();
  await expect(page.getByText(settings.businessName, { exact: true })).toBeVisible();
  await expect(page.getByText(settings.bankName, { exact: true })).toBeVisible();
  await expect(page.getByText(settings.customerName, { exact: true })).toBeVisible();

  for (const line of expectedLines) {
    const row = page.getByRole('row').filter({ has: page.getByRole('cell', { name: line.category, exact: true }) });
    await expect(row.getByRole('cell', { name: line.hours, exact: true })).toBeVisible();
    await expect(row.getByRole('cell', { name: `GBP ${line.rate}`, exact: true })).toBeVisible();
    await expect(row.getByRole('cell', { name: `GBP ${line.amount}`, exact: true })).toBeVisible();
  }
  await expect(page.getByRole('row').filter({ hasText: 'Total' }).getByText(`GBP ${expectedLines.at(-1).total}`)).toBeVisible();
}

async function assertPDF(page, invoiceID) {
  const response = await page.request.get(`/invoices/${invoiceID}/pdf`);
  expect(response.status()).toBe(200);
  expect(response.headers()['content-type']).toContain('application/pdf');
  const body = await response.body();
  expect(body.length).toBeGreaterThan(1000);
  expect(body.subarray(0, 4).toString()).toBe('%PDF');
}

async function assertDashboard(page, { year, month, category, totalHours, totalAmount }) {
  await gotoOK(page, `/?year=${year}&month=${month}`);
  const summary = page.locator('.summary-table');
  const row = summary.getByRole('row').filter({ hasText: category });
  await expect(row.getByRole('cell', { name: totalHours, exact: true })).toBeVisible();
  await expect(row.getByRole('cell', { name: `GBP ${totalAmount}`, exact: true })).toBeVisible();
}

module.exports = {
  assertDashboard,
  assertInvoicePreview,
  assertPDF,
  createEntry,
  createRate,
  deleteEntry,
  deleteRate,
  editEntry,
  editRate,
  generateInvoice,
  gotoOK,
  saveSettings,
  smokeNavigate,
  uniqueCategory,
};
