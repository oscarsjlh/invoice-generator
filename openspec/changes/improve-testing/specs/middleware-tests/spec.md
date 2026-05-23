## ADDED Requirements

### Requirement: CSRFMiddleware validates double-submit cookie
The system SHALL reject unsafe methods (POST/PUT/PATCH/DELETE) on non-public paths when the `csrf_token` cookie value does not match the request's CSRF token (from header or form body). It SHALL allow safe methods (GET/HEAD/OPTIONS) and public paths through without CSRF validation.

#### Scenario: POST with matching CSRF token in header
- **WHEN** a POST request to a protected path includes a `csrf_token` cookie and an `X-CSRF-Token` header with the same value
- **THEN** the request passes through to the next handler

#### Scenario: POST with matching CSRF token in form body
- **WHEN** a POST request to a protected path includes a `csrf_token` cookie and a form field `csrf_token` with the same value
- **THEN** the request passes through to the next handler

#### Scenario: POST with mismatched CSRF token
- **WHEN** a POST request to a protected path includes a `csrf_token` cookie and a CSRF token that differs
- **THEN** the request is rejected with HTTP 400 "invalid request"

#### Scenario: POST without CSRF cookie
- **WHEN** a POST request to a protected path has no `csrf_token` cookie
- **THEN** the request is rejected with HTTP 400 "invalid request"

#### Scenario: GET requests bypass CSRF
- **WHEN** a GET request is made to any path
- **THEN** CSRF validation is skipped and the request passes through

#### Scenario: Public path bypasses CSRF
- **WHEN** a POST request is made to `/login`, `/register`, or `/static/*`
- **THEN** CSRF validation is skipped and the request passes through

### Requirement: SecurityHeadersMiddleware sets security headers
The system SHALL set `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`, `X-Frame-Options: DENY`, `Permissions-Policy`, and `Content-Security-Policy` headers on every response.

#### Scenario: All security headers are set
- **WHEN** a request passes through SecurityHeadersMiddleware
- **THEN** the response includes `X-Content-Type-Options`, `Referrer-Policy`, `X-Frame-Options`, `Permissions-Policy`, and `Content-Security-Policy` headers with the correct values

#### Scenario: Inner handler still executes
- **WHEN** a request passes through SecurityHeadersMiddleware and the inner handler writes a response
- **THEN** the inner handler's response body is returned with the security headers added
