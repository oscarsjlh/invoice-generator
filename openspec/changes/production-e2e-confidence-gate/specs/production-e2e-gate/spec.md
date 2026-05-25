## ADDED Requirements

### Requirement: Production E2E Gate Runs Both Runtime Modes
The system SHALL provide a Docker-backed Playwright production confidence gate that runs against both auth-enabled mode and no-auth compatibility mode.

#### Scenario: Auth-enabled app service is tested
- **WHEN** the production E2E gate runs
- **THEN** it starts an app service with authentication enabled and runs auth-on Playwright specs against it

#### Scenario: No-auth app services are tested
- **WHEN** the production E2E gate runs
- **THEN** it starts no-auth app services with isolated legacy SQLite databases and runs no-auth Playwright specs against them

#### Scenario: Runtime modes do not share data
- **WHEN** auth-on and no-auth Playwright projects execute in the same gate
- **THEN** each mode uses separate database, auth database, user database, upload, and service state paths

### Requirement: Compiled E2E Setup Helpers
The system SHALL provide setup-only HTTP helpers that are compiled into e2e builds and absent from normal production builds.

#### Scenario: Helper creates authenticated session
- **WHEN** an auth-on Playwright test requests a seeded session from the e2e helper
- **THEN** the helper creates or selects a test user and establishes a valid session using the real auth/session code paths

#### Scenario: Helper requires e2e guard
- **WHEN** the e2e helper route is called without the required e2e environment guard
- **THEN** it refuses to perform setup

#### Scenario: Helper unavailable in production build
- **WHEN** the application is built without the e2e build tag
- **THEN** the helper routes are not registered

#### Scenario: Helpers are not used for workflow verification
- **WHEN** browser workflow tests verify entries, rates, settings, invoices, dashboards, or PDFs
- **THEN** they verify through user-visible UI or public HTTP responses rather than helper endpoints

### Requirement: Parallel Worker Isolation
The production E2E gate SHALL support two parallel Playwright workers without relying on serial test execution for correctness.

#### Scenario: Auth-on tests isolate by user
- **WHEN** auth-on tests run in parallel
- **THEN** each test uses its own seeded user/session or otherwise isolated authenticated user state

#### Scenario: No-auth tests isolate by app instance
- **WHEN** no-auth tests run in parallel
- **THEN** each worker targets a no-auth app instance with its own legacy database

#### Scenario: Worker configuration can scale
- **WHEN** the configured worker count increases from two to four in a future change
- **THEN** the service naming and Playwright routing model can be extended without redesigning the suite

### Requirement: Browser Smoke Coverage
The production E2E gate SHALL verify that critical user-visible pages render in a real Chromium browser.

#### Scenario: Dashboard page renders
- **WHEN** a browser opens the dashboard route
- **THEN** the response is successful and the dashboard content is visible

#### Scenario: Ledger pages render
- **WHEN** a browser opens entries, rates, invoices, and settings routes
- **THEN** each page responds successfully and displays its expected heading or primary content

### Requirement: Entry Workflow Coverage
The production E2E gate SHALL cover entry creation, HTMX editing, and HTMX deletion through browser interactions.

#### Scenario: Entry is created
- **WHEN** a browser submits a valid entry form
- **THEN** the entry appears in the entries table with the submitted category, hours, date, and notes

#### Scenario: Entry is edited through HTMX
- **WHEN** a browser edits an existing entry through the inline HTMX edit flow
- **THEN** the updated entry appears in the entries table without requiring a full-page-only verification path

#### Scenario: Entry is deleted through HTMX
- **WHEN** a browser deletes an existing entry through the HTMX action
- **THEN** the entry no longer appears in the entries table

### Requirement: Rate Workflow Coverage
The production E2E gate SHALL cover rate creation, editing, and deletion through browser interactions.

#### Scenario: Rate is created
- **WHEN** a browser submits a valid rate form
- **THEN** the rate appears in the rates table with the submitted category, date range, and amount

#### Scenario: Rate is edited through HTMX
- **WHEN** a browser edits an existing rate through the inline HTMX edit flow
- **THEN** the updated rate appears in the rates table

#### Scenario: Rate is deleted
- **WHEN** a browser deletes an existing rate
- **THEN** the rate no longer appears in the rates table

### Requirement: Settings Persistence Coverage
The production E2E gate SHALL cover saving and reloading business, payment, invoice, and client settings.

#### Scenario: Settings persist after reload
- **WHEN** a browser saves settings with business, payment, invoice, and client fields
- **THEN** a subsequent settings page load displays the saved values

### Requirement: Invoice Generation And Preview Coverage
The production E2E gate SHALL cover invoice generation from browser-created entries and rates with exact visible totals.

