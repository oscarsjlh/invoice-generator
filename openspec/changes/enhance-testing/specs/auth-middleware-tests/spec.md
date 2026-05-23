## ADDED Requirements

### Requirement: Auth middleware validates sessions and injects context
The system SHALL have tests for `authMiddleware` that verify session validation, user context injection, and legacy mode behavior.

#### Scenario: Valid session passes through
- **WHEN** a request with a valid session cookie passes through authMiddleware
- **THEN** the user is injected into the context and the request proceeds

#### Scenario: No session redirects to login
- **WHEN** a request without a session cookie passes through authMiddleware
- **THEN** the response is 303 SeeOther redirecting to `/login`
