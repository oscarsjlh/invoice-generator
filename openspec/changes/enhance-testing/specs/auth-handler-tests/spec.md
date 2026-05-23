## ADDED Requirements

### Requirement: Auth handler tests cover login and registration flows
The system SHALL have tests for `loginPage`, `registerPage`, `beginLogin`, `beginRegistration`, `finishLogin`, `finishRegistration`, and `logout` handlers.

#### Scenario: loginPage renders login form
- **WHEN** `loginPage` is called via HTTP
- **THEN** the response is 200 OK containing a login form

#### Scenario: registerPage renders registration form
- **WHEN** `registerPage` is called via HTTP
- **THEN** the response is 200 OK containing a registration form

#### Scenario: logout clears session and redirects
- **WHEN** `logout` is called with a valid session
- **THEN** the response is 303 SeeOther redirecting to `/login`
