## 1. Config

- [x] 1.1 Add `LogLevel` and `LogFormat` fields to `Config` struct in `internal/config/config.go`
- [x] 1.2 Read `LOG_LEVEL` env var (default `info`, valid: debug/info/warn/error) in `Load()`
- [x] 1.3 Read `LOG_FORMAT` env var (default `json`, valid: json/text) in `Load()`

## 2. Logger package

- [x] 2.1 Create `internal/handler/logger.go` with `NewLogger(level, format string) *slog.Logger` constructor
- [x] 2.2 Add `LoggerMiddleware(logger *slog.Logger) func(http.Handler) http.Handler` that injects logger with `request_id` into context
- [x] 2.3 Add `LoggerFromContext(ctx context.Context) *slog.Logger` helper (returns discard logger if none found)
- [x] 2.4 Add `generateRequestID() string` (12-char hex using `crypto/rand`)

## 3. Request logging middleware

- [x] 3.1 Add `responseWriter` wrapper struct implementing `http.ResponseWriter`, `http.Flusher` to capture status code and bytes written
- [x] 3.2 Add `RequestLoggingMiddleware() func(http.Handler) http.Handler` that logs method, path, status, duration, remote addr, user agent, request_id after response
- [x] 3.3 Include HTMX fields (`hx_request`, `hx_target`) when `HX-Request` header is present

## 4. Server entrypoint (cmd/server/main.go)

- [x] 4.1 Create global logger from config at startup
- [x] 4.2 Replace `fmt.Fprintf(os.Stdout, "invoice-app listening on %s\n", ...)` with `slog.Info("server_started", "addr", ...)`
- [x] 4.3 Replace `log.Fatalf` calls with `slog.Error` + `os.Exit(1)` for database and migration errors
- [x] 4.4 Replace `log.Fatalf("serve: %v", err)` with `slog.Error` + `os.Exit(1)`
- [x] 4.5 Update `App` struct to accept logger and pass it through middleware chain
- [x] 4.6 Replace existing `loggingMiddleware` in chain with `LoggerMiddleware` + `RequestLoggingMiddleware`

## 5. CSV migrator (cmd/migrate-csv/main.go)

- [x] 5.1 Create a text-format logger (always text for CLI)
- [x] 5.2 Replace `log.Fatal`/`log.Fatalf` calls with `slog.Error` + `os.Exit(1)`
- [x] 5.3 Replace `log.Printf` completion message with `slog.Info`

## 6. Handler error logging: invoices.go

- [x] 6.1 Log database errors in `handleGenerateInvoice` and `handleGenerateInvoiceAction`
- [x] 6.2 Log PDF generation errors
- [x] 6.3 Log email send errors
- [x] 6.4 Log successful invoice generation at info level
- [x] 6.5 Log missing rate errors (CountUnratedEntries > 0)
- [x] 6.6 Log invoice deletion errors in `handleDeleteInvoice` (no such handler exists — skipped)

## 7. Handler error logging: entries.go

- [x] 7.1 Log entry insert errors
- [x] 7.2 Log entry update errors (rate updates)
- [x] 7.3 Log entry deletion errors
- [x] 7.4 Log validation errors (missing fields, invalid dates) at warn level
- [x] 7.5 Log successful entry creation at info level

## 8. Handler error logging: rates.go

- [x] 8.1 Log rate insert errors
- [x] 8.2 Log rate update errors
- [x] 8.3 Log rate deletion errors
- [x] 8.4 Log validation errors at warn level

## 9. Handler error logging: settings.go and dashboard.go

- [x] 9.1 Log settings save errors
- [x] 9.2 Log settings load errors
- [x] 9.3 Log dashboard data fetch errors

## 10. Documentation
- [x] 10.1 Update `AGENTS.md` environment variables table with `LOG_LEVEL` and `LOG_FORMAT`
- [x] 10.2 Verify `go fmt ./...` passes and `go build ./cmd/server` succeeds
