## 1. Setup test infrastructure

- [x] 1.1 Add testify and gofakeit dependencies to go.mod (`go get github.com/stretchr/testify github.com/brianvoe/gofakeit/v6`)
- [x] 1.2 Create `invoice-service/internal/testutil/` package with shared helpers:
  - [ ] 1.2.1 `NewTestDB(t) (*db.Store, func())` — creates temp-dir SQLite, applies migrations, returns cleanup function
  - [ ] 1.2.2 `SampleEntry()` — returns a valid db.Entry for tests
  - [ ] 1.2.3 `SampleRate()` — returns a valid db.Rate for tests
  - [ ] 1.2.4 `SampleInvoice()` — returns a valid db.Invoice with lines for tests
  - [ ] 1.2.5 `AssertHTMLContains(t, body []byte, substr string)` — assertion helper
- [x] 1.3 Add test targets to `invoice-service/Makefile`:
  - [ ] 1.3.1 `make test` — `go test -race -count=1 ./...`
  - [ ] 1.3.2 `make test-fast` — `go test -race -count=1 -tags='!e2e' ./...`
  - [ ] 1.3.3 `make test-cover` — runs with `-coverprofile=coverage.out` and opens HTML report
- [x] 1.4 Create root-level `.gitignore` entry for `*.db`, `*.out`, `coverage.html`, `data/`

## 2. Implement Storer interface for dependency injection

- [x] 2.1 Create `invoice-service/internal/db/storer.go` with the `Storer` interface containing all public methods on `*db.Store`
- [x] 2.2 Update `handler/app.go` to use `Storer` instead of `*db.Store` in the `App` struct
- [x] 2.3 Verify `go build ./...` still passes (Store satisfies Storer implicitly)
- [x] 2.4 Run `go fmt ./...`

## 3. Write unit tests for handler helper functions

- [x] 3.1 Create `invoice-service/internal/handler/helpers_test.go`:
  - [ ] 3.1.1 Test `parsePositiveFloat` — valid, zero, negative, non-numeric, whitespace-trimmed inputs
  - [ ] 3.1.2 Test `parsePositiveInt` — valid, zero, non-numeric inputs
  - [ ] 3.1.3 Test `validateDate` — valid YYYY-MM-DD, invalid formats, whitespace trimming
  - [ ] 3.1.4 Test `validateMonth` — valid YYYY-MM, full date rejection
  - [ ] 3.1.5 Test `normalizeCategory` — non-empty trimmed, empty string → "All"
  - [ ] 3.1.6 Test `formatWithCommas` — small numbers, thousands, millions, negatives
  - [ ] 3.1.7 Test `money` — whole numbers, decimals
  - [ ] 3.1.8 Test `numfmt` — simple decimals, large numbers with commas
  - [ ] 3.1.9 Test `dateLabel` — YYYY-MM-DD, RFC3339, unparseable strings
  - [ ] 3.1.10 Test `urlQueryEscape` — spaces, ampersands, percent signs
  - [ ] 3.1.11 Test `isHTMX` — with and without HX-Request header

## 4. Write unit tests for config package

- [x] 4.1 Create `invoice-service/internal/config/config_test.go`:
  - [ ] 4.1.1 Test default values when no env vars are set (Address, LogLevel, LogFormat, SMTPPort)
  - [ ] 4.1.2 Test env var overrides for Address, LogLevel, LogFormat
  - [ ] 4.1.3 Test OCR_ENABLED parsing — "true" → true, empty → false
  - [ ] 4.1.4 Test database path and migrations dir defaults

## 5. Write unit tests for PDF package

- [x] 5.1 Create `invoice-service/internal/pdf/generate_test.go`:
  - [ ] 5.1.1 Test `FormatInvoiceTyp` produces valid Typst template data (check output contains expected fields)
  - [ ] 5.1.2 Test `ParseAddress` — single line, two lines, three lines, empty input

## 6. Write integration tests for database layer

- [x] 6.1 Create `invoice-service/internal/db/entries_test.go`:
  - [ ] 6.1.1 Test CreateEntry + GetEntry round-trip
  - [ ] 6.1.2 Test ListEntries returns all entries sorted by date DESC
  - [ ] 6.1.3 Test UpdateEntry modifies fields in place
  - [ ] 6.1.4 Test DeleteEntry removes record permanently (GetEntry returns error)
