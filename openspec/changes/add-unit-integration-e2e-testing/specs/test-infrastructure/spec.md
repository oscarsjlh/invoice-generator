## ADDED Requirements

### Requirement: Test database factory
The test infrastructure SHALL provide a factory function that creates an isolated SQLite database for each test, with WAL mode enabled and migrations applied.

#### Scenario: Factory creates a fresh database
- **WHEN** the factory is invoked
- **THEN** a new temporary directory is created, a SQLite database is opened in that directory, and all migration files from the configured migrations directory are executed

#### Scenario: Factory cleans up after test
- **WHEN** the test completes (via t.Cleanup or t.Finish)
- **THEN** the temporary directory and all its contents are removed from disk

### Requirement: Storer interface for dependency injection
The database layer SHALL expose a `Storer` interface that defines all CRUD operations, allowing handlers to be tested with mock implementations.

#### Scenario: Store implements Storer
- **WHEN** `*db.Store` is used where a `Storer` is expected
- **THEN** it satisfies the interface without any wrapper or adapter code

#### Scenario: Mock store can be created for handler tests
- **WHEN** a test needs to verify handler behavior without touching the database
- **THEN** a mock `Storer` can be constructed with configurable return values and call tracking

### Requirement: Makefile test targets
The project SHALL provide Makefile targets for running tests at different scopes.

#### Scenario: Run all tests
- **WHEN** `make test` is executed
- **THEN** all Go tests are run with race detection enabled and a single count (no cache)

#### Scenario: Run tests with coverage report
- **WHEN** `make test-cover` is executed
- **THEN** tests run with coverage profiling and an HTML coverage report is generated in the project root

#### Scenario: Run fast tests only
- **WHEN** `make test-fast` is executed
- **THEN** only unit and integration tests are run; e2e-tagged tests are excluded

### Requirement: Test utility helpers
The test infrastructure SHALL provide helper functions for common test operations.

#### Scenario: Helper creates test entries with valid data
- **WHEN** a test needs sample entry data
- **THEN** a helper function returns a fully populated `db.Entry` with valid date, category, and hours

#### Scenario: Helper creates test rates with valid data
- **WHEN** a test needs sample rate data
- **THEN** a helper function returns a fully populated `db.Rate` with valid category, dates, and rate value

### Requirement: Assert helpers for HTTP responses
The test infrastructure SHALL provide helpers to assert on HTTP response properties.

#### Scenario: Assert status code
- **WHEN** an HTTP response is received during a test
- **THEN** the helper can verify the status code matches the expected value with a clear error message

#### Scenario: Assert response body contains text
- **WHEN** an HTML response is received during a test
- **THEN** the helper can verify that specific strings appear in the response body (for checking form fields, notices, etc.)
