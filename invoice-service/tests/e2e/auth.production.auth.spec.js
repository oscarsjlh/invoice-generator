const { expect, test } = require('./support/fixtures');
const {
  assertDashboard,
  assertInvoicePreview,
  assertPDF,
  createEntry,
  createRate,
  generateInvoice,
  gotoOK,
  saveSettings,
  smokeNavigate,
  uniqueCategory,
} = require('./support/workflows');

function settings(prefix) {
  return {
    businessName: `${prefix} Business`,
    bankName: `${prefix} Bank`,
    accountName: `${prefix} Accounts`,
    accountNumber: '87654321',
    sortCode: '65-43-21',
    defaultDueDays: '30',
    businessAddress: '9 Auth Street\nSession City',
    paymentTerms: 'Auth terms.',
    customerName: `${prefix} Customer`,
    customerEmail: 'auth-customer@example.test',
    customerCity: 'Authtown',
    customerPostalCode: 'AU1 1TH',
    customerAddress: '11 Auth Customer Road',
  };
}

test('seeded session reaches auth-on app and smoke pages render', async ({ page }) => {
  await smokeNavigate(page);
});

test('single-category invoice and dashboard coverage works in auth-on mode', async ({ page }) => {
  const prefix = uniqueCategory('Auth Invoice');
  const selected = `${prefix} Selected`;
  const other = `${prefix} Other`;
  const savedSettings = settings(prefix);

  await saveSettings(page, savedSettings);
  await createRate(page, { category: selected, startDate: '2026-05-01', endDate: '2026-05-31', rate: '150.00' });
  await createRate(page, { category: other, startDate: '2026-05-01', endDate: '2026-05-31', rate: '90.00' });
  await createEntry(page, { date: '2026-05-08', category: selected, hours: '4', notes: 'selected work' });
  await createEntry(page, { date: '2026-05-09', category: other, hours: '10', notes: 'other work' });

  const invoiceID = await generateInvoice(page, {
    month: '2026-05',
    category: selected,
    invoiceDate: '2026-05-31',
    dueDays: '10',
  });

  await assertInvoicePreview(page, {
    invoiceID,
    month: '2026-05',
    category: selected,
    settings: savedSettings,
    expectedLines: [
      { category: selected, hours: '4.00', rate: '150.00', amount: '600.00', total: '600.00' },
    ],
  });
  await expect(page.getByText(other, { exact: true })).toHaveCount(0);
  await assertPDF(page, invoiceID);
  await assertDashboard(page, {
    year: '2026',
    month: '05',
    category: selected,
    totalHours: '4.00',
    totalAmount: '600.00',
  });
});

test('unauthenticated protected route redirects and logout invalidates access', async ({ browser }, testInfo) => {
  const baseURL = testInfo.project.use.baseURL;
  const unauthenticated = await browser.newPage({ baseURL });
  await unauthenticated.goto('/entries');
  await expect(unauthenticated).toHaveURL(/\/login\?notice=/);
  await expect(unauthenticated.getByText('Please sign in')).toBeVisible();
  await unauthenticated.close();

  const context = await browser.newContext({ baseURL });
  const page = await context.newPage();
  const response = await page.request.post('/__e2e/session', {
    data: { username: `logout-${Date.now()}`, display_name: 'Logout User' },
  });
  expect(response.ok()).toBe(true);
  await gotoOK(page, '/entries');
  await page.getByRole('button', { name: 'Logout' }).click();
  await expect(page).toHaveURL(/\/login\?notice=/);
  await page.goto('/entries');
  await expect(page).toHaveURL(/\/login\?notice=/);
  await context.close();
});

test('auth-on user data is isolated between seeded users', async ({ browser }, testInfo) => {
  const baseURL = testInfo.project.use.baseURL;
  const categoryA = uniqueCategory('User A Private');

  async function newSeededPage(username) {
    const context = await browser.newContext({ baseURL });
    const page = await context.newPage();
    const response = await page.request.post('/__e2e/session', {
      data: { username, display_name: username },
    });
    expect(response.ok()).toBe(true);
    return { context, page };
  }

  const userA = await newSeededPage(`isolation-a-${Date.now()}`);
  await createRate(userA.page, { category: categoryA, startDate: '2026-06-01', endDate: '2026-06-30', rate: '200.00' });
  await createEntry(userA.page, { date: '2026-06-04', category: categoryA, hours: '2', notes: 'private user A entry' });
  await saveSettings(userA.page, settings('User A Private'));
  const invoiceID = await generateInvoice(userA.page, {
    month: '2026-06',
    category: categoryA,
    invoiceDate: '2026-06-30',
    dueDays: '14',
  });
  await assertPDF(userA.page, invoiceID);

  const userB = await newSeededPage(`isolation-b-${Date.now()}`);
  for (const path of ['/entries', '/rates', '/invoices', '/settings', '/?year=2026&month=06']) {
    await gotoOK(userB.page, path);
    await expect(userB.page.getByText(categoryA, { exact: true })).toHaveCount(0);
    await expect(userB.page.getByText('User A Private Business', { exact: true })).toHaveCount(0);
  }

  await userA.context.close();
  await userB.context.close();
});
