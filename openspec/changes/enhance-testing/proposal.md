## Why

After the initial testing improvement, auth handlers (login/register/logout), auth middleware, OCR handlers, and the full middleware chain remain untested. Existing tests inline 14-field Settings structs in 4+ places. DB tests use flat patterns instead of table-driven subtests. No tests exercise the actual HTTP router with middleware composition. This change closes these gaps.

## What Changes

- Add HTTP handler tests for all auth endpoints (login/register/logout flows)
- Add auth middleware tests (session validation, context injection, legacy mode)
- Consolidate inline Settings structs to use `testutil.SampleSettings()` factory
- Add full middleware chain integration test via `httptest.Server` on `Routes()`
- Add OCR handler tests with a testable client interface
- Add concurrent MultiStore thread-safety test
- Convert DB CRUD tests to table-driven with subtests

## Capabilities

### New Capabilities
- `auth-handler-tests`: HTTP handler tests for login, register, logout flows
- `auth-middleware-tests`: Tests for authMiddleware session validation and context injection
- `fixtures-consolidation`: Replace inline Settings structs with SampleSettings() across all tests
- `integration-tests`: Full middleware chain test via httptest.Server using Routes()
- `ocr-handler-tests`: Handler tests for upload, start session, status, confirm, delete
- `concurrent-tests`: Thread-safety test for MultiStore.ForUser under parallel access
- `table-driven-tests`: Convert DB entry/rate/invoice tests to table-driven with t.Run

## Impact

- Affected: `internal/handler/` (auth_handlers_test.go, middleware_test.go, invoices_test.go, etc.), `internal/db/` (entries_test.go, rates_test.go, invoices_test.go, settings_test.go, multistore_test.go), `internal/ocr/`
- New file: `internal/handler/integration_test.go`
- No breaking changes
