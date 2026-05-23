## ADDED Requirements

### Requirement: WebAuthnManager BeginRegistration validates uniqueness
The system SHALL reject registration attempts for a username that already exists by returning a "username already taken" error from `BeginRegistration`. The system SHALL create a new user record and a pending registration session stored in-memory when the username is available.

#### Scenario: Registration with available username
- **WHEN** `BeginRegistration` is called with a new username
- **THEN** a new user is created in the AuthDB, a pending session of kind "reg" is stored with a generated session ID, and the session ID is returned alongside WebAuthn creation options

#### Scenario: Registration with taken username
- **WHEN** `BeginRegistration` is called with a username that already exists in the AuthDB
- **THEN** an error containing "username already taken" is returned

### Requirement: WebAuthnManager FinishRegistration completes passkey creation
The system SHALL retrieve the pending registration session by session ID, call the WebAuthn library to finish credential creation, save the credential to the AuthDB, and return the user ID and credential. It SHALL reject invalid or missing session IDs.

#### Scenario: FinishRegistration with valid session
- **WHEN** `FinishRegistration` is called with a session ID from a prior `BeginRegistration` and a valid HTTP request
- **THEN** the credential is saved to the AuthDB and the user ID is returned

#### Scenario: FinishRegistration with invalid session ID
- **WHEN** `FinishRegistration` is called with a session ID that does not exist
- **THEN** an error containing "no registration session found" is returned

### Requirement: WebAuthnManager BeginLogin validates user existence
The system SHALL look up the user by username and return a pending login session when the user exists. It SHALL reject login attempts for nonexistent users.

#### Scenario: BeginLogin for existing user
- **WHEN** `BeginLogin` is called with a username that exists in the AuthDB
- **THEN** a pending session of kind "login" is stored with a generated session ID, and the session ID and user ID are returned

#### Scenario: BeginLogin for nonexistent user
- **WHEN** `BeginLogin` is called with a username that does not exist
- **THEN** an error containing "user not found" is returned

### Requirement: SessionManager creates and destroys sessions
The system SHALL set an HttpOnly session cookie (`invoice_session`) and a non-HttpOnly CSRF cookie (`csrf_token`) when creating a session, and SHALL clear both cookies when destroying a session.

#### Scenario: CreateSession sets cookies
- **WHEN** `CreateSession` is called with a valid user ID
- **THEN** the response includes an `invoice_session` cookie (HttpOnly, Path "/") and a `csrf_token` cookie (non-HttpOnly, Path "/"), and the session token is stored in the AuthDB

#### Scenario: DestroySession clears cookies
- **WHEN** `DestroySession` is called
- **THEN** the `invoice_session` and `csrf_token` cookies are set with MaxAge -1, and the session is deleted from the AuthDB

### Requirement: WebAuthnUser implements webauthn.User interface
The system SHALL provide a `WebAuthnUser` wrapper around `db.User` that satisfies the `webauthn.User` interface with correct ID, name, display name, and credentials methods.

#### Scenario: WebAuthnUser credentials are loaded
- **WHEN** a `WebAuthnUser` is created with a `db.User` that has associated credentials
- **THEN** `WebAuthnCredentials()` returns the credentials

#### Scenario: WebAuthnUser with empty credentials
- **WHEN** a `WebAuthnUser` is created with a `db.User` that has no credentials
- **THEN** `WebAuthnCredentials()` returns an empty slice (not nil)
