## Why

The app needs browser-level confidence that critical invoice workflows still load and behave correctly after changes. Playwright smoke and E2E tests will catch regressions that unit tests and handler-level checks can miss, especially around rendered pages, forms, navigation, and user-visible flows.

## What Changes

- Add Playwright as the browser automation test runner for acceptance-level coverage.
- Add smoke tests that verify the app starts and key pages render successfully.
- Add E2E tests for core invoice workflows, including entries, rates, settings, and invoice generation where supported by the current app behavior.
- Add npm scripts and/or test commands so the browser tests can run locally and in CI.
- Document the expected setup for running the Playwright suite, including any required app server startup behavior.

## Capabilities

### New Capabilities
- `playwright-acceptance-tests`: Browser-level smoke and E2E tests that validate critical user-visible invoice app workflows.

### Modified Capabilities

None.

## Impact

- Adds Playwright test dependencies and configuration.
- Adds browser test files for smoke and E2E coverage.
- May add lightweight test fixtures or helpers for deterministic setup and cleanup.
- May update package scripts and CI-facing commands for running acceptance tests.
- Does not change application runtime APIs or user-facing invoice behavior.
