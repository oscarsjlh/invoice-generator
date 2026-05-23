## ADDED Requirements

### Requirement: Integration test exercises full HTTP stack
The system SHALL have an integration test that starts an `httptest.Server` using the app's `Routes()` mux and makes real HTTP requests through the complete middleware chain.

#### Scenario: GET / returns 200
- **WHEN** a GET request is made to `/` on the test server
- **THEN** the response is 200 OK

#### Scenario: POST to /entries with CSRF fails
- **WHEN** a POST request is made to `/entries` without a CSRF token
- **THEN** the response is 400 Bad Request (CSRF middleware active)

#### Scenario: Nonexistent path returns 404
- **WHEN** a GET request is made to a nonexistent path
- **THEN** the response is 404 Not Found