#### Scenario: Multi-category invoice is generated
- **WHEN** a browser creates entries and rates for multiple categories and generates an invoice for category "All"
- **THEN** the invoice preview shows all expected line items and exact line totals

#### Scenario: Single-category invoice is generated
- **WHEN** a browser creates entries and rates for multiple categories and generates an invoice for one selected category
- **THEN** the invoice preview shows only that category's expected line items and totals

#### Scenario: Invoice preview displays required details
- **WHEN** a browser views a generated invoice preview
- **THEN** the page displays the invoice number, month, category, line items, subtotal, total, payment details, and client details

### Requirement: PDF Endpoint Coverage
The production E2E gate SHALL verify that generated invoice PDFs are available through the public PDF endpoint.

#### Scenario: PDF endpoint returns valid PDF response
- **WHEN** a browser or Playwright request context fetches the PDF endpoint for a generated invoice
- **THEN** the response status is 200, the content type includes `application/pdf`, the body is non-trivial, and the body begins with `%PDF`

### Requirement: Dashboard Summary And Filter Coverage
The production E2E gate SHALL verify dashboard summaries and filters using data created through browser workflows.

#### Scenario: Dashboard shows correct summary totals
- **WHEN** browser-created entries and rates exist for multiple months and categories
- **THEN** the dashboard displays the expected total hours and amounts

#### Scenario: Dashboard filters by period
- **WHEN** a browser selects a specific year and month on the dashboard
- **THEN** the displayed summary and recent invoices reflect only the selected period

### Requirement: Auth-On Boundary Coverage
The production E2E gate SHALL verify authentication boundaries and tenant isolation in auth-enabled mode.

#### Scenario: Seeded session reaches protected app
- **WHEN** an auth-on browser context has a session created by the e2e setup helper
- **THEN** it can access protected application pages

#### Scenario: Unauthenticated protected route redirects
- **WHEN** an unauthenticated browser opens a protected application route
- **THEN** the app redirects to the login page with a sign-in notice

#### Scenario: Logout invalidates access
- **WHEN** a signed-in browser submits the real logout UI
- **THEN** a subsequent protected-route visit redirects to login and protected app content is not visible

#### Scenario: User data is isolated
- **WHEN** user A creates entries, rates, settings, invoices, and dashboard-visible totals through the browser
- **THEN** user B cannot see user A's entries, rates, invoices, dashboard totals, or settings through normal app navigation

### Requirement: Browser Matrix Policy
The production E2E gate SHALL use Chromium desktop as the required blocking browser and reserve Firefox/WebKit for scheduled non-blocking runs.

#### Scenario: Chromium blocks changes
- **WHEN** a relevant PR, main branch build, or release preflight runs
- **THEN** the Chromium production E2E gate must pass

#### Scenario: Firefox and WebKit are scheduled
- **WHEN** scheduled browser matrix jobs run for Firefox or WebKit
- **THEN** failures are reported as non-blocking unless a later change promotes them to blockers

### Requirement: Failure Artifacts
The production E2E gate SHALL capture artifacts that make CI failures diagnosable.

#### Scenario: Playwright failure artifacts are captured
- **WHEN** a Playwright test fails
- **THEN** the run preserves trace, screenshot, video-on-failure, HTML report, and browser console logs

#### Scenario: Docker logs are captured
- **WHEN** the production E2E gate completes or fails
- **THEN** Docker logs are available and labelled by service or mode, including auth-on, noauth-1, noauth-2, and Playwright

### Requirement: CI Execution Policy
The production E2E gate SHALL run in CI for changes that can affect browser-visible application behavior or release safety.

#### Scenario: Relevant PRs run the gate
- **WHEN** a PR changes app code, templates, migrations, Docker, config, or tests
- **THEN** CI runs the Chromium production E2E gate

#### Scenario: Main and release preflight run the gate
- **WHEN** CI runs for main branch or before release tags
- **THEN** the Chromium production E2E gate runs

#### Scenario: One retry is visible
- **WHEN** CI retries a failing Playwright test once and the retry passes
- **THEN** the report or artifacts make the retry visible rather than hiding the flake completely

### Requirement: Initial Scope Exclusions
The production E2E gate SHALL exclude OCR import and visual regression assertions from the initial blocking scope.

#### Scenario: OCR is not part of initial blocker
- **WHEN** the initial production E2E gate runs
- **THEN** it does not require OCR import coverage to pass

#### Scenario: Screenshots are artifacts, not golden assertions
- **WHEN** the initial production E2E gate runs
- **THEN** screenshots are used for diagnostics and not for blocking visual snapshot comparisons
