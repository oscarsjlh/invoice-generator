## ADDED Requirements

### Requirement: Tests use shared SampleSettings fixture
The system SHALL replace inline `db.Settings` struct literals with `testutil.SampleSettings()` in all test files where a full settings struct is constructed.

#### Scenario: All settings construction uses factory
- **WHEN** any test creates a `db.Settings` struct with 8+ fields
- **THEN** it calls `testutil.SampleSettings()` and modifies only the fields it needs to customize
