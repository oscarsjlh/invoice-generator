## 1. Playwright Setup

- [x] 1.1 Add Playwright test dependencies to `invoice-service/package.json` and update the pnpm lockfile.
- [x] 1.2 Add a Playwright configuration under `invoice-service` with `webServer` startup, base URL, browser project selection, and health-check readiness.
- [x] 1.3 Configure the Playwright server environment to use `AUTH_ENABLED=false` and test-owned data paths for SQLite databases, uploads, and user stores.
- [x] 1.4 Add package scripts for running the Playwright suite and installing required browser binaries.

## 2. Smoke Coverage

- [x] 2.1 Add smoke tests that verify the dashboard loads successfully in a browser.
- [x] 2.2 Add smoke tests that verify entries, rates, invoices, and settings pages render their expected headings or primary content.
- [x] 2.3 Ensure smoke tests fail on non-successful page responses or app startup failures.

## 3. E2E Workflow Coverage

- [x] 3.1 Add an E2E test that creates a valid time entry through the entries page and verifies it appears in the rendered entries view.
- [x] 3.2 Add an E2E test that creates a valid rate through the rates page and verifies it appears in the rendered rates view.
- [x] 3.3 Add an E2E test that saves settings through the settings page and verifies the saved values after reload.
- [x] 3.4 Add an E2E test that creates the required entry and rate data, generates an invoice through the browser, and verifies the generated invoice preview.
- [x] 3.5 Use semantic locators and user-visible assertions instead of brittle template-specific selectors wherever the current markup allows it.

## 4. Documentation And Verification

- [x] 4.1 Document the Playwright commands and any first-run browser installation step for local and CI use.
- [ ] 4.2 Run the Playwright suite and fix any test or setup failures.
- [ ] 4.3 Run the existing Go test suite to confirm the browser test setup did not regress server behavior.
- [ ] 4.4 Run `openspec validate add-playwright-smoke-e2e-tests --strict` and resolve any proposal/spec/task validation issues.
