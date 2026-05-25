const base = require('@playwright/test');

function noAuthBaseURL(testInfo) {
  const configured = testInfo.project.use.noAuthBaseURLs || [];
  const urls = configured.length > 0 ? configured : ['http://127.0.0.1:18081', 'http://127.0.0.1:18082'];
  return urls[testInfo.parallelIndex % urls.length];
}

const test = base.test.extend({
  page: async ({ browser }, use, testInfo) => {
    const mode = testInfo.project.use.mode;
    const baseURL = mode === 'noauth' ? noAuthBaseURL(testInfo) : testInfo.project.use.baseURL;
    const context = await browser.newContext({ baseURL });
    const page = await context.newPage();
    const consoleLines = [];

    page.on('console', (message) => {
      consoleLines.push(`[${message.type()}] ${message.text()}`);
    });
    page.on('pageerror', (error) => {
      consoleLines.push(`[pageerror] ${error.message}`);
    });

    if (mode === 'auth-on') {
      const safeTitle = testInfo.title.replace(/[^a-z0-9]+/gi, '-').toLowerCase();
      const username = `e2e-${testInfo.workerIndex}-${Date.now()}-${safeTitle}`.slice(0, 80);
      const response = await page.request.post('/__e2e/session', {
        data: { username, display_name: username },
      });
      if (!response.ok()) {
        throw new Error(`Failed to seed authenticated session: ${response.status()} ${await response.text()}`);
      }
    }

    await use(page);

    if (consoleLines.length > 0) {
      await testInfo.attach('browser-console.log', {
        body: consoleLines.join('\n'),
        contentType: 'text/plain',
      });
    }
    await context.close();
  },
});

module.exports = {
  expect: base.expect,
  test,
};
