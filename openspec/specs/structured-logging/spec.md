## ADDED Requirements

### Requirement: Log level configuration
The system SHALL support configurable log levels via the `LOG_LEVEL` environment variable. Valid values are `debug`, `info`, `warn`, and `error`. The default level SHALL be `info` when the variable is unset or invalid.

#### Scenario: Default log level
- **WHEN** `LOG_LEVEL` is not set
- **THEN** the system logs at `info` level and above

#### Scenario: Debug log level
- **WHEN** `LOG_LEVEL=debug`
- **THEN** the system logs at `debug` level and above (all messages)

#### Scenario: Invalid log level
- **WHEN** `LOG_LEVEL=verbose`
- **THEN** the system defaults to `info` level

### Requirement: Log format selection
The system SHALL support JSON and text log formats via the `LOG_FORMAT` environment variable. Valid values are `json` and `text`. The default format SHALL be `json`.

#### Scenario: Default format
- **WHEN** `LOG_FORMAT` is not set
- **THEN** logs are written as JSON

#### Scenario: Text format
- **WHEN** `LOG_FORMAT=text`
- **THEN** logs are written in human-readable text format

#### Scenario: Invalid format
- **WHEN** `LOG_FORMAT=xml`
- **THEN** logs default to JSON format

### Requirement: Logger injection via middleware
The system SHALL provide a `LoggerMiddleware` that creates a `*slog.Logger` with a unique request ID and injects it into the request context. The logger SHALL be accessible to downstream handlers via `r.Context()`.

#### Scenario: Logger available in handler
- **WHEN** a request passes through `LoggerMiddleware`
- **THEN** the request context contains a `*slog.Logger` with a `request_id` attribute

#### Scenario: Unique request IDs
- **WHEN** two concurrent requests pass through `LoggerMiddleware`
- **THEN** each request receives a logger with a different `request_id`

### Requirement: Logger context helper
The system SHALL provide a `LoggerFromContext` function that extracts the `*slog.Logger` from a request context. The function SHALL return a no-op discard logger if no logger is found, to prevent nil pointer panics.

#### Scenario: Logger found in context
- **WHEN** `LoggerFromContext(ctx)` is called with a context containing a logger
- **THEN** the stored logger is returned

#### Scenario: No logger in context
- **WHEN** `LoggerFromContext(ctx)` is called with a context lacking a logger
- **THEN** a discard logger (outputs nothing) is returned

### Requirement: Global logger for startup and fatal errors
The system SHALL create a global logger at startup before the HTTP server is configured. This logger SHALL be used for startup messages, configuration errors, and fatal errors where no request context is available. The CSV migrator SHALL use text format regardless of `LOG_FORMAT`.

#### Scenario: Server startup message
- **WHEN** the server starts and binds to the address
- **THEN** a structured `info`-level log is emitted with `addr` and `event: "server_started"` attributes

#### Scenario: Fatal database error
- **WHEN** the database cannot be opened at startup
- **THEN** a structured `error`-level log is emitted before `os.Exit(1)`

### Requirement: Configuration integration
The system SHALL add `LogLevel` and `LogFormat` fields to the existing `Config` struct in `internal/config/config.go`. These fields SHALL be populated from `LOG_LEVEL` and `LOG_FORMAT` environment variables with the documented defaults.

#### Scenario: Config struct populated
- **WHEN** `config.Load()` is called with `LOG_LEVEL=debug` and `LOG_FORMAT=json`
- **THEN** `cfg.LogLevel` is `debug` and `cfg.LogFormat` is `json`

### Requirement: Request ID generation
The logger middleware SHALL generate unique request IDs using a format compatible with common tracing systems. The ID SHALL be short enough for readability but unique enough to avoid collisions.

#### Scenario: Request ID format
- **WHEN** a new request ID is generated
- **THEN** it is a 12-character hex string (e.g., `a1b2c3d4e5f6`)
