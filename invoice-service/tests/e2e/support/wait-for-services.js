const urls = [
  process.env.PLAYWRIGHT_AUTH_BASE_URL || 'http://app-auth-on:8080',
  ...(process.env.PLAYWRIGHT_NOAUTH_BASE_URLS || 'http://app-noauth-1:8080,http://app-noauth-2:8080')
    .split(',')
    .map((url) => url.trim())
    .filter(Boolean),
].map((url) => `${url}/health`);

async function waitFor(url) {
  const end = Date.now() + 120_000;
  while (Date.now() < end) {
    try {
      const response = await fetch(url);
      if (response.ok) return;
    } catch (_) {
      // Retry until Docker DNS and the app listener are ready.
    }
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  throw new Error(`Timed out waiting for ${url}`);
}

(async () => {
  await Promise.all(urls.map(waitFor));
})().catch((err) => {
  console.error(err);
  process.exit(1);
});
