# Add Unit, Integration, and End-to-End Testing

## Why

The project currently has zero tests — `go fmt ./...` is the only quality tool. This means any refactor or bug fix carries unbounded regression risk. As the application grows (OCR import, email sending, PDF generation), ensuring correctness becomes critical. A layered test suite gives developers confidence to change code safely and serves as living documentation of expected behavior.

## What Changes

- **Unit tests** for pure/helper functions: `parsePositiveFloat`, `parsePositiveInt`, `validateDate`, `validateMonth`, `normalizeCategory`, `formatWithCommas`, `dateLabel`, `money`, `numfmt`, `urlQueryEscape`
- **Unit tests** for config loading with env var overrides
- **Unit tests** for PDF generation logic (Typst template rendering, data mapping)
- **Integration tests** for the database layer: CRUD on entries, rates, invoices, settings; migration execution; SQLite pragmas
- **Integration tests** for handler HTTP endpoints using `net/http/httptest`: create/update/delete entries, rates, invoices; HTMX vs non-HTMX response paths; error handling (400/500)
- **End-to-end tests** for full request/response cycles: multi-step invoice generation flow (create entries → set rates → generate invoice → preview PDF), covering the happy path and key edge cases
- **Test infrastructure**: test database factory (in-memory/temp-dir SQLite), mockable store interface or test helpers, Makefile targets (`make test`, `make test-cover`), CI-friendly setup
- **Coverage target**: ≥60% overall, ≥80% for internal packages (handler, db, config)

## Capabilities

### New Capabilities

- `test-infrastructure`: Shared test helpers, database factories, mock store interface, and Makefile targets for running the full suite
- `unit-tests`: Unit tests for pure functions in handler, config, and PDF packages
- `integration-tests`: Integration tests for DB layer CRUD operations and HTTP handler endpoints
- `e2e-tests`: End-to-end tests covering multi-step user workflows (entry creation → rate assignment → invoice generation)

### Modified Capabilities

<!-- None — no existing spec-level behavior changes -->

## Impact

- **Code**: New `_test.go` files across `internal/handler/`, `internal/db/`, `internal/config/`, `internal/pdf/`. Possibly a new `internal/testutil/` package for shared helpers.
- **Dependencies**: May add `github.com/stretchr/testify` (assertions) and/or `github.com/brianvoe/gofakeit/v6` (test data generation). Both are dev-only dependencies.
- **Build**: `go test ./...` becomes the standard verification command. CI pipeline should run tests on every PR.
- **Existing code**: No breaking changes. The store interface may be abstracted slightly to allow mocking in handler tests, but this is backward-compatible.
