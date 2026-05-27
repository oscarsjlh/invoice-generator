## ADDED Requirements

### Requirement: Handler groups are self-describing modules

Each route handler group SHALL be a struct with an explicit constructor that declares only the dependencies the group actually uses. A caller SHALL be able to construct and test a group in isolation by providing only those dependencies.

#### Scenario: Auth group has explicit dependencies

- **WHEN** constructing `AuthHandlers`
- **THEN** the constructor accepts `*auth.WebAuthnManager`, `*auth.SessionManager` (or `*auth.SessionCookie`), and `*Renderer`
- **AND** the group does not have access to `MultiStore`, `Config`, `OCRJobRunner`, or other fields not relevant to auth

#### Scenario: Entry group has minimal dependencies

- **WHEN** constructing `EntryHandlers`
- **THEN** the constructor accepts only `*Renderer`
- **AND** the store and user are accessed exclusively through request context (`StoreFromContext`, `UserFromContext`)

### Requirement: Each handler group registers its own routes

Each handler group SHALL provide a `Routes(mux *http.ServeMux)` method that registers all routes belonging to that group on the provided mux using Go 1.22+ method-based patterns.

#### Scenario: Auth group registers all auth routes

- **WHEN** `AuthHandlers.Routes(mux)` is called and auth is enabled
- **THEN** `GET /login`, `POST /login/begin`, `POST /login/finish`, `POST /logout` are registered on the mux
- **AND** if registration is enabled, `GET /register`, `POST /register/begin`, `POST /register/finish` are also registered

#### Scenario: Entry group registers all entry routes

- **WHEN** `EntryHandlers.Routes(mux)` is called
- **THEN** `GET /entries`, `GET /entries/table`, `POST /entries`, `GET /entries/{id}/edit`, `POST /entries/{id}`, `POST /entries/{id}/delete` are registered

### Requirement: App is a wiring coordinator

The `App` struct SHALL be reduced to a wiring coordinator that creates handler groups, calls each group's `Routes` method, applies middleware, and returns the top-level `http.Handler`. `App` SHALL NOT hold handler method implementations.

#### Scenario: App.Routes wires groups together

- **WHEN** `App.Routes()` is called
- **THEN** a new `http.ServeMux` is created
- **AND** each handler group's `Routes(mux)` is called
- **AND** middleware is applied in the existing order (recover → SecurityHeaders → Logger → RequestLogging → authMiddleware → CSRF → mux)
- **AND** `otelhttp.NewHandler` wraps the result

### Requirement: Handler group extraction preserves existing behavior

After extraction, every existing route SHALL respond identically to the same requests. No route path, HTTP method, handler behavior, template rendering, or context-based store/user injection SHALL change.

#### Scenario: Dashboard renders identically after extraction

- **WHEN** an authenticated user requests `GET /` after extraction
- **THEN** the dashboard page renders with the same MonthlySummary, RecentInvoices, years, months, totals, and user data as before extraction

#### Scenario: Entry CRUD works identically after extraction

- **WHEN** an authenticated user creates, reads, updates, and deletes entries after extraction
- **THEN** all operations behave identically to before extraction (same responses, same database state)