- [x] 6.2 Create `invoice-service/internal/db/rates_test.go`:
  - [ ] 6.2.1 Test CreateRate + GetRate round-trip
  - [ ] 6.2.2 Test ListRates returns all rates
  - [ ] 6.2.3 Test UpdateRate modifies fields in place
  - [ ] 6.2.4 Test DeleteRate removes record permanently
- [x] 6.3 Create `invoice-service/internal/db/invoices_test.go`:
  - [ ] 6.3.1 Test GenerateInvoice creates invoice with line items when all entries have rates
  - [ ] 6.3.2 Test GenerateInvoice fails with error when entries lack rates (CountUnratedEntries > 0)
  - [ ] 6.3.3 Test GetInvoice returns invoice with lines populated
  - [ ] 6.3.4 Test ListInvoices returns summaries in correct order
  - [ ] 6.3.5 Test CountUnratedEntries returns 0 when all rated, >0 when unrated exist
  - [ ] 6.3.6 Test nextInvoiceNumber generates unique slugs with suffix fallback
- [x] 6.4 Create `invoice-service/internal/db/settings_test.go`:
  - [ ] 6.4.1 Test SaveSettings + GetSettings round-trip
  - [ ] 6.4.2 Test GetSettings returns zero-value when nothing saved

## 7. Write integration tests for HTTP handlers

- [x] 7.1 Create `invoice-service/internal/handler/entries_test.go`:
  - [ ] 7.1.1 Test POST /entries creates entry and redirects (303) with valid data
  - [ ] 7.1.2 Test POST /entries returns 400 when category is empty
  - [ ] 7.1.3 Test POST /entries returns 400 when hours is invalid
  - [ ] 7.1.4 Test GET /entries/table returns HTML partial with entries list
  - [ ] 7.1.5 Test POST /entries/{id} with HTMX header returns partial HTML (not redirect)
  - [ ] 7.1.6 Test POST /entries/{id}/delete removes entry and redirects
- [x] 7.2 Create `invoice-service/internal/handler/rates_test.go`:
  - [ ] 7.2.1 Test GET /rates returns 200 with rate list HTML
  - [ ] 7.2.2 Test POST /rates creates a rate and redirects
  - [ ] 7.2.3 Test POST /rates/{id}/delete removes rate
- [x] 7.3 Create `invoice-service/internal/handler/invoices_test.go`:
  - [ ] 7.3.1 Test GET /invoices returns 200 with invoice list HTML
  - [ ] 7.3.2 Test POST /invoices/generate creates invoice when rates exist (redirect to preview)
  - [ ] 7.3.3 Test POST /invoices/generate fails when entries lack rates (error notice)
  - [ ] 7.3.4 Test GET /invoices/{id} returns invoice preview page with data
  - [ ] 7.3.5 Test GET /invoices/{id} returns 404 for non-existent invoice
- [x] 7.4 Create `invoice-service/internal/handler/settings_test.go`:
  - [ ] 7.4.1 Test GET /settings returns saved settings in form fields
  - [ ] 7.4.2 Test POST /settings saves settings and redirects back with notice
- [x] 7.5 Create `invoice-service/internal/handler/middleware_test.go`:
  - [ ] 7.5.1 Test recoverMiddleware catches panic and returns 500 (server continues)

## 8. Write end-to-end tests

- [x] 8.1 Create `invoice-service/internal/handler/e2e_test.go` with e2e build tag:
  - [ ] 8.1.1 Test full invoice workflow: create entries → set rates → generate invoice → verify preview page contains all line items
  - [ ] 8.1.2 Test entry CRUD via HTTP: create → edit → delete entries and verify table updates
  - [ ] 8.1.3 Test rate management: create rate → update rate → verify list reflects changes
  - [ ] 8.1.4 Test settings persistence: save settings → reload page → verify form fields contain saved values
  - [ ] 8.1.5 Test dashboard summary shows correct totals for entries with matching rates
  - [ ] 8.1.6 Test invoice PDF endpoint returns application/pdf content type (skip actual PDF content validation since typst is external)

## 9. Verify and polish

- [x] 9.1 Run `make test` and verify all tests pass
- [x] 9.2 Run `make test-cover` and check coverage meets targets (≥60% overall, ≥80% for internal packages)
- [x] 9.3 Run `make test-fast` to confirm fast path works (<10s)
- [x] 9.4 Run `go vet ./...` to catch any static analysis issues in test code
- [x] 9.5 Ensure all tests use `t.Parallel()` where safe (no shared state between tests)
- [x] 9.6 Verify cleanup functions remove temp directories (check disk after full suite run)
