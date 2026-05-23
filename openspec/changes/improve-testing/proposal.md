## Why

Overall test coverage is 22.3% with entire critical packages at 0% (auth, CSRF middleware, `AuthDB`, `MultiStore`, OCR, email). The WebAuthn auth layer, session management, and CSRF protection — which gate all user data — have no tests at all. Test quality is further hampered by duplicated setup code, unused helper utilities, and no shared test fixtures. This change addresses the most impactful testing gaps and cleans up the existing test infrastructure to make future test writing easier.

## What Changes

- Add unit tests for `internal/auth/` (WebAuthn flow, session manager, models) — **CRITICAL**
- Add unit tests for `internal/db/authdb.go` (user/credential/session CRUD)
- Add unit tests for `internal/db/multistore.go` (per-user DB opening, LRU eviction, idle sweeping)
- Add unit tests for `CSRFMiddleware` and `SecurityHeadersMiddleware`
- Add unit tests for untested handler methods: `updateEntry`, `editEntryForm`, `updateRate`, `editRateForm`, `dashboard`
- Add unit tests for `internal/email/send.go`
- Add shared test fixtures to `testutil/` to eliminate struct duplication across test files
- Consolidate duplicated `findMigrationsDir` helpers into `testutil` package
- Add `NewTestAuthDB` helper to `testutil/` for auth database testing
- Remove unused test helper functions (`SampleEntry`, `SampleRate`, `AssertHTMLContains`) or use them in tests
- Fix `internal/db/settings_test.go` to use `package db` (internal) instead of `package db_test` (external) for consistency

## Capabilities

### New Capabilities
- `auth-tests`: Tests for WebAuthn registration/login flows, session creation/validation, and credential storage
- `authdb-tests`: Tests for the shared auth database user/credential/session CRUD operations
- `multistore-tests`: Tests for per-user database opening, LRU cache eviction, and idle connection sweeping
- `middleware-tests`: Tests for CSRFMiddleware (double-submit cookie validation) and SecurityHeadersMiddleware
- `handler-tests`: Tests for untested handler methods (updateEntry, editEntryForm, updateRate, editRateForm, dashboard)
- `email-tests`: Tests for SMTP email sending with mock transport
- `test-fixtures`: Shared test fixtures and improved test infrastructure (NewTestAuthDB, settings factory, dedup cleanup)

### Modified Capabilities
<!-- No existing spec-level requirements are changing -->

## Impact

- Affected code: `internal/auth/`, `internal/db/authdb.go`, `internal/db/multistore.go`, `internal/handler/` (CSRF, security, entries, rates, dashboard handlers), `internal/email/`, `internal/testutil/`
- New test dependencies: none (existing `stretchr/testify` and `modernc.org/sqlite` are sufficient)
- No breaking changes — purely additive test coverage
- Test execution time will increase moderately but all tests remain parallel-safe
