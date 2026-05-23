## Context

The previous change added unit tests for auth, AuthDB, MultiStore, middleware, handlers, and email. This change builds on that foundation to test the remaining uncovered handlers (auth, OCR), integration points (middleware chain, concurrent access), and improve test quality (fixtures, table-driven patterns).

## Goals / Non-Goals

**Goals:**
- Test all auth handler endpoints (login/register/logout) through HTTP handlers
- Test authMiddleware (session validation, context injection, legacy mode fallback)
- Consolidate duplicate Settings struct literals to use SampleSettings()
- Test the full HTTP router + middleware chain via httptest.Server
- Test OCR handlers by making OCR client an interface
- Test MultiStore thread safety under concurrent access
- Convert flat DB tests to table-driven with subtests

**Non-Goals:**
- Testing actual WebAuthn browser flow (requires real browser)
- Testing AWS Bedrock integration (requires external service)
- Testing typst binary execution

## Decisions

### Decision 1: Auth handler tests use real handlers with test AuthDB

Test `loginPage`, `registerPage`, `beginLogin`, `finishLogin`, `beginRegistration`, `finishRegistration`, `logout` by constructing a test App with real AuthDB and WebAuthnManager, injecting the store/user into context as needed. For BeginRegistration/FinishRegistration flows, test the error paths (username taken, invalid session) which don't require real WebAuthn browser interactions.

### Decision 2: Integration test uses httptest.Server on Routes() mux

Create a test server with `httptest.NewServer(a.Routes())` and make HTTP requests through the full stack. Use `AUTH_ENABLED=false` for simpler setup. Verify status codes, redirects, and response content through the complete middleware chain.

### Decision 3: OCR tests use a Client interface

Make the OCR `Client` an interface with `Extract` method. Tests pass a mock client. The production code path uses the real AWS Bedrock client unchanged.

### Decision 4: Concurrent test uses goroutines + sync.WaitGroup

Launch 20 goroutines calling `ForUser` on different user IDs simultaneously, verify all succeed and no race conditions with `-race`.

### Decision 5: Table-driven conversion preserves existing test logic

Convert each test function to table-driven by extracting test cases while preserving the exact assertions. No behavior changes.

