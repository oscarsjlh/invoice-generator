## ADDED Requirements

### Requirement: NewTestAuthDB provides temp auth database
The system SHALL provide a `testutil.NewTestAuthDB(t)` helper that creates a temporary SQLite database with auth migrations applied, registers cleanup, and returns an `*db.AuthDB` ready for testing.

#### Scenario: Test helper creates ready-to-use AuthDB
- **WHEN** `NewTestAuthDB(t)` is called in a test
- **THEN** a temporary database is created, auth migrations are applied, and the returned `*db.AuthDB` can immediately create users and sessions

### Requirement: AuthMigrationsDir finds auth-migrations directory
The system SHALL provide a `testutil.AuthMigrationsDir(t)` helper that locates the `auth-migrations` directory by searching upward from the working directory.

#### Scenario: Migrations directory found from test package
- **WHEN** `AuthMigrationsDir(t)` is called from a test in any `internal/` package
- **THEN** the path to the `auth-migrations` directory is returned

### Requirement: SampleSettings provides a complete settings fixture
The system SHALL provide a `testutil.SampleSettings()` function returning a `db.Settings` with all 14+ fields populated with valid test data (business name, address, bank details, payment terms, tax info, logo data).

#### Scenario: SampleSettings returns valid settings
- **WHEN** `SampleSettings()` is called
- **THEN** a populated `db.Settings` struct is returned with non-empty business name, address, and bank details

### Requirement: Duplicated helpers are consolidated into testutil
The system SHALL use `testutil.MigrationsDir` consistently across all test packages, eliminating the duplicated `findMigrationsDir` in `internal/db/entries_test.go`.

#### Scenario: All DB tests use testutil.MigrationsDir
- **WHEN** any test in `internal/db/` needs the migrations directory
- **THEN** it calls `testutil.MigrationsDir(t)` instead of a local `findMigrationsDir`

### Requirement: settings_test.go uses internal package
The system SHALL change `internal/db/settings_test.go` from `package db_test` to `package db` for consistency with all other DB test files.

#### Scenario: Settings test imports as internal package
- **WHEN** `settings_test.go` is compiled
- **THEN** it uses `package db` and has access to unexported symbols in the `db` package
