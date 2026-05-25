const { test } = require('./support/fixtures');
const {
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
  saveSettings,
  smokeNavigate,
  uniqueCategory,
} = require('./support/workflows');

function settings(prefix) {
  return {
    businessName: `${prefix} Business`,
    bankName: `${prefix} Bank`,
    accountName: `${prefix} Accounts`,
    accountNumber: '12345678',
    sortCode: '12-34-56',
    defaultDueDays: '21',
    businessAddress: '1 Test Street\nBrowser City',
    paymentTerms: 'Please pay within 21 days.',
    customerName: `${prefix} Customer`,
    customerEmail: 'customer@example.test',
    customerCity: 'Testville',
    customerPostalCode: 'TE1 1ST',
    customerAddress: '2 Customer Road',
  };
}

test('smoke navigation renders no-auth pages', async ({ page }) => {
  await smokeNavigate(page);
});

test('entry and rate HTMX workflows create, edit, and delete rows', async ({ page }) => {
  const entryCategory = uniqueCategory('NoAuth Entry');
  const editedEntryCategory = `${entryCategory} Edited`;
  const rateCategory = uniqueCategory('NoAuth Rate');
  const editedRateCategory = `${rateCategory} Edited`;

  await createEntry(page, {
    date: '2026-05-04',
    category: entryCategory,
    hours: '7.5',
    notes: 'original noauth entry',
  });
  await editEntry(page, {
    category: entryCategory,
    newCategory: editedEntryCategory,
    newHours: '6.25',
    newNotes: 'edited noauth entry',
  });
  await deleteEntry(page, editedEntryCategory);

  await createRate(page, {
    category: rateCategory,
    startDate: '2026-05-01',
    endDate: '2026-05-31',
    rate: '125.50',
  });
  await editRate(page, {
    category: rateCategory,
    newCategory: editedRateCategory,
    newRate: '130.75',
  });
  await deleteRate(page, editedRateCategory);
});

test('multi-category invoice, PDF, and dashboard coverage uses browser-created data', async ({ page }, testInfo) => {
  const prefix = uniqueCategory('NoAuth Invoice');
  const alpha = `${prefix} Alpha`;
  const beta = `${prefix} Beta`;
  const savedSettings = settings(prefix);
  const monthNumber = String(5 + testInfo.retry).padStart(2, '0');
  const month = `2026-${monthNumber}`;

  await saveSettings(page, savedSettings);
  await createRate(page, { category: alpha, startDate: `${month}-01`, endDate: `${month}-28`, rate: '100.00' });
  await createRate(page, { category: beta, startDate: `${month}-01`, endDate: `${month}-28`, rate: '80.00' });
  await createEntry(page, { date: `${month}-04`, category: alpha, hours: '8', notes: 'alpha invoice work' });
  await createEntry(page, { date: `${month}-05`, category: beta, hours: '3.5', notes: 'beta invoice work' });

  const invoiceID = await generateInvoice(page, {
    month,
    category: 'All',
    invoiceDate: `${month}-28`,
    dueDays: '14',
  });

  await assertInvoicePreview(page, {
    invoiceID,
    month,
    category: 'All',
    settings: savedSettings,
    expectedLines: [
      { category: alpha, hours: '8.00', rate: '100.00', amount: '800.00', total: '1080.00' },
      { category: beta, hours: '3.50', rate: '80.00', amount: '280.00', total: '1080.00' },
    ],
  });
  await assertPDF(page, invoiceID);
  await assertDashboard(page, {
    year: '2026',
    month: monthNumber,
    category: alpha,
    totalHours: '8.00',
    totalAmount: '800.00',
  });
});
