## Context

The invoice generator is a single-binary Go web app using SQLite, `net/http`, `html/template`, and HTMX. A comprehensive code review identified multiple critical and high-severity issues spanning security vulnerabilities, data integrity bugs, and performance problems. The application is not currently safe for production use without these fixes.

Key constraints:
- No CGO (pure Go SQLite via `modernc.org/sqlite`)
- Templates embedded via `//go:embed`
- No external dependencies beyond what's already in `go.mod` unless justified
- SQLite single-writer model means transactions serialize writes naturally

## Goals / Non-Goals

**Goals:**
- Eliminate all critical and high-severity security vulnerabilities
- Fix data integrity bugs (migration execution, invoice race conditions)
- Improve request latency by caching compiled templates
- Ensure graceful shutdown of background OCR processing
- Harden input handling and error responses across all handlers

**Non-Goals:**
- CSRF protection (deferred until session/cookie auth is introduced)
- Pagination for entries/rates lists (deferred — not yet a bottleneck)
- PDF caching (deferred — PDF generation is infrequent)
- Image resize performance optimization (deferred — OCR is optional and infrequent)
- Adding a test suite (tracked in separate change `add-unit-integration-e2e-testing`)

## Decisions

### 1. Typst escaping: comprehensive character escaping

**Decision:** Replace the two-character `escape()` function with one that escapes all Typst special characters: `\`, `"`, `#`, `@`.

**Rationale:** Typst uses `#` for expressions and `@` for labels. A business name like `Test #show: link("https://evil.com")` could inject arbitrary Typst code. The `read` and `write` functions in Typst could potentially access the filesystem.

**Alternatives considered:**
- *Use a Typst template engine*: No mature Go library exists. Overkill for this use case.
- *Sandbox Typst execution*: Typst has no sandbox mode. The `--root` flag limits file access but doesn't prevent code execution within the template.

### 2. Migration runner: split statements + version tracking

**Decision:** Split migration files on `;` boundaries and execute each statement individually. Add a `schema_migrations` table to track which migrations have run.

**Rationale:** `database/sql`'s `Exec()` with the SQLite driver only executes the first statement. Splitting on `;` is simple and works for the current migration files. The `schema_migrations` table prevents re-running migrations and enables future rollback support.

**Alternatives considered:**
- *Use a migration library (e.g., `golang-migrate`)*: Adds a dependency for a simple use case. The current migration files are small and unlikely to grow complex.
- *Use `db.Exec` with multiple statements in a single call*: The SQLite driver doesn't support this reliably.

### 3. Invoice generation: single transaction with deferred FK check

**Decision:** Wrap the entire `GenerateInvoice` flow (unrated check, line query, insert invoice, insert lines) in a single transaction. Use `PRAGMA defer_foreign_keys = ON` within the transaction to avoid FK constraint issues during intermediate steps.

**Rationale:** The current code checks for unrated entries and queries lines outside the transaction, then inserts inside. This creates a TOCTOU window where entries can be modified between check and commit. Moving everything into one transaction eliminates this. SQLite's write lock serializes concurrent invoice creation naturally.

**Alternatives considered:**
- *Application-level locking*: Overkill. SQLite's write lock already serializes.
- *Optimistic concurrency with version column*: Unnecessary complexity for a single-user app.

### 4. Template caching: parse at startup, store on App

**Decision:** Parse all templates in `New()` and store `*template.Template` on the `App` struct. `renderPage` and `renderPartial` use the cached templates. A shared `cloneForPage` method clones the base template and adds page-specific files.

**Rationale:** Go's `html/template` supports cloning via `template.Clone()`, which is safe for concurrent use. Parsing once at startup eliminates 50-200ms of latency per request.

**Alternatives considered:**
- *Parse on first request (lazy caching)*: Adds complexity for minimal benefit — startup parsing is fast enough.
- *Use a third-party template library*: Unnecessary. The standard library is sufficient.

### 5. SMTP TLS: explicit TLS config

**Decision:** Add `d.TLSConfig = &tls.Config{ServerName: cfg.SMTPHost}` to the SMTP dialer.

**Rationale:** Without explicit TLS config, `go-mail` may fall back to plaintext if the server doesn't advertise STARTTLS. Explicit config ensures credentials are always encrypted.

### 6. OCR cleanup: defer-based cleanup + WaitGroup

**Decision:** On upload failure, delete the created session and any uploaded files. Track background goroutines with a `sync.WaitGroup` on the `App` struct, and call `Wait()` during graceful shutdown.

**Rationale:** Orphaned sessions and files accumulate on failure. A WaitGroup is the idiomatic Go pattern for goroutine lifecycle management.

### 7. Handler hardening: standard library replacements

**Decision:** Replace `urlQueryEscape` with `net/url.QueryEscape`. Replace `http.Error(w, err.Error(), ...)` with generic messages. Sanitize `Content-Disposition` values by stripping control characters.

**Rationale:** The custom `urlQueryEscape` only handles 7 characters and is error-prone. Generic error messages prevent information leakage. `Content-Disposition` must not contain CR/LF per RFC 6266.

## Risks / Trade-offs

| Risk | Mitigation |
|---|---|
| Splitting migrations on `;` could break if a statement contains `;` inside a string literal | Current migrations don't have this issue. If needed in future, use a proper SQL parser. |
| Template caching means template changes require a server restart | Acceptable for a self-hosted app. Could add a dev-mode reload later. |
| Single transaction for invoice generation holds a write lock longer | Acceptable — invoice generation is infrequent and fast. |
| `sync.WaitGroup` on App struct requires modifying shutdown logic in `main.go` | Minimal change — add a `Wait()` call after `server.Shutdown()`. |

## Migration Plan

1. New migration `004_migration_tracking.sql` creates `schema_migrations` table
2. The updated migration runner checks this table before running each migration
3. Existing migrations (001-003) are idempotent (`IF NOT EXISTS`), so re-running is safe
4. No data migration needed — all changes are additive or bug fixes
5. Rollback: revert to previous binary; new migration is harmless if left in place

## Open Questions

- Should the `escape()` function also handle newlines in Typst strings? Currently Typst strings can span multiple lines, but embedded newlines in business names could break formatting. **Decision:** Escape newlines as `\n` for safety.
- Should `ConfirmDraftEntries` return a list of skipped entries to the handler for user feedback? **Decision:** Yes — return `([]int64 confirmed, []int64 skipped, error)` so the handler can show a notice.
