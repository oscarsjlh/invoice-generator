## Context

The invoice-service has 13 test files covering DB CRUD, handler helpers, PDF formatting, and config loading. However, entire packages are at 0% coverage: `internal/auth` (WebAuthn flows, session management), `internal/db/authdb.go` (auth database CRUD), `internal/db/multistore.go` (per-user DB management), CSRF and security middleware, and `internal/email`. Test infrastructure has duplicated `findMigrationsDir` helpers and unused fixture functions. Overall coverage is 22.3%.

All tests use `modernc.org/sqlite` (in-process, no CGO) in temporary directories via `t.TempDir()`. Tests use `testing.T.Parallel()` throughout. Assertions use `stretchr/testify` in handler tests and raw `testing` package in db tests.

## Goals / Non-Goals

**Goals:**
- Add unit tests for all public functions in `internal/auth/` (WebAuthnManager, SessionManager, WebAuthnUser)
- Add unit tests for all public methods in `internal/db/authdb.go` (AuthDB: user/credential/session CRUD)
- Add unit tests for `internal/db/multistore.go` (ForUser, LRU eviction, idle sweeping)
- Add unit tests for CSRFMiddleware and SecurityHeadersMiddleware
- Add unit tests for untested handler methods: `updateEntry`, `editEntryForm`, `updateRate`, `editRateForm`, `dashboard`
- Add unit tests for `internal/email/send.go` with a testable SMTP interface
- Add `NewTestAuthDB` helper to `testutil/` and `AuthMigrationsDir` for auth DB testing
- Add a `SampleSettings` factory to eliminate 14-field struct duplication
- Consolidate duplicated `findMigrationsDir` into `testutil.MigrationsDir`
- Remove or use `SampleEntry`, `SampleRate`, `AssertHTMLContains` in `testutil/`
- Fix `internal/db/settings_test.go` to use `package db` instead of `package db_test`

**Non-Goals:**
- E2E tests that spin up a real HTTP server
- Testing `internal/ocr/` (requires external AWS Bedrock service)
- Testing `GenerateInvoicePDF` (requires `typst` binary)
- Testing `cmd/` entrypoints (thin main functions)
- Testing the `static/` or `templates/` packages
- Adding mock libraries, testcontainers, or any new test dependencies
- Changing any production code interfaces (no refactoring to enable testing)

## Decisions

### Decision 1: AuthDB tests use real SQLite with auth-migrations

**Choice:** Create a temporary SQLite database with auth-migrations applied, same pattern as the existing data DB tests.

**Rationale:** The existing pattern works well — `modernc.org/sqlite` is fast, no external dependencies, and tests are parallel-safe with temp dirs. The auth DB uses the same SQLite driver and has its own `migrateAuthDB` path. We add `testutil.NewTestAuthDB(t)` that mirrors `NewTestDB`.

**Alternatives considered:**
- In-memory SQLite with `:memory:` — rejected because temp files give us better isolation for parallel tests and match the existing pattern.

### Decision 2: WebAuthnManager tests use real WebAuthn library, not mocks

