## 1. Migration Runner Fix

- [x] 1.1 Create migration `004_migration_tracking.sql` with `schema_migrations` table (filename TEXT PRIMARY KEY, applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)
- [x] 1.2 Rewrite `Store.Migrate()` in `internal/db/db.go` to split migration files on `;` boundaries, execute each statement individually, and record successful migrations in `schema_migrations`
- [x] 1.3 Verify all existing migrations (001-003) execute correctly with the new runner

## 2. Template Caching

- [x] 2.1 Add `baseTmpl *template.Template` field to the `App` struct
- [x] 2.2 Create `compileTemplates()` method that parses the base layout template at startup using `template.ParseFS`
- [x] 2.3 Refactor `renderPage()` to use cached base template via `template.Clone()` and parse page-specific files
- [x] 2.4 Refactor `renderPartial()` to use cached base template via `template.Clone()` and parse partial files
- [x] 2.5 Remove template parsing from the request path in both methods (base template is cached, only page-specific files parsed per request)
- [x] 2.6 Verify `go fmt ./...` passes and server starts correctly

## 3. Typst Template Security

- [x] 3.1 Rewrite `escape()` in `internal/pdf/generate.go` to escape `\`, `"`, `#`, `@`, and newline characters
- [x] 3.2 Add test cases for `escape()` covering all special characters
- [x] 3.3 Verify PDF generation works with business names containing `#`, `@`, and quotes

## 4. Invoice Atomicity

- [x] 4.1 Refactor `GenerateInvoice()` in `internal/db/invoices.go` to wrap the entire flow (unrated check, line query, insert invoice, insert lines) in a single transaction
- [x] 4.2 Use the transaction for `CountUnratedEntries` and `queryInvoiceLines` calls instead of `s.db`
- [x] 4.3 Move `nextInvoiceNumber()` to use the passed transaction (already does — verified)
- [x] 4.4 Verify concurrent invoice generation produces unique invoice numbers (SQLite write lock serializes naturally)
- [x] 4.5 Run `go fmt ./...` and verify no regressions

## 5. SMTP TLS Enforcement

- [x] 5.1 Add `TLSConfig: &tls.Config{ServerName: cfg.SMTPHost}` to the SMTP dialer in `internal/email/send.go`
- [x] 5.2 Add `"crypto/tls"` import to `send.go`
- [x] 5.3 Verify email sending still works with the TLS config (build compiles)

## 6. OCR Resilience

- [x] 6.1 Add `ocrWg sync.WaitGroup` field to the `App` struct
- [x] 6.2 Add `ocrWg.Add(1)` before launching `go a.processOCRSession()` and `defer a.ocrWg.Done()` at the start of `processOCRSession()`
- [x] 6.3 Add `WaitForOCR()` method on `App` that calls `a.ocrWg.Wait()`
- [x] 6.4 Call `app.WaitForOCR()` in `main.go` after `server.Shutdown()` (added graceful shutdown with signal handling)
- [x] 6.5 Add cleanup logic in `ocrStartSession()` to delete the session and remove uploaded files if the loop fails mid-way
- [x] 6.6 Update `ConfirmDraftEntries()` to return `([]int64 confirmed, []int64 skipped, error)` instead of just `error`
- [x] 6.7 Update `ocrConfirmDrafts()` handler to display a notice showing confirmed and skipped counts

## 7. Handler Hardening

- [x] 7.1 Replace all calls to `urlQueryEscape()` with `net/url.QueryEscape()` in `internal/handler/app.go`
- [x] 7.2 Remove the `urlQueryEscape` function from `app.go`
- [x] 7.3 Sanitize `Content-Disposition` header values by stripping `\r`, `\n`, and `\x00` characters
- [x] 7.4 Replace `http.Error(w, err.Error(), ...)` with `http.Error(w, "Internal server error", ...)` in all handlers, logging the detailed error server-side
- [x] 7.5 Remove the `notFoundIfNoRows` function from `app.go`
- [x] 7.6 Handle the ignored error in `handler/invoices.go:228` (`settings, _ := a.store.LoadSettings()`) — log the error and proceed with empty settings
- [x] 7.7 Rename `generateInvoicePDF` method to `buildPDF` to avoid confusion with the `generateInvoice` handler
- [x] 7.8 Run `go fmt ./...` and verify all handlers compile

## 8. Verification

- [x] 8.1 Run `go build` and verify binary builds successfully
- [x] 8.2 Run `go test ./...` and verify all tests pass
- [x] 8.3 Run `go fmt ./...` and verify no formatting issues
- [x] 8.4 Manually test: start server, verify graceful shutdown with signal handling
