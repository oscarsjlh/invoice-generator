## Context

The application is a single-binary Go web server (Go 1.26) with SQLite, `net/http`, and `html/template` + HTMX. It currently has no structured logging. Two `main` entrypoints exist: `cmd/server/main.go` (web server) and `cmd/migrate-csv/main.go` (CLI CSV importer). Request logging uses `fmt.Printf` with method+path only. No handler errors are logged.

Go 1.21+ includes `log/slog` in the standard library, providing structured, leveled, JSON-capable logging with near-zero overhead and no external dependencies.

## Goals / Non-Goals

**Goals:**
- Replace all `fmt.Printf` and `log.Print*`/`log.Fatal*` with `slog` calls
- Produce JSON-structured log output by default with `LOG_FORMAT=text` fallback
- Make log level configurable via `LOG_LEVEL` env var
- Structured HTTP request logging with method, path, status, duration, remote addr, user agent
- Log all handler-level errors (DB, PDF, email, validation) with relevant fields
- Inject logger into request context via middleware; provide a helper to extract it

**Non-Goals:**
- OpenTelemetry traces or spans (future capability)
- Log aggregation service integration (e.g., Loki, Elasticsearch)
- Log sampling or rate-limiting
- Audit logging for sensitive operations
- Changing how errors are surfaced to the user (HTTP responses stay the same)
- Adding logging to the `internal/db/` and `internal/pdf/` packages directly (errors bubble up to handlers via return values)

## Decisions

### Decision 1: Use `log/slog` from stdlib (not a third-party library)

**Rationale:** The project uses Go 1.26. `log/slog` has been stable since Go 1.21 and provides structured, leveled, JSON logging with zero external dependencies. It integrates directly with `net/http` middleware patterns and supports handler-based customization. Third-party libraries like `zap` or `zerolog` add dependency weight and complexity without meaningful benefit for this project's scale.

**Alternatives considered:**
- `logrus`: Popular but in maintenance mode, no structured fields, heavier API.
- `zap`: Very fast but adds dependency, more complex API, overkill for a single-binary app.
- `zerolog`: Also fast, but another dependency; slog is simpler and built-in.

### Decision 2: JSON output by default, text as opt-in via `LOG_FORMAT`

**Rationale:** JSON is the standard for structured log ingestion in production. Text format is useful for local development readability. The default should favor production readiness.

### Decision 3: Logger injected via request context, not global

**Rationale:** Context-scoped logging allows attaching request-specific fields (request ID, user info) to all log entries within a request's lifetime. A single global logger (for startup/fatal errors) is still used, but handlers get their logger from `r.Context()`.

### Decision 4: New middleware replaces existing `loggingMiddleware` (not wraps it)

**Rationale:** The existing `loggingMiddleware` is too minimal to retrofit. Replacing it with a new `LoggerMiddleware` that both logs requests AND injects the logger into context is cleaner. The middleware captures the response via a `responseWriter` wrapper to read the status code and bytes written.

### Decision 5: Separate `LoggerMiddleware` and `RequestLoggingMiddleware`

**Rationale:** `LoggerMiddleware` injects the logger into context (run first in chain). `RequestLoggingMiddleware` logs the request/response summary (run after). This separation allows other middleware (recovery, auth) to access the logger from context.

### Decision 6: `LOG_LEVEL` env var, no dynamic reload

**Rationale:** Log level is set at startup from `LOG_LEVEL` (default: `info`). Dynamic reload adds complexity disproportionate to value for this application. Valid values: `debug`, `info`, `warn`, `error`.

## Risks / Trade-offs

- **Risk:** JSON logs during local development are harder to read → **Mitigation:** `LOG_FORMAT=text` for `make run` or local dev; default to JSON for production.
- **Risk:** Adding handler error logging may expose sensitive data in logs → **Mitigation:** Log structured fields (error message, entry ID, invoice ID) but never log full request bodies, passwords, or email bodies.
- **Risk:** `slog` in context pattern adds a small allocation per request → **Mitigation:** Negligible for this app's throughput; context value lookup is O(1).

## Open Questions

- Should PDF generation errors log the full Typst stderr output? (Decision deferred: log the error message only; stderr can be added later if debugging needs warrant it.)
- Should the CSV migrator use JSON logs or text? (Use text format — it's a CLI tool, not a server.)