**Choice:** Test `BeginRegistration`, `FinishRegistration`, `BeginLogin`, `FinishLogin` using the real `go-webauthn/webauthn` library against a real AuthDB. Sessions are managed in-memory (the manager's `sessions` map).

**Rationale:** The go-webauthn library is the production dependency. Testing against the real library ensures we catch API changes and verify actual cryptographic operations work. The library itself is well-tested, so we're testing our integration, not reinventing crypto. Each test creates a fresh `WebAuthnManager` with a temp AuthDB.

**Alternatives considered:**
- Mock the `webauthn.WebAuthn` interface — rejected because it would test against a mock, not the real library, and setting up realistic credential creation/assertion responses is more complex than using the real thing.

### Decision 3: SessionManager tests use httptest.ResponseRecorder

**Choice:** Test `CreateSession` and `DestroySession` by passing `httptest.NewRecorder()` and verifying the cookies set on the response.

**Rationale:** SessionManager's only interaction with the HTTP layer is setting/deleting cookies. `httptest.ResponseRecorder` captures headers and cookies directly. No need for a full HTTP server.

**Alternatives considered:**
- Real HTTP server — rejected as overkill for cookie verification.

### Decision 4: Email tests make `SendInvoice` accept a dialer interface

**Choice:** Refactor `SendInvoice` to accept a `Dialer` interface (with a `DialAndSend` method) as a parameter, default to the real `gomail.Dialer`. Tests pass a mock dialer that records the message without sending.

**Rationale:** The current `SendInvoice` creates `gomail.NewDialer` internally, making it impossible to test without a real SMTP server. Extracting the dialer as a parameter with a default preserves backward compatibility. This is the only production code change needed.

**Alternatives considered:**
- Start a local SMTP server in tests — rejected as flaky and heavyweight.
- Use `gomail.SendFunc` or `gomail.SendCloser` — rejected because `gomail.Dialer` has a `DialAndSend` method directly; a simple interface wrapping that is cleaner.

### Decision 5: Middleware tests use httptest with mock handlers

**Choice:** Test `CSRFMiddleware` and `SecurityHeadersMiddleware` by wrapping a mock handler and sending `httptest` requests through the middleware chain, verifying the response.

**Rationale:** Middleware is pure HTTP transformation — it takes a request, optionally modifies it or rejects it, and passes to the next handler. Testing through `httptest` with a simple "200 OK" inner handler is the standard Go pattern for middleware testing.

**Alternatives considered:**
- Test through full `App` handler chain — rejected because it couples middleware tests to auth and routing concerns unnecessarily.

### Decision 6: Test fixtures in testutil, not per-package

**Choice:** Add `testutil.SampleSettings(t)` returning a `db.Settings` with all 14 fields populated. Also add factory helpers for entries and rates that actually create DB records (matching the existing `NewTestDB` pattern).

**Rationale:** The same Settings struct literal appears in 4+ test functions. A factory function in `testutil` eliminates duplication and makes test data consistent. Using the `testutil` package means handler tests, db tests, and future tests all share the same fixtures.

**Alternatives considered:**
- Per-package `testdata` packages — rejected because a single shared package is simpler and we already have `testutil/`.

### Decision 7: Remove settings_test.go external test package

**Choice:** Change `package db_test` to `package db` in `settings_test.go`.

**Rationale:** `settings_test.go` is the only test using an external test package — all other DB tests use `package db`. This creates inconsistency and can lead to accidentally testing through the public API only. Since settings tests don't need external package constraints, move them inline.

**Alternatives considered:**
- Move all DB tests to `package db_test` — rejected because internal package tests can access unexported helpers (like `setupTestDB`) directly, which is useful for comprehensive testing.

## Risks / Trade-offs

- **WebAuthn tests may be slow:** Each BeginRegistration/FinishRegistration involves cryptographic operations. Mitigation: tests are parallel and use `t.Parallel()`. We limit to essential flow tests (4-6 test functions).
- **WebAuthn tests are complex to write:** The go-webauthn library's FinishingRegistration/FinishingLogin requires a valid `*http.Request` with proper headers set by the browser's WebAuthn API. Mitigation: research how other projects test this; may need to construct the protocol response payload manually or accept that only the Begin* methods are directly testable and Finish* methods get error-path tests.
- **AuthDB cleanup goroutine leak:** The `cleanupLoop` in `WebAuthnManager` starts a goroutine that runs indefinitely. Mitigation: tests will let the temp DB and manager fall out of scope; the goroutine will stop when the ticker channel is garbage collected. Since each test uses its own temp dir, there's no resource contention.
- **Email dialer extraction is a production code change:** Adding a parameter to `SendInvoice` requires updating its call site. Mitigation: this is a minimal, backward-compatible change — the existing signature is preserved with a wrapper or default parameter.

## Open Questions

- Can `FinishRegistration` and `FinishLogin` be tested without a browser? The go-webauthn library expects browser-generated attestation/assertion data in the request. Research needed: check go-webauthn's test suite for patterns, or limit to error-path tests (invalid session ID, missing session, wrong session kind).
- Should the email `Dialer` interface be defined in `internal/email/` or a shared package? Tentative: define in `internal/email/` since it's specific to email sending.
