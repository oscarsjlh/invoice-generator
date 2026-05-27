## 1. Deepen SessionManager into SessionCookie

- [x] 1.1 In `internal/auth/session.go`, rename `SessionManager` → `SessionCookie`
- [x] 1.2 Rename `NewSessionManager` → `NewSessionCookie` (same parameters: `*db.AuthDB`, `time.Duration`, `bool`)
- [x] 1.3 Rename `CreateSession` → `Set` — same signature `(w http.ResponseWriter, r *http.Request, userID int64) error`
- [x] 1.4 Rename `GetUserFromRequest` → `Get` — same signature `(r *http.Request) (*db.User, error)`
- [x] 1.5 Rename `DestroySession` → `Clear` — same signature `(w http.ResponseWriter, r *http.Request) error`
- [x] 1.6 Add `ValidateCSRFToken(r *http.Request, formValue string) bool` standalone function — reads `csrf_token` cookie, compares to `formValue` (decided: standalone fn, not method, since it needs no receiver state)
- [x] 1.7 Add `EnsureCSRF(w http.ResponseWriter, r *http.Request)` method (instance) + `EnsureCSRFCookie(w, r, secure)` standalone function (for no-auth path)
- [x] 1.8 Move CSRF token generation into shared internal helper `generateCSRFToken() []byte`
- [x] 1.9 Run `go test ./internal/auth/...` — auth package tests pass with renamed methods

## 2. Update CSRF middleware to delegate

- [x] 2.1 In `internal/handler/middleware_csrf.go`, updated to call `auth.ValidateCSRFToken(r, extractedToken)` instead of inline comparison
- [x] 2.2 Replaced the inline double-submit comparison with a call to `auth.ValidateCSRFToken(r, extractedToken)`
- [x] 2.3 The middleware retains `extractCSRFToken(r)` — it still extracts from headers and form fields
- [x] 2.4 Run `go test ./internal/handler/... -run CSRF` — CSRF middleware tests pass

## 3. Remove ensureCSRFCookie and consolidate

- [x] 3.1 Removed `internal/handler/csrf_cookie.go` — `ensureCSRFCookie` function deleted
- [x] 3.2 In `internal/handler/app.go`, replaced calls with `auth.EnsureCSRFCookie(w, r, secure)` (no-auth path) and `a.sessionCookie.EnsureCSRF(w, r)` (auth path)
- [x] 3.3 Verified `EnsureCSRF` uses `IsSecure` logic internally
- [x] 3.4 Run `go build ./...` — compiles cleanly (no references to removed `ensureCSRFCookie`)

## 4. Update logout handler

- [x] 4.1 In `internal/handler/auth_handlers.go`, manual CSRF check was already removed in `fix-csrf-duplication-in-logout`
- [x] 4.2 The logout handler now calls `a.sessionCookie.Clear(w, r)` and redirects
- [x] 4.3 No explicit `ValidateCSRF` guard added — design decision 1 says to trust the middleware
- [x] 4.4 Logout test in `auth_handlers_test.go` already updated — tests CSRF through middleware
- [x] 4.5 Run `go test ./internal/handler/... -run Logout` — logout tests pass

## 5. Update all references

- [x] 5.1 In `cmd/server/main.go`, updated `NewSessionManager` → `NewSessionCookie`, variable `sessionManager` → `sessionCookie`
- [x] 5.2 In `internal/handler/app.go`, updated field type `sessions *auth.SessionManager` → `sessionCookie *auth.SessionCookie` and all method calls
- [x] 5.3 In `internal/handler/auth_handlers.go`, updated all session method calls
- [x] 5.4 In `internal/handler/auth_handlers_test.go`, updated `SessionManager` references → `SessionCookie`, `sm` → `sc`
- [x] 5.5 In `internal/handler/middleware_test.go`, updated `ta.sm.CreateSession` → `ta.sc.Set`
- [x] 5.6 In `internal/auth/auth_test.go`, updated test setup and method calls, renamed test functions
- [x] 5.7 Run `go build ./...` — compiles cleanly with zero references to `SessionManager`

## 6. Verification

- [x] 6.1 Run `go test ./...` — full test suite passes (all 9 packages)
- [x] 6.2 Run `go vet ./...` — no vet warnings
- [ ] 6.3 Manual test: register a user, verify `invoice_session` and `csrf_token` cookies are set correctly
- [ ] 6.4 Manual test: log in with WebAuthn, verify cookies set, navigate protected routes, verify session persists
- [ ] 6.5 Manual test: log out, verify both cookies are cleared, verify redirect to /login
- [ ] 6.6 Manual test: attempt POST to protected route without CSRF token, verify 400 response from middleware
- [ ] 6.7 Manual test: attempt POST to `/logout` without CSRF token, verify 400 response from middleware
