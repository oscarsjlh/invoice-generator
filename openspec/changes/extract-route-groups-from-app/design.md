## Context

The `App` struct (`internal/handler/app.go`) is the central orchestrator: 9 fields (multiStore, stores, authDB, webAuthn, sessions, cfg, logger, renderer, ocrJobs, authEnabled, authLimiter) and 30+ handler methods. Every route handler is a method on `App`, meaning every handler has access to all fields. The current architecture evolved organically as features were added — auth, entries, rates, invoices, settings, OCR — each landing as methods on the same struct.

The `RequestStoreProvider` seam (ADR-0005) already decouples data access from auth mode. The middleware chain (recover → SecurityHeaders → Logger → RequestLogging → authMiddleware → CSRF → mux) is well-architected and stays as-is.

Gortex community skills already identify natural groupings: `gortex-handler-auth`, `gortex-handler-entries`, etc. This refactor aligns the code structure with those community boundaries.

## Goals / Non-Goals

**Goals:**
- Each handler group has an explicit constructor declaring only the dependencies it uses
- `App` shrinks to a wiring coordinator that creates groups and returns `http.Handler`
- Group constructors replace `newTestApp` / `newTestAppWithAuth` — tests construct only the group they test
- Existing route paths, middleware chain, template rendering, and context-based store/user injection are preserved
- No behavior change — every existing test continues to pass with updated test helpers

**Non-Goals:**
- Changing route paths, HTTP methods, or handler behavior
- Changing the middleware chain order or composition
- Changing the `StoreProvider` interface or per-user store resolution
- Extracting auth middleware into the route groups (stays on `App`)
- Creating new Go packages — groups stay in `internal/handler/`
- Adding interfaces where only one adapter exists (hypothetical seams are not real seams)

## Decisions

**Decision 1: Route groups stay in `internal/handler/` package, not sub-packages**

Rationale: The groups share types (`DashboardPageData`, context keys, `Renderer`, etc.) and the handler package is already the home of HTTP concerns. Moving to sub-packages would create import cycles or require extracting shared types — premature abstraction. The `internal/handler/` package gains more files but each file is smaller and self-describing.

Alternative considered: `internal/handler/auth/`, `internal/handler/entries/`, etc. Rejected because shared page data types and `Renderer` would need a separate shared package, adding indirection without real benefit at this scale.

**Decision 2: Each group struct has a `Routes(mux *http.ServeMux)` method**

Each group registers its own routes on the provided mux. This avoids each group needing to return a sub-handler and keeps the Go 1.22+ method-based routing pattern (`"GET /path"`, `"POST /path/{id}"`) unchanged.

Alternative considered: Each group returns its own `http.Handler` (sub-mux). Rejected because it complicates middleware application — we'd need to decide whether middleware wraps the sub-mux or the outer mux, and the current middleware order applies uniformly which is simpler.

**Decision 3: `authMiddleware` stays on `App`, not in any group**

Rationale: The auth middleware needs access to `authEnabled`, `RegistrationEnabled`, `sessions`, `stores`, and public path logic — a cross-cutting concern. It is the wiring between auth config and per-request behavior. Moving it into `AuthHandlers` would be misleading because it gates all routes, not just auth routes.

**Decision 4: Group constructors accept dependencies, not config**

Each constructor takes exactly the types it needs (e.g., `*Renderer`, `*auth.WebAuthnManager`), not `config.Config`. The caller (`App.New` or `main.go`) reads from config and passes the resolved dependencies. This makes dependencies visible in the constructor signature and testable with fakes.

**Decision 5: Extract in dependency order (auth first, then core domain)**

Auth handlers are the most complex group (6 routes, 4 dependencies). Extracting them first validates the pattern, then entries/rates/invoices/settings follow the same template. OCR is last due to its `OCRJobRunner` dependency.

## Risks / Trade-offs

**[Risk] Test helper churn** — ~10 test files reference `newTestApp` or `newTestAppWithAuth`. Each test file needs its helpers updated to use group constructors.
→ **Mitigation**: Update test helpers in parallel with extraction. Each extraction batch includes both the group file and its test file updates. Run `go test ./internal/handler/...` after each batch.

**[Risk] Merge conflicts with in-flight feature work** — the `App` struct and its methods are touched by most feature branches.
→ **Mitigation**: This is a pure structural refactor (no behavior change). Merge conflicts are mechanical: move method from `App` to `GroupStruct`. Communicate the change to the team before starting.

**[Trade-off] More files, fewer lines per file** — the handler package goes from ~15 files to ~25 files.
→ Acceptable: each file is self-describing. A newcomer reading `auth_handlers_group.go` sees exactly what auth routes exist and what they depend on, without scanning a 437-line App.
