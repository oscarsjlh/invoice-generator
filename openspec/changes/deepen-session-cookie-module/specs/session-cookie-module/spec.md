## ADDED Requirements

### Requirement: SessionCookie owns the complete session cookie lifecycle

The system SHALL provide a `SessionCookie` module that owns the complete lifecycle of session and CSRF cookies behind four methods: `Set`, `Get`, `Clear`, and `ValidateCSRF`. No other module SHALL directly set or read the `invoice_session` or `csrf_token` cookies.

#### Scenario: Set creates both cookies atomically

- **WHEN** `SessionCookie.Set(w, userID)` is called
- **THEN** an `invoice_session` cookie is set with HttpOnly=true, SameSite=Lax, Path=/, and the session TTL as MaxAge
- **AND** a `csrf_token` cookie is set with HttpOnly=false, SameSite=Lax, Path=/, and the session TTL as MaxAge
- **AND** both cookies are set with Secure=true when the connection is TLS or X-Forwarded-Proto is https (trusted proxy)
- **AND** the session token is hashed (SHA-256) and stored in the auth database

#### Scenario: Get validates and returns user

- **WHEN** `SessionCookie.Get(r)` is called with a request containing a valid `invoice_session` cookie
- **THEN** the method decodes the base64 token, validates the SHA-256 hash against the auth database, checks expiration
- **AND** returns the associated `*db.User`
- **WHEN** the cookie is missing, malformed, or expired
- **THEN** the method SHALL return nil (no error for missing/expired — this is a normal unauthenticated state)

#### Scenario: Clear removes both cookies

- **WHEN** `SessionCookie.Clear(w, r)` is called
- **THEN** both `invoice_session` and `csrf_token` cookies are set with MaxAge=-1 (deleted)
- **AND** the session token is deleted from the auth database

### Requirement: ValidateCSRF is the canonical CSRF check

The `SessionCookie.ValidateCSRF(r, formValue) bool` method SHALL be the single source of truth for CSRF double-submit comparison. It SHALL read the `csrf_token` cookie from the request and compare its value against the provided form value.

#### Scenario: Matching tokens pass validation

- **WHEN** `ValidateCSRF(r, "abc123")` is called
- **AND** the request contains a `csrf_token` cookie with value `"abc123"`
- **THEN** the method returns `true`

#### Scenario: Mismatched tokens fail validation

- **WHEN** `ValidateCSRF(r, "wrong")` is called
- **AND** the request contains a `csrf_token` cookie with value `"correct"`
- **THEN** the method returns `false`

#### Scenario: Missing cookie fails validation

- **WHEN** `ValidateCSRF(r, "anything")` is called
- **AND** the request has no `csrf_token` cookie
- **THEN** the method returns `false`

### Requirement: CSRF middleware delegates to SessionCookie

The `CSRFMiddleware` SHALL delegate the double-submit token comparison to `SessionCookie.ValidateCSRF` instead of implementing its own comparison. The middleware SHALL retain responsibility for extracting the CSRF token from request headers and form fields.

#### Scenario: CSRF middleware calls ValidateCSRF

- **WHEN** a POST request arrives at a protected route
- **AND** `CSRFMiddleware` extracts the CSRF token from the `X-CSRF-Token` header or form field
- **THEN** the middleware SHALL call `SessionCookie.ValidateCSRF(r, extractedToken)`
- **AND** if `ValidateCSRF` returns `false`, the middleware SHALL respond with 400

### Requirement: Lazy CSRF cookie creation is handled by SessionCookie

The `SessionCookie` module SHALL provide an `EnsureCSRF(w, r)` method that creates a `csrf_token` cookie if one does not already exist on the request. This SHALL replace the standalone `ensureCSRFCookie` function.

#### Scenario: Creates CSRF cookie when missing

- **WHEN** `SessionCookie.EnsureCSRF(w, r)` is called
- **AND** the request has no `csrf_token` cookie
- **THEN** a new random 16-byte CSRF token is generated and set as the `csrf_token` cookie

#### Scenario: No-op when CSRF cookie exists

- **WHEN** `SessionCookie.EnsureCSRF(w, r)` is called
- **AND** the request already has a `csrf_token` cookie
- **THEN** no new cookie is set (the existing cookie is preserved)
