## ADDED Requirements

### Requirement: MultiStore is thread-safe under concurrent access
The system SHALL have a test that verifies `MultiStore.ForUser` is safe when called from multiple goroutines simultaneously.

#### Scenario: Concurrent ForUser calls succeed
- **WHEN** 20 goroutines call `ForUser` on different user IDs simultaneously
- **THEN** all calls succeed without race conditions or panics
