## Why

The application currently has no structured logging. Request logging uses bare `fmt.Printf` with no timestamp, status code, duration, or structured fields. All handler-level errors (database failures, PDF generation errors, email send failures, validation errors) are silently swallowed after `http.Error()` — making production debugging impossible. Replacing ad-hoc `fmt.Printf` and `log.Fatal` calls with Go's standard `log/slog` (JSON output) gives observability parity with any production service.

## What Changes

- Add `log/slog` JSON structured logging throughout the application
- Replace all `fmt.Printf` and `log.Print*` calls with `slog` equivalents
- Upgrade the logging middleware to log structured fields (method, path, status, duration, remote addr, user agent)
- Add error-level logging in all HTTP handlers for database, PDF, email, and validation errors
- Make log level configurable via `LOG_LEVEL` environment variable (debug, info, warn, error)
- Add a `LoggerMiddleware` that injects a `*slog.Logger` into the request context
- Provide a helper to extract the logger from context in handlers
- Support JSON log output by default with an optional `LOG_FORMAT` env var for text fallback

## Capabilities

### New Capabilities
- `structured-logging`: Core slog setup, level configuration, JSON/text format selection, logger injection into request context via middleware, and a helper to retrieve the logger from context.
- `request-logging`: Structured HTTP request logging middleware logging method, path, status code, duration, remote address, user agent, and request ID.
- `handler-error-logging`: Error-level logging in all HTTP handlers for database errors, PDF generation failures, email send failures, and validation errors with relevant structured fields.

### Modified Capabilities
<!-- None — this is a greenfield addition. No existing specs to modify. -->

## Impact

- **Affected code:** `cmd/server/main.go`, `cmd/migrate-csv/main.go`, `internal/config/config.go`, `internal/handler/` (all files), `internal/handler/app.go` (middleware chain), `internal/db/` (optional, for DB error context)
- **New files:** `internal/handler/logger.go` (logger middleware + context helpers)
- **Dependencies:** None — `log/slog` is in Go 1.21+ stdlib (project uses Go 1.26)
- **Breaking changes:** None — all existing HTTP responses and redirects are preserved
- **Environment:** New `LOG_LEVEL` and `LOG_FORMAT` env vars
