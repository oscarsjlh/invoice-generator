## Why

`SessionManager` (`internal/auth/session.go`) has 4 public methods but 3 of them are thin wrappers over `AuthDB` methods plus cookie manipulation. The CSRF cookie lifecycle is split across `SessionManager` (sets the cookie on session creation), `ensureCSRFCookie` (sets it lazily on safe requests), `CSRFMiddleware` (validates on POST/PUT/DELETE), and the `logout` handler (re-validates inline). Four modules handle what should be a single concern: the session cookie and its CSRF partner. Deepening `SessionManager` into a `SessionCookie` module that owns the complete lifecycle improves locality and testability.

## What Changes

- Rename and deepen `SessionManager` → `SessionCookie` with a cleaner interface: `Set(w, userID)`, `Get(r) → *db.User`, `Clear(w, r)`, `ValidateCSRF(r, formValue) → bool`
- `Set` creates both `invoice_session` (HttpOnly) and `csrf_token` cookies atomically
- `Clear` removes both cookies atomically
- `ValidateCSRF` replaces the manual double-submit comparison in the `logout` handler and becomes the canonical CSRF validation
- `ensureCSRFCookie` in `csrf_cookie.go` is subsumed by `SessionCookie.Set` (called lazily) — the module owns cookie creation, not an external helper
- `CSRFMiddleware` calls `SessionCookie.ValidateCSRF` instead of implementing its own cookie-vs-token comparison
- `IsSecure` remains as an internal detail of the module

## Capabilities

### New Capabilities

- `session-cookie-module`: A deepened module that owns the complete session+CSRF cookie lifecycle behind four methods (`Set`, `Get`, `Clear`, `ValidateCSRF`), replacing scattered cookie logic across `SessionManager`, `ensureCSRFCookie`, `CSRFMiddleware`, and the `logout` handler.

### Modified Capabilities

<!-- None — session and CSRF behavior is preserved. The implementation location changes, not the behavior. -->

## Impact

- `internal/auth/session.go` — significant rewrite: rename `SessionManager` → `SessionCookie`, add `ValidateCSRF`, consolidate cookie management (115 → ~130 lines)
- `internal/handler/csrf_cookie.go` — `ensureCSRFCookie` is removed/replaced by calls to `SessionCookie`
- `internal/handler/middleware_csrf.go` — `CSRFMiddleware` delegates token extraction+comparison to `SessionCookie.ValidateCSRF`
- `internal/handler/auth_handlers.go` — `logout` handler removes manual CSRF check; calls `SessionCookie.ValidateCSRF` if needed
- `internal/handler/app.go` — `authMiddleware` uses new `SessionCookie` interface; removes direct `ensureCSRFCookie` calls
- `cmd/server/main.go` — `NewSessionManager` → `NewSessionCookie` (name change)
- All test files that reference `SessionManager` — update to `SessionCookie` (name + method signature changes)
- `internal/auth/auth_test.go` — tests update for new interface; `SessionCookie` can be tested in isolation with `httptest.ResponseRecorder`
