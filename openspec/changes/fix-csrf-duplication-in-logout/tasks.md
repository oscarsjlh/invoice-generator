## 1. Remove duplicate CSRF check from logout handler

- [x] 1.1 In `internal/handler/auth_handlers.go`, remove the manual CSRF double-submit check block (lines 207-216: `if err := r.ParseForm(); ...`)
- [x] 1.2 The `logout` handler body now contains only: `a.sessions.DestroySession(w, r)` and redirect to `/login?notice=signed_out`
- [x] 1.3 Remove unused `net/http` dependency if no longer needed (still needed — used by other handlers in the file)

## 2. Update logout CSRF test

- [x] 2.1 In `internal/handler/auth_handlers_test.go`, find the logout CSRF test (renamed from `TestLogoutWithoutCSRFCookie`)
- [x] 2.2 Update the test: instead of testing the handler's inline check, test that the CSRF middleware catches a missing/invalid token before the handler runs
- [x] 2.3 The test should send `POST /logout` without a valid CSRF token and assert a `400 Bad Request` response
- [x] 2.4 The test should also verify that a valid CSRF token results in a redirect to `/login` (existing happy-path test)

## 3. Verification

- [x] 3.1 Run `go test ./internal/handler/... -run Logout` — all logout tests pass
- [x] 3.2 Run `go test ./internal/handler/... -run CSRF` — all CSRF middleware tests pass
- [x] 3.3 Run `go test ./...` — full test suite passes
- [ ] 3.4 Manual test: log in, click Logout, verify successful logout and redirect
- [ ] 3.5 Manual test: attempt `POST /logout` via `curl` without CSRF token, verify 400 response
- [ ] 3.6 Manual test: attempt `POST /logout` via browser with valid CSRF token, verify logout succeeds
