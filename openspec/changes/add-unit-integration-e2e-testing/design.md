# Design: Unit, Integration, and End-to-End Testing

## Context

The invoice-generator is a single-binary Go web app (Go 1.26) using SQLite (`modernc.org/sqlite`), `net/http`, `html/template`, and HTMX. There are **zero tests** currently — `go fmt ./...` is the only quality gate. The codebase has ~700+ lines of handler logic, a multi-table database layer (entries, rates, invoices, settings, OCR sessions), PDF generation via Typst, and optional OCR/email features.

Key constraints:
- No CGO — SQLite must stay as `modernc.org/sqlite` (pure Go)
- `typst` is an external binary dependency for PDFs
- The handler uses a concrete `*db.Store` type directly (no interface)
- Tests need to work in CI without requiring typst or SMTP

## Goals / Non-Goals

**Goals:**
- Layered test suite: unit → integration → e2e, each building on the previous
- Fast local feedback (<30s for full suite on developer machine)
- ≥60% overall coverage, ≥80% for internal packages (handler, db, config)
- Test infrastructure that scales as new features are added
- Zero breaking changes to production code

**Non-Goals:**
- 100% coverage (some template logic and edge-case error paths may be excluded)
- Testing the OCR service or SMTP email delivery (external dependencies)
- Performance/load testing
- Browser-based E2E (Cypress/Playwright) — we use `net/http` + HTML assertions instead

## Decisions

### 1. Use `testify` for assertions and mock support

**Decision:** Add `github.com/stretchr/testify` as a dev dependency for `assert` and `require` packages, plus `mock` for handler tests.

**Rationale:** `testify` is the Go ecosystem standard — well-maintained, minimal API surface, and the `mock` generator saves boilerplate. Alternatives like `go-cmp` or custom assertions add no meaningful benefit here.

### 2. Abstract the store behind an interface for handler tests

**Decision:** Create a `db.Storer` interface that defines all methods on `*db.Store`, then have `*db.Store` implement it. Handler tests use `testify/mock` or hand-written stubs; production code passes `*db.Store` (which satisfies the interface).

**Rationale:** Without an interface, handler tests require a real SQLite database for every test, which slows things down and couples HTTP logic to DB details. An interface lets us:
- Unit-test handlers in isolation with deterministic mocks
- Still run integration tests against real SQLite for the full stack path

**Alternative considered:** Test-only factory functions returning `*db.Store`. Simpler but doesn't allow true handler unit tests. We go with the interface because the handler has many branches (HTMX vs non-HTMX, error paths) worth testing in isolation.

### 3. Test database uses temp directories with WAL mode

**Decision:** Each integration/e2e test creates a temporary directory, opens a new SQLite connection with `PRAGMA journal_mode = WAL` and `PRAGMA foreign_keys = ON`, runs migrations from the same migration files, and cleans up in `t.Cleanup()`.

**Rationale:** Using real SQLite (not an in-memory database) ensures tests exercise the actual storage engine. Temp dirs avoid collisions between parallel tests (`t.Parallel()`). This mirrors production behavior exactly.

**Alternative considered:** In-memory SQLite (`file::memory:`). Faster but differs from production WAL mode and can have subtle differences in behavior. We prefer fidelity over marginal speed gains.

### 4. E2E tests use `net/http/httptest` with full app initialization

**Decision:** E2E tests start a real HTTP server (via `httptest.NewServer`), configure it with a temp database, and make real HTTP requests with `http.Client`. They verify status codes, response bodies (HTML content), and redirects.

**Rationale:** This exercises the full stack — routing, middleware (recovery, logging), template rendering, DB interaction — without requiring a running server or browser. It's faster than browser-based E2E and sufficient for a Go web app with HTMX.

### 5. Test data generation with `gofakeit`

**Decision:** Use `github.com/brianvoe/gofakeit/v6` for generating realistic test data (dates, category names, notes). Keep it minimal — just strings and dates.

**Rationale:** Hand-rolling test data is error-prone and verbose. `gofakeit` is lightweight and doesn't add external API surface. We use it only for non-critical fields; critical values (IDs, amounts) are set explicitly.

### 6. Makefile targets: `test`, `test-cover`, `test-race`

**Decision:** Add standard targets:
- `make test` — `go test -race -count=1 ./...`
- `make test-cover` — same with `-coverprofile` and HTML report
- `make test-fast` — skip e2e tests (tagged with `// +build e2e`)

**Rationale:** Developers need quick feedback during iteration. Tagging e2e tests separately lets them run unit+integration in <10s while keeping full coverage available.

## Risks / Trade-offs

| Risk | Mitigation |
|------|-----------|
| Adding an interface changes production code slightly | Interface is additive; `*db.Store` already satisfies it. No callers need to change. |
| Test suite becomes slow as it grows | Use `t.Parallel()`, tag e2e tests, use temp DB cleanup efficiently. Target <30s total. |
| Tests drift from production behavior | Integration tests use real SQLite + real migrations. E2E tests hit the full HTTP stack. |
| Mock store in handler tests misses DB-level bugs | Separate integration tests cover the real DB path. Mocks are only for handler logic isolation. |
| `gofakeit` adds a new dependency | It's lightweight (~50KB compiled), dev-only, and widely used. Trade-off is worth reduced boilerplate. |

## Migration Plan

1. **Phase 1:** Add test infrastructure (testutil package, Makefile targets, dependencies)
2. **Phase 2:** Write unit tests for pure functions (config, helpers)
3. **Phase 3:** Create the `Storer` interface and update handler to use it
4. **Phase 4:** Write integration tests for DB layer
5. **Phase 5:** Write integration tests for HTTP handlers
6. **Phase 6:** Write e2e tests for key workflows
7. **Phase 7:** Verify coverage targets, add CI configuration

Each phase is independently verifiable — the suite grows without breaking.

## Open Questions

1. **Should we test template rendering?** Template parsing errors are caught at startup, and HTML content assertions can be brittle. We'll skip dedicated template tests but cover template-dependent paths via handler integration tests.
2. **CI configuration scope:** Should CI config (GitHub Actions) be part of this change or a follow-up? Recommendation: include a basic `.github/workflows/test.yml` in this change to ensure the suite runs on every PR.
3. **PDF generation testing:** Since `typst` is external, should we test PDF generation at all? Recommendation: skip PDF content tests (they depend on an external binary). Instead, test that the Typst template file exists and data mapping produces valid output structure.
