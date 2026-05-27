## Context

Session and CSRF cookie management is spread across four modules:

| Module | Responsibility |
|--------|---------------|
| `SessionManager` (`auth/session.go`) | Create/destroy session cookies, validate tokens via AuthDB, check `IsSecure` |
| `ensureCSRFCookie` (`handler/csrf_cookie.go`) | Lazily set a CSRF cookie on safe-method requests when one doesn't exist |
| `CSRFMiddleware` (`handler/middleware_csrf.go`) | Validate the double-submit token on POST/PUT/DELETE |
| `logout` handler (`handler/auth_handlers.go`) | Re-validates CSRF inline before destroying the session |

`SessionManager` is shallow: 3 of its 4 methods are thin wrappers over `AuthDB` + cookie manipulation. The CSRF lifecycle is fragmented — created in two places (`SessionManager.CreateSession` and `ensureCSRFCookie`), validated in two places (`CSRFMiddleware` and `logout` handler).

The `SessionCookie` concept deepens this: a single module that owns the complete session+CSRF cookie lifecycle behind four methods.

## Goals / Non-Goals

**Goals:**
- `SessionCookie` owns: session token creation, cookie setting, token validation, cookie clearing, CSRF token generation, and CSRF double-submit comparison
- `CSRFMiddleware` delegates token extraction+comparison to `SessionCookie.ValidateCSRF`
- `ensureCSRFCookie` is subsumed — `SessionCookie` handles lazy CSRF cookie creation
- The `logout` handler's manual CSRF check is removed (trusts the middleware, or calls `ValidateCSRF` as an explicit guard)
- `SessionCookie` is testable in isolation with `httptest.ResponseRecorder` and a mock `AuthDB`

**Non-Goals:**
- Changing the session token format or hash algorithm (SHA-256 remains)
- Changing cookie attributes (HttpOnly, SameSite=Lax, Secure, Path=/)
- Changing the CSRF double-submit pattern (cookies vs headers/form fields)
- Adding session refresh or sliding expiration (unchanged behavior)
- Adding a `SessionStore` interface — only one adapter exists (cookie-based), so this is a hypothetical seam

## Decisions

**Decision 1: `SessionCookie` stays in `internal/auth/`, not `internal/handler/`**

Rationale: The module depends on `db.AuthDB` (same package boundary as `SessionManager` today). Moving it to `handler/` would create a dependency from `auth/` on `handler/` or require extracting cookie utilities into a shared package. Keeping it in `auth/` preserves the current dependency direction: `handler/` depends on `auth/`.

**Decision 2: `ValidateCSRF(r *http.Request, formValue string) bool` signature**

The method reads the `csrf_token` cookie from the request, compares it against the provided `formValue`, and returns a boolean. This is the canonical comparison. The `formValue` comes from the CSRF middleware (which already extracts it from headers and form fields). The middleware's `extractCSRFToken` logic stays — it handles the multi-source extraction (header, multipart form, form-encoded body). `SessionCookie` only does the comparison.

Alternative considered: `ValidateCSRF` takes the raw request and does both extraction and comparison. Rejected because extraction logic is middleware-specific (headers vs form body) and the middleware already handles it well.

**Decision 3: Lazy CSRF cookie creation is a method on `SessionCookie`, not a standalone function**

`SessionCookie.EnsureCSRF(w, r)` creates a CSRF cookie if one doesn't exist on the request. This replaces `ensureCSRFCookie` in `csrf_cookie.go`. The `IsSecure` logic is reused internally.

Alternative considered: Fold CSRF cookie creation into `Set`. Rejected because `Set` is called once at login/registration, but the CSRF cookie may need to be set on any safe request (e.g., a user with a stale CSRF cookie visiting the login page). Separate methods for separate concerns.

**Decision 4: `SessionCookie` is a concrete struct, not an interface**

Only one adapter exists (cookie-based sessions). Adding an interface now is premature — it would be a hypothetical seam. If OAuth or JWT sessions are added later, extract the interface at that point when a second adapter makes it a real seam.

## Risks / Trade-offs

**[Risk] `CSRFMiddleware` now depends on `SessionCookie`** — the middleware currently depends only on `http.Handler`. Adding a dependency on `SessionCookie` couples middleware to the auth module.
→ **Mitigation**: The middleware already implicitly depends on session cookies (it reads `csrf_token` from `r.Cookie`). Making the dependency explicit is clarifying, not adding. If decoupling is desired later, extract a `CSRFValidator` interface.

**[Risk] Name change breaks all references** — `SessionManager` → `SessionCookie` is a rename that touches `cmd/server/main.go`, `app.go`, `auth_handlers.go`, and all test files.
→ **Mitigation**: Mechanical rename. Use IDE refactoring tools to update all references atomically. The compiler will catch any missed references.

**[Trade-off] `SessionCookie` depends on `http.ResponseWriter` and `*http.Request`** — the module is coupled to `net/http`. This is acceptable: it is a web-specific module in a web application. Extracting HTTP coupling would require an abstraction (`CookieWriter`?) that has no second adapter and adds indirection for no benefit.
