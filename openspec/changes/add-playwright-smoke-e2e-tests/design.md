## Context

The invoice service is a Go application using `net/http`, SQLite, server-rendered HTML templates, and HTMX. It already has Go handler and integration tests, but those tests do not exercise the app through a real browser or validate JavaScript/HTMX behavior, page rendering, navigation, or form interactions from the user's perspective.

The frontend dependencies are managed with `pnpm` in `invoice-service/package.json`, and local runtime defaults write SQLite data under `data/`. Playwright tests need to run without depending on a developer's existing database or authenticated browser state.

All the set up should be done using docker and docker compose to make it easier to move to cid

## Goals / Non-Goals

**Goals:**

- Add a repeatable Playwright test setup under `invoice-service`.
- Validate app startup and key page rendering with smoke tests.
- Validate core browser workflows for entries, rates, settings, and invoice generation with E2E tests.
- Run tests against isolated temporary data paths with auth disabled unless a test explicitly covers authentication.
- Provide clear local commands that can be used by CI without requiring manual server startup.

**Non-Goals:**

- Do not redesign app routes, templates, or persistence behavior just to support tests.
- Do not add broad visual regression testing or screenshot approval workflows.
- Do not require WebAuthn/browser credential flows for the initial smoke and E2E suite.
- Do not cover external SMTP delivery, Typst PDF rendering, or OCR service behavior in the initial Playwright suite.

## Decisions

1. Keep Playwright configuration in `invoice-service`.

   Rationale: The app's web assets, Go module, package manager lockfile, and server command all live under `invoice-service`, so colocating browser tests there keeps commands and fixtures local to the service being tested.

   Alternative considered: Put Playwright at the repository root. This would make cross-service orchestration easier later, but it adds indirection now for a single web service.

2. Use Playwright's `webServer` to start the Go server for tests.

   Rationale: A single `pnpm test:e2e` command should start the app, wait for readiness, run tests, and stop the server. This avoids fragile manual setup and makes CI integration straightforward.

   Alternative considered: Require tests to connect to an already-running server. That is useful for ad hoc debugging, but it is less reliable as the default automation path.

3. Run acceptance tests in legacy no-auth mode by default.

   Rationale: The requested coverage targets invoice app behavior, not WebAuthn. Setting `AUTH_ENABLED=false` avoids unrelated credential setup while preserving coverage for the user-visible ledger and invoice workflows.

   Alternative considered: Register/login through WebAuthn in every test. That would couple the first Playwright suite to browser credential APIs and make basic workflow coverage more brittle.

4. Use isolated temporary SQLite paths for Playwright runs.

   Rationale: Tests must be deterministic and must not read or mutate a developer's local invoice data. The Playwright server command should set `DATABASE_PATH`, `AUTH_DB_PATH`, `USER_DB_DIR`, and related data directories to test-owned temporary locations.

   Alternative considered: Reuse `data/invoices.db`. This is simpler but unsafe and non-deterministic.

5. Prefer semantic locators and user-visible assertions.

   Rationale: Tests should verify behavior through labels, roles, headings, navigation, and visible notices rather than brittle CSS selectors or implementation-only markup details.

   Alternative considered: Use low-level selectors matching template structure. This may be quicker initially but makes template refactors unnecessarily expensive.

## Risks / Trade-offs

- Browser tests can be slower than Go tests -> Keep the suite focused on smoke and critical workflows, and keep unit/handler tests as the primary exhaustive coverage layer.
- Server startup can race database migrations or asset preparation -> Make the Playwright server command run the existing asset setup before `go run ./cmd/server`, and wait on `/health` before executing tests.
- HTMX interactions can be timing-sensitive -> Assert on user-visible post-action state and navigation outcomes instead of fixed sleeps.
- Invoice generation may depend on exact seeded entries and rates -> Build deterministic setup through browser actions within the test or through a minimal helper that prepares only the data required by that test.
- Auth-disabled test mode does not cover production login behavior -> Keep auth coverage out of scope for this change and add a later dedicated capability if browser-level auth tests are needed.

## Migration Plan

Add Playwright dependencies, configuration, tests, and scripts under `invoice-service`. Existing Go tests and runtime behavior remain unchanged. Rollback is removing the Playwright config, browser test files, package scripts, and dependency lockfile updates.

## Open Questions

None for the initial implementation. The suite should start with no-auth smoke and invoice workflow coverage, then expand only when specific browser regressions or workflows justify it.
