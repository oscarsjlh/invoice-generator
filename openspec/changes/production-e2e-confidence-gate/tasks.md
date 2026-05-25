## 1. E2E Helper Surface

- [x] 1.1 Add e2e-only Go route registration behind `//go:build e2e`.
- [x] 1.2 Add an environment guard such as `E2E_TEST_HELPERS=true` before helper routes perform setup.
- [x] 1.3 Add a helper endpoint that creates a user and sets/returns a valid authenticated session using the real auth/session code paths.
- [x] 1.4 Add tests or build checks proving helper routes are absent from normal production builds.

## 2. Docker And Playwright Infrastructure

- [x] 2.1 Build the e2e app image with `go build -tags=e2e` only for the e2e Compose environment.
- [x] 2.2 Add separate Compose services for auth-on, noauth-1, and noauth-2 with isolated SQLite/user/upload paths.
- [x] 2.3 Configure Playwright projects/fixtures for auth-on and no-auth modes.
- [x] 2.4 Route no-auth tests to worker-specific no-auth app instances.
- [x] 2.5 Configure two CI workers initially, while keeping the service naming/config scalable to four.
- [x] 2.6 Keep one CI retry initially and expose retry information in reports/artifacts.

## 3. Shared Browser Test Helpers

- [x] 3.1 Create shared helpers for navigating smoke pages.
- [x] 3.2 Create shared helpers for entry create/edit/delete browser workflows.
- [x] 3.3 Create shared helpers for rate create/edit/delete browser workflows.
- [x] 3.4 Create shared helpers for settings save/reload browser workflows.
- [x] 3.5 Create shared helpers for invoice generation, preview assertions, PDF response checks, and dashboard summary/filter assertions.
- [x] 3.6 Keep helper endpoints limited to setup; avoid using them to verify business workflow results.

## 4. Production Gate Coverage

- [x] 4.1 Add smoke navigation coverage for dashboard, entries, rates, invoices, and settings in both modes.
- [x] 4.2 Add entry create, HTMX edit, and HTMX delete coverage.
- [x] 4.3 Add rate create, HTMX edit, and delete coverage.
- [x] 4.4 Add settings save and reload coverage.
- [x] 4.5 Add invoice generation coverage with exact line totals.
- [x] 4.6 Add invoice preview assertions for invoice number, month, category, line items, totals, payment details, and client details.
- [x] 4.7 Add PDF assertions for status 200, `application/pdf`, non-trivial body length, and `%PDF` file header.
- [x] 4.8 Add dashboard summary/filter coverage from data created through browser workflows.
- [x] 4.9 Add both multi-category `All` invoice coverage and focused single-category invoice coverage without duplicating both in every mode.

## 5. Auth-On Boundary Coverage

- [x] 5.1 Add seeded-session test that reaches the app in auth-on mode.
- [x] 5.2 Add unauthenticated protected-route redirect coverage.
- [x] 5.3 Add logout coverage through the real UI logout form/button.
- [x] 5.4 Add user-isolation coverage proving user B cannot see user A's entries, rates, invoices, dashboard totals, or settings.

## 6. Artifacts, Documentation, And CI

- [x] 6.1 Configure traces, screenshots, video-on-failure, HTML report, and browser console log capture.
- [x] 6.2 Capture and label Docker logs for auth-on, noauth-1, noauth-2, and Playwright services.
- [x] 6.3 Document local and CI commands for the production Chromium gate.
- [x] 6.4 Document scheduled non-blocking Firefox/WebKit job expectations.
- [x] 6.5 Configure CI path filters for PRs touching app code, templates, migrations, Docker, config, or tests.
- [x] 6.6 Ensure the Chromium production gate runs on relevant PRs, main, and before release tags.

## 7. Verification

- [x] 7.1 Run the production Chromium e2e gate locally or in the Docker e2e environment.
- [x] 7.2 Run the existing Go test suite.
- [x] 7.3 Run `openspec validate production-e2e-confidence-gate --strict` and resolve validation issues.
