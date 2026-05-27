## ADDED Requirements

### Requirement: Server publishes username validation policy

The system SHALL expose a public endpoint `GET /auth/username-policy` that returns the canonical username validation policy as JSON. The response SHALL include the validation regex pattern as a string, the human-readable validation message, and the normalization rules.

#### Scenario: Successful policy fetch

- **WHEN** a client requests `GET /auth/username-policy`
- **THEN** the response status is 200
- **AND** the response Content-Type is `application/json`
- **AND** the response body contains `pattern` (regex string), `message` (human-readable string), and `normalization` (description of normalization rules)

#### Scenario: Policy endpoint is public

- **WHEN** an unauthenticated client requests `GET /auth/username-policy`
- **THEN** the response status is 200 (no redirect to login)
- **AND** no session cookie is required

### Requirement: Client fetches and caches username policy

The client-side auth module (`static/auth.js`) SHALL fetch the username validation policy from `GET /auth/username-policy` before performing its first validation. The policy SHALL be cached in a module-level variable so subsequent validations use the cached value. If the fetch fails, the module SHALL fall back to the hardcoded regex and message values.

#### Scenario: Client uses cached policy

- **WHEN** the client has already fetched the username policy
- **AND** the user types a username and submits the form
- **THEN** the client validates against the cached policy without making a second HTTP request to the policy endpoint

#### Scenario: Client gracefully degrades on fetch failure

- **WHEN** the client attempts to fetch the policy
- **AND** the request fails (network error, 500, timeout)
- **THEN** the client falls back to the hardcoded regex and message
- **AND** validation proceeds normally with the fallback values
- **AND** no error is shown to the user for the policy fetch failure

### Requirement: Server-side validation is authoritative

The server-side `ValidateUsername` and `NormalizeUsername` functions SHALL remain the canonical implementation. The policy endpoint SHALL derive its values from the same source as these functions. Changing the validation rule SHALL require changing only `internal/auth/username.go`.

#### Scenario: Server rejects invalid username even when client validates incorrectly

- **WHEN** a client with a stale cached policy submits a username that passes the stale client validation
- **AND** the server's current policy rejects the username
- **THEN** the server SHALL respond with the current validation error message (not the stale client message)
- **AND** the client SHALL display the server's error message to the user
