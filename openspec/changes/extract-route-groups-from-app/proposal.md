## Why

The `App` struct in `internal/handler/app.go` has 9 fields and 30+ handler methods across 437 lines. Every handler is a method on `App`, so every handler implicitly depends on every field. The **interface** (30+ methods) is nearly as complex as the **implementation** (the handlers themselves) — the definition of shallow. Test construction requires wiring the full `App` even for single-handler tests. Extracting route groups into self-describing modules with explicit dependency constructors deepens the handler layer and improves locality.

## What Changes

- Extract `AuthHandlers` struct with its own constructor taking `WebAuthnManager`, `SessionManager`, `Renderer`, and rate limiter — owning the 6 auth routes
- Extract `EntryHandlers` struct with constructor taking `Renderer` only (store from context) — owning the 6 entry routes
- Extract `RateHandlers` struct — owning the 5 rate routes
- Extract `InvoiceHandlers` struct — owning the 5 invoice routes
- Extract `SettingsHandlers` struct — owning the 2 settings routes
- Extract `OCRHandlers` struct — owning the 5 OCR routes
- `App` shrinks to a wiring coordinator: creates groups, registers routes, returns `http.Handler`
- Each group's `Routes()` method registers its own routes on a sub-mux or directly
- Middleware (auth, CSRF, logging, security headers, recovery, tracing) stays on `App` as the outermost orchestration layer
- **No behavior change** — pure structural refactor, existing tests remain passing

## Capabilities

### New Capabilities

- `route-group-extraction`: Each handler group is a self-describing module with an explicit dependency constructor. Callers can test a group in isolation by providing only the dependencies that group actually uses.

### Modified Capabilities

<!-- None — this is a structural refactor with no requirement-level behavior changes. -->

## Impact

- `internal/handler/app.go` — shrinks significantly, becomes wiring-only
- New files: `internal/handler/auth_handlers_group.go`, `internal/handler/entry_handlers_group.go`, etc. (each ~50-80 lines)
- `cmd/server/main.go` — wiring may shift from passing deps to `handler.New` to passing deps to individual group constructors
- All existing test files in `internal/handler/` — test helpers update to use group constructors instead of full `App`
- No change to routes, templates, middleware behavior, or external API surface
