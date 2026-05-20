## Why

A code review identified critical security vulnerabilities (Typst template injection, path traversal, CRLF header injection), data integrity bugs (TOCTOU race in invoice generation, broken multi-statement migrations), and performance issues (templates re-parsed on every request). These must be addressed before the application can be considered production-ready.

## What Changes

- **Typst template escaping** — Replace the incomplete `escape()` function to properly escape all Typst special characters (`#`, `@`, `\`, `"`) preventing template injection via user-controlled fields
- **Migration runner** — Fix multi-statement SQL execution (currently only first statement runs) and add a `schema_migrations` table for version tracking
- **Invoice generation** — Move the entire invoice creation flow (unrated check, line query, insert) into a single database transaction to eliminate TOCTOU race conditions
- **Template caching** — Parse and compile Go templates once at startup, cache on the `App` struct, and reuse on each request
- **SMTP TLS enforcement** — Add explicit TLS configuration to the SMTP dialer to prevent credential leakage
- **OCR session resilience** — Add cleanup of orphaned sessions/files on upload failure and use `sync.WaitGroup` for goroutine lifecycle management during shutdown
- **Handler hardening** — Replace custom `urlQueryEscape` with `net/url.QueryEscape`, sanitize `Content-Disposition` header values, replace internal error messages with generic user-facing messages, and remove dead code (`notFoundIfNoRows`)

## Capabilities

### New Capabilities

- `template-security`: Proper escaping of Typst template content to prevent injection attacks via user-controlled fields (business name, customer name, etc.)
- `migration-runner`: Reliable multi-statement SQL migration execution with version tracking via a `schema_migrations` table
- `invoice-atomicity`: Atomic invoice generation within a single database transaction, eliminating race conditions between validation and commit
- `template-caching`: Template compilation at application startup with cached reuse per request, eliminating per-request parsing overhead
- `smtp-tls`: Enforced TLS configuration for SMTP connections to protect credentials in transit
- `ocr-resilience`: Proper cleanup of orphaned OCR sessions and uploaded files on failure, plus graceful goroutine shutdown via `sync.WaitGroup`
- `handler-hardening`: Input sanitization for HTTP headers, safe generic error responses to users, and removal of dead code paths

### Modified Capabilities

<!-- No existing spec requirements are changing -->

## Impact

- `internal/pdf/generate.go` — `escape()` function rewritten; `FormatInvoiceTyp` unchanged interface
- `internal/db/db.go` — Migration runner rewritten; new `schema_migrations` table (migration 004)
- `internal/db/invoices.go` — `GenerateInvoice` refactored to use a single transaction
- `internal/handler/app.go` — Template caching added to `App` struct; `renderPage`/`renderPartial` refactored; `urlQueryEscape` replaced; `notFoundIfNoRows` removed
- `internal/handler/invoices.go` — `Content-Disposition` header sanitized; `generateInvoicePDF` renamed; settings error handled
- `internal/email/send.go` — TLS config added to SMTP dialer
- `internal/handler/ocr.go` — Upload loop cleanup on failure; `sync.WaitGroup` for background processing
- `migrations/004_migration_tracking.sql` — New migration for `schema_migrations` table
- No breaking changes to external APIs or user-facing behavior
