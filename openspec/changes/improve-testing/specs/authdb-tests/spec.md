## ADDED Requirements

### Requirement: AuthDB creates and retrieves users
The system SHALL support creating a user with a username and display name, and SHALL retrieve users by ID or username. Duplicate usernames SHALL be rejected.

#### Scenario: Create user
- **WHEN** `CreateUser` is called with a username and display name
- **THEN** a new user record is inserted and the user ID is returned

#### Scenario: Get user by ID
- **WHEN** `GetUserByID` is called with a valid user ID
- **THEN** the user with that ID is returned with their credentials loaded

#### Scenario: Get user by username
- **WHEN** `GetUserByUsernameForAuth` is called with a valid username
- **THEN** the user with that username is returned with all credentials loaded

#### Scenario: Get nonexistent user
- **WHEN** `GetUserByID` is called with an invalid user ID
- **THEN** nil is returned without an error

### Requirement: AuthDB manages WebAuthn credentials
The system SHALL support saving WebAuthn credentials for a user and retrieving all credentials for a user.

#### Scenario: Save credential
- **WHEN** `SaveCredential` is called with a valid credential and user ID
- **THEN** the credential is persisted in the database and can be retrieved by `GetCredentials`

#### Scenario: Get credentials for user with no credentials
- **WHEN** `GetCredentials` is called for a user ID that exists but has no credentials
- **THEN** an empty slice is returned without an error

### Requirement: AuthDB manages sessions with hashed tokens
The system SHALL create sessions by storing a SHA-256 hash of the session token in the database, SHALL validate sessions by comparing the token hash and checking expiry, and SHALL support session deletion.

#### Scenario: Create session
- **WHEN** `CreateSession` is called with a user ID and TTL
- **THEN** a random 32-byte session token is returned (plaintext) and its SHA-256 hash is stored in the database

#### Scenario: Validate valid session
- **WHEN** `ValidateSessionToken` is called with a valid, non-expired token
- **THEN** the associated User is returned

#### Scenario: Validate expired session
- **WHEN** `ValidateSessionToken` is called with an expired token
- **THEN** nil is returned without an error

#### Scenario: Validate invalid token
- **WHEN** `ValidateSessionToken` is called with a token that has no matching hash
- **THEN** nil is returned without an error

#### Scenario: Delete session
- **WHEN** `DeleteSession` is called with a valid token
- **THEN** the session is removed from the database and subsequent validation returns nil
