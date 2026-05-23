## 1. Test Infrastructure

- [x] 1.1 Add `NewTestAuthDB(t)` helper to `internal/testutil/helpers.go` that creates a temp SQLite DB, applies auth migrations, and returns `*db.AuthDB`
- [x] 1.2 Add `AuthMigrationsDir(t)` helper to `internal/testutil/helpers.go` that finds the `auth-migrations/` directory
- [x] 1.3 Add `SampleSettings()` factory to `internal/testutil/helpers.go` returning a fully populated `db.Settings` with all 14+ fields
- [x] 1.4 Consolidate `internal/db/entries_test.go` to use `testutil.MigrationsDir` instead of local `findMigrationsDir` — kept local helper due to import cycle (testutil imports db)
- [x] 1.5 Change `internal/db/settings_test.go` from `package db_test` to `package db`

## 2. AuthDB Tests (`internal/db/authdb_test.go`)

- [x] 2.1 Write tests for `OpenAuthDB`, `CreateUser`, `GetUserByID`, `GetUserByUsernameForAuth`
- [x] 2.2 Write tests for `SaveCredential`, `GetCredentials`
- [x] 2.3 Write tests for `CreateSession`, `ValidateSessionToken` (valid, expired, invalid), `DeleteSession`, `DeleteUser`

## 3. Auth Tests (`internal/auth/auth_test.go`)

- [x] 3.1 Write tests for `NewWebAuthnUser` and `WebAuthnUser` webauthn interface methods (WebAuthnID, WebAuthnName, WebAuthnDisplayName, WebAuthnCredentials)
- [x] 3.2 Write tests for `BeginRegistration` (success with new username, failure with taken username)
- [x] 3.3 Write tests for `FinishRegistration` error paths (invalid session ID, missing session)
- [x] 3.4 Write tests for `BeginLogin` (success with existing user, failure with nonexistent user)
- [x] 3.5 Write tests for `FinishLogin` error paths (invalid session ID, missing session, wrong session kind)
- [x] 3.6 Write tests for `SessionManager.CreateSession` and `SessionManager.DestroySession`
- [x] 3.7 Write tests for `SessionManager.GetUserFromRequest` (valid session cookie, missing cookie, invalid token, expired session)

## 4. MultiStore Tests (`internal/db/multistore_test.go`)

- [x] 4.1 Write tests for `NewMultiStore`, `ForUser` (creates DB on first access, returns cached store on second access)
- [x] 4.2 Write tests for `Exists` (true for existing user DB, false for nonexistent)
- [x] 4.3 Write tests for `SetLegacyStore` (store is accessible after setting)
- [x] 4.4 Write tests for LRU eviction (cache evicts oldest when exceeding maxStores)
- [x] 4.5 Write tests for idle sweeping (stores closed after idle timeout, by testing `Close` and verifying goroutine cleanup)

## 5. Middleware Tests (`internal/handler/middleware_test.go`)

- [x] 5.1 Write tests for `CSRFMiddleware`: POST with matching X-CSRF-Token header passes through
- [x] 5.2 Write tests for `CSRFMiddleware`: POST with matching form field passes through
- [x] 5.3 Write tests for `CSRFMiddleware`: POST with mismatched token returns 400
- [x] 5.4 Write tests for `CSRFMiddleware`: POST without csrf_token cookie returns 400
- [x] 5.5 Write tests for `CSRFMiddleware`: GET requests bypass CSRF
- [x] 5.6 Write tests for `CSRFMiddleware`: public paths (`/login`, `/register`, `/static/*`) bypass CSRF
- [x] 5.7 Write tests for `SecurityHeadersMiddleware`: all security headers are present and correct
- [x] 5.8 Write tests for helper functions: `isSafeMethod`, `extractCSRFToken`, `isPublicPath`

## 6. Handler Tests

- [x] 6.1 Write tests for `editEntryForm` in `internal/handler/entries_test.go` (valid entry renders form, invalid entry returns 404)
- [x] 6.2 Write tests for `updateEntry` in `internal/handler/entries_test.go` (success redirect, missing category returns form with error)
- [x] 6.3 Write tests for `editRateForm` in `internal/handler/rates_test.go` (valid rate renders form, invalid rate returns 404)
- [x] 6.4 Write tests for `updateRate` in `internal/handler/rates_test.go` (success redirect, missing category returns form with error)
- [x] 6.5 Write test for `dashboard` in `internal/handler/invoices_test.go` (renders successfully with content)

## 7. Email Tests

- [x] 7.1 Define `Dialer` interface in `internal/email/send.go` with `DialAndSend(*gomail.Message) error`
- [x] 7.2 Update `SendInvoice` to accept a `Dialer` parameter instead of constructing `gomail.Dialer` internally
- [x] 7.3 Update `SendInvoice` call site in handler to pass `gomail.NewDialer(...)`
- [x] 7.4 Write `internal/email/send_test.go`: `SendInvoice` with mock dialer composes correct message (from, to, subject, body, PDF attachment)

## 8. Cleanup & Verification

- [x] 8.1 Remove or integrate unused helpers in `testutil/` (`SampleEntry`, `SampleRate`, `AssertHTMLContains`)
- [x] 8.2 Run `go test -race -count=1 -tags='!e2e' ./...` and verify all new and existing tests pass
- [x] 8.3 Run `go test -race -count=1 ./...` to verify e2e tests still pass
- [x] 8.4 Verify overall test coverage has improved from ~22% and auth/CSRF packages are no longer at 0%
