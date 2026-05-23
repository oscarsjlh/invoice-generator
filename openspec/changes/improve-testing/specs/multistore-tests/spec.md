## ADDED Requirements

### Requirement: MultiStore opens per-user databases on demand
The system SHALL create a per-user SQLite database at `{dir}/{userID}/invoices.db` on first access, apply data migrations, and cache the store. Subsequent accesses SHALL return the cached store.

#### Scenario: First access creates database
- **WHEN** `ForUser` is called for a user ID that has no existing database
- **THEN** a new SQLite database is created at the correct path, migrations are applied, and the store is returned

#### Scenario: Subsequent access returns cached store
- **WHEN** `ForUser` is called twice for the same user ID
- **THEN** the same `*Store` instance is returned (same pointer)

### Requirement: MultiStore evicts least recently used stores
The system SHALL evict the least recently used store when the cache exceeds the configured maximum. Evicted stores SHALL be closed.

#### Scenario: Cache eviction when over limit
- **WHEN** `ForUser` is called for more users than the configured maximum store count
- **THEN** the least recently used store is closed and removed from the cache

### Requirement: MultiStore sweeps idle connections
The system SHALL periodically close stores that have not been accessed within the idle timeout. The sweep runs on a background goroutine.

#### Scenario: Idle store is closed after timeout
- **WHEN** a store has not been accessed for longer than the idle timeout
- **THEN** the store is closed and removed from the cache on the next sweep cycle

### Requirement: MultiStore SetLegacyStore provides fallback
The system SHALL support setting a legacy store for backwards compatibility when auth is disabled.

#### Scenario: Legacy store is accessible
- **WHEN** `SetLegacyStore` is called with a valid store
- **THEN** subsequent calls to the legacy store path use the provided store

### Requirement: MultiStore Exists checks database presence
The system SHALL check whether a per-user database file exists on disk for a given user ID.

#### Scenario: Database exists
- **WHEN** `Exists` is called for a user ID whose database has been created
- **THEN** true is returned

#### Scenario: Database does not exist
- **WHEN** `Exists` is called for a user ID whose database has not been created
- **THEN** false is returned
