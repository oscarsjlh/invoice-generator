## ADDED Requirements

### Requirement: CSRF validation for logout is handled exclusively by the middleware

The `logout` handler SHALL NOT perform its own CSRF double-submit check. CSRF validation for the `POST /logout` route SHALL be handled exclusively by the `CSRFMiddleware`, which runs on all non-public POST routes including `/logout`.

#### Scenario: Logout with valid CSRF token succeeds

- **WHEN** a client sends `POST /logout` with a valid `csrf_token` cookie and matching `X-CSRF-Token` header or form field
- **THEN** the CSRF middleware passes the request to the logout handler
- **AND** the logout handler destroys the session and redirects to `/login?notice=signed_out`

#### Scenario: Logout without CSRF token is rejected by middleware

- **WHEN** a client sends `POST /logout` without a `csrf_token` cookie or with a mismatched token
- **THEN** the CSRF middleware SHALL respond with HTTP 400
- **AND** the logout handler SHALL NOT execute (session is not destroyed)

#### Scenario: Logout handler does not duplicate CSRF logic

- **WHEN** reading the `logout` handler source code
- **THEN** there SHALL be no inline CSRF double-submit comparison (no code reading the `csrf_token` cookie and comparing it against a form value)
- **AND** the handler SHALL contain only session destruction and redirect logic
