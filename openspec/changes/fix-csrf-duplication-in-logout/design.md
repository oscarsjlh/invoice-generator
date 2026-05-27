## Context

The `logout` handler (`internal/handler/auth_handlers.go:205-220`) performs a manual CSRF double-submit check before destroying the session. It compares the `csrf_token` cookie value against the `csrf_token` form field value. This is logically identical to the check performed by `CSRFMiddleware` (`internal/handler/middleware_csrf.go`), which runs on all non-public POST/PUT/DELETE routes — including `/logout`, which is not in the `isPublicPath` list.

The check in the handler is therefore redundant. The middleware already validates the CSRF token before the handler runs. The duplication is a seam leak: the CSRF concern is implemented in two places that must stay synchronized.

## Goals / Non-Goals

**Goals:**
- Remove the manual CSRF double-submit check from the `logout` handler
- `/logout` is protected exclusively by `CSRFMiddleware`, like every other protected POST handler
- CSRF protection quality is unchanged — the same check runs, just in one place instead of two

**Non-Goals:**
- Changing the CSRF validation algorithm or cookie format
- Moving `/logout` into or out of the public paths list
- Modifying `CSRFMiddleware` behavior
- Calling `SessionCookie.ValidateCSRF` from the handler (that belongs in `deepen-session-cookie-module`)

## Decisions

**Decision 1: Remove the manual check entirely, do not add an alternative guard**

The `logout` handler should trust the middleware, same as `createEntry`, `saveSettings`, and every other protected POST handler. Adding an alternative guard (e.g., calling `SessionCookie.ValidateCSRF`) would be redundant if the middleware already validates — it would be the same seam leak in a different form.

**Decision 2: Keep `/logout` out of the public paths list**

`/logout` requires authentication (a session cookie must exist to destroy it). It should not be in `isPublicPath`. The CSRF middleware should run on it. This is unchanged.

## Risks / Trade-offs

**[Risk] If `CSRFMiddleware` is ever removed or bypassed, `/logout` loses CSRF protection** — but so does every other protected POST handler. The risk is uniform across all routes, not specific to logout.
→ **Mitigation**: The CSRF middleware is a core security component applied uniformly. Removing it would be a deliberate architectural change, not an accident.

**[Risk] Test coverage for CSRF-on-logout may need updating** — the current `TestLogoutCSRF` test likely exercises the handler's inline check. The test should instead verify that the middleware catches a missing/invalid CSRF token (400 response) before the handler runs.
→ **Mitigation**: Update the test to assert that a POST to `/logout` without a valid CSRF token receives a 400 response before reaching the handler.
