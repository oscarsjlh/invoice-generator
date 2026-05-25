## Why

The existing Playwright suite provides useful smoke coverage, but it is not strong enough to act as a production confidence gate. It currently runs only in no-auth mode, uses a single app instance, keeps workers serial, and covers only a subset of the critical invoice workflows.

Production releases need browser-level confidence across both supported runtime modes: authenticated multi-user mode and no-auth compatibility mode. The gate should prove that real browser workflows, HTMX interactions, PDF generation, dashboard summaries, persistence, and auth boundaries work against isolated test data without depending on manual setup.

## What Changes

- Expand Playwright from smoke coverage into a Chromium production confidence gate.
- Run both auth-enabled and no-auth app modes in Docker Compose.
- Add compiled e2e-only HTTP helpers for setup, especially seeded authenticated sessions.
- Support two parallel Playwright workers/projects without shared-state dependence.
- Run multiple no-auth app instances so no-auth tests can execute in parallel against isolated legacy databases.
- Add browser workflow coverage for smoke navigation, entry CRUD, rate CRUD, settings, invoice generation, invoice preview, PDF response checks, dashboard filtering, and auth-only boundaries.
- Keep test helpers limited to setup; verify behavior through UI or public HTTP responses.
- Capture useful failure artifacts and document CI expectations.

## Capabilities

### New Capabilities

- `production-e2e-gate`: A Docker-backed Playwright production confidence gate covering authenticated and no-auth browser workflows with isolated data and failure artifacts.

### Modified Capabilities

- `playwright-acceptance-tests`: Existing Playwright smoke tests become part of a broader production gate rather than the full browser confidence story.

## Impact

- Adds e2e-only Go helper routes behind an `e2e` build tag and an e2e environment guard.
- Updates Docker/Compose configuration for auth-on, noauth-1, noauth-2, and Playwright services.
- Updates Playwright configuration, fixtures, projects, reporters, traces, screenshots, and videos.
- Reorganizes browser specs into shared workflow helpers and mode-specific spec files.
- Updates documentation and CI-facing commands for the production gate and scheduled non-blocking browser matrix.
- Does not change user-facing application behavior or expose test helper routes in production builds.
