## ADDED Requirements

### Requirement: DB tests use table-driven patterns
The system SHALL convert flat DB test functions in `entries_test.go`, `rates_test.go`, `invoices_test.go` to table-driven tests with `t.Run` subtests.

#### Scenario: Entry CRUD uses table-driven tests
- **WHEN** entry tests run
- **THEN** each operation (create, read, update, delete) is a subtest within `TestEntryCRUD`
