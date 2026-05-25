## ADDED Requirements

### Requirement: Playwright test runner configuration
The system SHALL provide a Playwright configuration for the invoice service that can start the application server, wait for it to become healthy, execute browser tests, and stop the server after the run.

#### Scenario: Test command starts the app
- **WHEN** the Playwright test command is run from the invoice service
- **THEN** the command starts the Go HTTP server before executing browser tests

#### Scenario: Tests wait for readiness
- **WHEN** the Go HTTP server is still starting
- **THEN** Playwright waits for the app health endpoint before running tests

### Requirement: Smoke coverage for critical pages
The system SHALL include smoke tests that verify critical user-visible pages render successfully in a real browser.

#### Scenario: Dashboard renders
- **WHEN** a smoke test opens the dashboard route
- **THEN** the page responds successfully and displays the invoice app navigation or dashboard content

#### Scenario: Ledger pages render
- **WHEN** a smoke test opens the entries, rates, invoices, and settings routes
- **THEN** each page responds successfully and displays its expected heading or primary content

### Requirement: Browser workflow coverage for invoice data
The system SHALL include E2E tests that exercise core invoice data workflows through browser interactions.

#### Scenario: Entry can be created from the browser
- **WHEN** a browser test submits a valid time entry through the entries page
- **THEN** the entry appears in the rendered entries view

#### Scenario: Rate can be created from the browser
- **WHEN** a browser test submits a valid rate through the rates page
- **THEN** the rate appears in the rendered rates view

#### Scenario: Settings can be saved from the browser
- **WHEN** a browser test updates business, invoice, payment, or customer settings through the settings page
- **THEN** the saved values are visible when the settings page is reloaded

#### Scenario: Invoice can be generated from browser-created data
- **WHEN** a browser test creates the required entry and rate data and submits invoice generation
- **THEN** the app navigates to or displays a generated invoice preview for the selected month and category

### Requirement: Isolated acceptance test data
The system SHALL run Playwright tests against isolated test data paths so browser tests do not read or mutate local development or production invoice data.

#### Scenario: Test database is isolated
- **WHEN** the Playwright suite starts the app server
- **THEN** the app uses test-owned SQLite database paths instead of the default local data paths

#### Scenario: Auth is disabled for workflow tests
- **WHEN** the Playwright suite runs invoice workflow tests
- **THEN** the app runs in no-auth compatibility mode unless a test explicitly opts into authentication behavior

### Requirement: Developer and CI test commands
The system SHALL provide documented package scripts or commands for installing browser test dependencies and running the Playwright suite locally or in CI.

#### Scenario: Local E2E command exists
- **WHEN** a developer runs the documented Playwright test command from the invoice service
- **THEN** the smoke and E2E browser tests execute without requiring manual server startup

#### Scenario: Browser installation command exists
- **WHEN** a fresh environment needs Playwright browser binaries
- **THEN** the project provides a documented command to install the required browsers
