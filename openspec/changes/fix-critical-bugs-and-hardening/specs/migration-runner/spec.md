## ADDED Requirements

### Requirement: Multi-statement SQL migrations must execute all statements

The system SHALL split each migration file into individual SQL statements and execute each one separately. Statement splitting SHALL handle semicolons that appear inside string literals correctly for the current migration files.

#### Scenario: Migration file with multiple CREATE TABLE statements
- **WHEN** a migration file contains three `CREATE TABLE` statements separated by semicolons
- **THEN** all three tables are created in the database

#### Scenario: Migration file with CREATE TABLE and INSERT statements
- **WHEN** a migration file contains a `CREATE TABLE` followed by `INSERT` statements
- **THEN** the table is created and all rows are inserted

#### Scenario: Migration execution failure rolls back
- **WHEN** a statement within a migration file fails
- **THEN** the migration is marked as failed and subsequent migrations are not executed

### Requirement: Migration version tracking

The system SHALL maintain a `schema_migrations` table that records which migration files have been successfully applied. The migration runner SHALL skip any migration file whose filename is already recorded in this table.

#### Scenario: First run applies all migrations
- **WHEN** the application starts with an empty database
- **THEN** all migration files are executed and their filenames are recorded in `schema_migrations`

#### Scenario: Second run skips already-applied migrations
- **WHEN** the application starts with an existing database that has migrations recorded
- **THEN** only migration files not yet recorded are executed

#### Scenario: Migration file added after initial run
- **WHEN** a new migration file is added to the migrations directory after the initial run
- **THEN** only the new migration file is executed on next startup
