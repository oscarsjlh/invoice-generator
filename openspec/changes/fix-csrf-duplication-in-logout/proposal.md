## Why

The `logout` handler (`internal/handler/auth_handlers.go:208-216`) performs its own CSRF double-submit check comparing the `csrf_token` cookie against a form field. The `CSRFMiddleware` already performs the identical check on `/logout` (it is not a public path). This duplication is a **seam leak**: the same concern is implemented in two places that must stay synchronized. If CSRF policy changes (e.g., adding an `Origin` header check), both sites need updating. Removing the redundancy improves locality.

## What Changes

- Remove the manual CSRF double-submit check from the `logout` handler (lines 207-216)
- `/logout` continues to be protected by the `CSRFMiddleware` which already runs on it in the middleware chain
- The `logout` handler trusts the middleware — same as every other protected POST handler in the application
- CSRF test for logout (`auth_handlers_test.go`) is updated: tests verify the middleware catches CSRF violations, not the handler
- **No behavior change** — CSRF protection quality is identical; the middleware already enforces the same check

## Capabilities

### New Capabilities

<!-- None — this removes duplication, no new capability is introduced. -->

### Modified Capabilities

<!-- None — CSRF behavior is identical; the check just runs in one place instead of two. -->

## Impact

- `internal/handler/auth_handlers.go` — remove lines 207-216 (manual CSRF check block); `logout` handler becomes ~8 lines simpler
- `internal/handler/auth_handlers_test.go` — update logout CSRF tests: verify that a missing/invalid CSRF token is caught at the middleware layer (e.g., a 400 response), rather than testing the handler's own check
- No change to `CSRFMiddleware`, `csrf_cookie.go`, or any other file
- If done together with `deepen-session-cookie-module`, the `logout` handler may optionally call `SessionCookie.ValidateCSRF` as an additional guard, but this is optional
