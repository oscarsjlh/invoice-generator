## Context

The invoice service is a Go `net/http` application with server-rendered templates, HTMX, SQLite, optional authentication, and PDF generation through Typst. Existing browser tests live under `invoice-service/tests/e2e` and currently run Chromium against a Docker Compose app with `AUTH_ENABLED=false`.

The production confidence gate must cover two supported runtime modes:

- Auth-enabled mode, where each authenticated user gets an isolated SQLite database under `USER_DB_DIR/{userID}/invoices.db`.
- No-auth compatibility mode, where requests use a legacy shared SQLite database.

Auth-enabled parallelism can be achieved by seeding a fresh user and session per test. No-auth parallelism needs separate app instances because a single legacy store would be shared across overlapping tests.

## Goals / Non-Goals

**Goals:**

- Treat Chromium Playwright tests as a production release blocker.
- Cover both auth-enabled and no-auth modes.
- Seed auth sessions through compiled e2e-only HTTP helpers instead of automating WebAuthn in every workflow.
- Support two parallel workers initially, with a design that can scale to four.
- Keep verification user-visible: rendered UI, redirects, table rows, public page content, and public HTTP responses.
- Capture enough artifacts to diagnose CI failures without rerunning locally.

**Non-Goals:**

- Do not include OCR import in the initial production gate.
- Do not add visual regression snapshot assertions to the blocking gate.
- Do not parse full PDF text content in Playwright.
- Do not make Firefox/WebKit blocking for normal PRs.
- Do not use test helper endpoints for business workflow verification.

## Decisions

### 1. Run both auth-on and no-auth modes

The gate SHALL run browser workflows against both production auth mode and no-auth compatibility mode.

Rationale: Both modes are supported product behavior. Running only no-auth leaves session validation, redirects, logout, per-user store routing, and tenant isolation untested.

### 2. Seed sessions through compiled e2e-only helper routes

Auth-on tests SHALL create users and sessions through a helper route that exists only in binaries built with the `e2e` build tag. The route SHALL also require an e2e environment guard before doing setup work.

Rationale: Direct DB seeding would couple Playwright tests to schema details, token hashing, cookie names, and container file paths. A helper can call the real auth/session code while remaining absent from normal production binaries.

### 3. Use separate Docker Compose app services per mode

The Docker e2e environment SHALL run separate app services for auth-on and no-auth. No-auth SHALL run multiple app instances, initially two, each with its own `DATABASE_PATH`.

Rationale: Auth mode is process-level configuration. Separate services produce cleaner logs, failure artifacts, and state isolation. Multiple no-auth instances avoid race conditions caused by a shared legacy database.

### 4. Support parallel Playwright execution

The suite SHALL support two parallel workers initially. Tests SHALL be isolated enough that correctness does not depend on serial execution.

Rationale: `workers: 1` hides shared-state bugs and makes the suite slower as coverage grows. Two workers exposes isolation problems while keeping Docker Compose complexity manageable.

### 5. Keep setup helpers separate from verification

E2E helper endpoints SHALL be used for setup only, such as creating sessions or allocating worker-specific state. Browser workflow verification SHALL use visible UI or public HTTP responses.

Rationale: The gate should prove the app can be used end to end. Helper endpoints are acceptable for expensive or impractical setup, but they must not replace the actual workflow under test.

### 6. Organize specs by mode with shared behavior helpers

Reusable workflow functions SHALL live in shared Playwright helpers. Mode-specific specs SHALL remain separate for report readability and setup clarity.

Rationale: Auth-on and no-auth should stay behaviorally aligned without hiding important mode-specific setup and failure context.

### 7. Keep Chromium as the blocking browser

Chromium desktop SHALL be the required production gate. Firefox/WebKit MAY run on a scheduled non-blocking job.

Rationale: The app is mostly server-rendered HTML with HTMX. Workflow correctness, auth boundaries, PDF, and persistence are higher-risk than cross-browser layout differences.

### 8. Collect failure artifacts

The gate SHALL collect traces, screenshots, video-on-failure, an HTML report, browser console logs, and per-service Docker logs labelled by mode/worker.

Rationale: A release-blocking failure must be diagnosable from CI artifacts.

## Risks / Trade-offs

- Compiled helper routes add sensitive setup capabilities -> Build them only with `-tags=e2e`, require an e2e env guard, and do not publish images built with the tag.
- Multiple app services increase Compose complexity -> Start with two workers and name services predictably (`auth-on`, `noauth-1`, `noauth-2`).
- Retrying tests can hide flakes -> Allow one retry initially, publish retry visibility, and plan to move to zero retries after the suite is stable.
- PDF assertions may miss content regressions -> Verify invoice content in HTML preview and use the PDF test to prove the route and renderer produce a valid PDF response.
- Firefox/WebKit can find real issues later -> Run them scheduled and non-blocking until they justify becoming release blockers.

## Migration Plan

1. Add the `production-e2e-gate` spec and tasks.
2. Add e2e-only helper route code and tests proving it is unavailable without the build tag/env guard.
3. Update Docker Compose and build commands for auth-on and no-auth app services.
4. Refactor Playwright config and tests into shared helpers and mode-specific specs.
5. Add workflow coverage incrementally, keeping the existing smoke suite passing throughout.
6. Enable artifact collection and document CI/local commands.
7. Wire the Chromium gate into PR/main/release CI with path filters, and add scheduled non-blocking Firefox/WebKit later.

## Open Questions

None. Initial worker count, browser matrix, artifact policy, helper usage, and coverage scope have been decided.
