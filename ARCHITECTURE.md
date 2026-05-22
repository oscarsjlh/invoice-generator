# Invoice Generator — Architecture Deep Dive

## Overview

This is a single-binary Go web application for generating invoices from time-tracking entries. It uses **WebAuthn (passkeys)** for authentication, **SQLite** for persistence, and **Typst** for PDF generation. The architecture is designed around **multi-tenancy via per-user SQLite databases**.

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Browser Client                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │  Dashboard   │  │   Entries    │  │   Invoices   │  │   Settings   │     │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘     │
│         │                 │                 │                 │            │
│         └─────────────────┴────────┬──────────┴─────────────────┘            │
│                                    │                                         │
│                    WebAuthn (navigator.credentials)                          │
│                           Passkey Authentication                             │
└────────────────────────────────────┼─────────────────────────────────────────┘
                                     │ HTTP (port 8080)
                                     ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Go HTTP Server                                     │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │                        http.ServeMux (Go 1.22+)                      │  │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐   │  │
│  │  │  GET /   │ │/register │ │  /login  │ │ /static  │ │  /invoices│   │  │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘   │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
│                                     │                                       │
│                     ┌───────────────┴───────────────┐                       │
│                     │      Middleware Stack         │                       │
│                     │  recover -> logging -> auth  │                       │
│                     └───────────────────────────────┘                       │
│                                     │                                       │
│                     ┌───────────────┴───────────────┐                       │
│                     │     AuthMiddleware            │                       │
│                     │  (skips /login, /register,    │                       │
│                     │   /static/*)                  │                       │
│                     └───────────────────────────────┘                       │
└─────────────────────────────────────┼───────────────────────────────────────┘
                                      │
                    ┌─────────────────┼─────────────────┐
                    │                 │                 │
                    ▼                 ▼                 ▼
           ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
           │    AuthDB     │  │   MultiStore  │  │    WebAuthn  │
           │  (SQLite)     │  │              │  │   Manager    │
           │               │  │              │  │              │
           │ users         │  │ ForUser()    │  │ Begin/Finish │
           │ credentials   │  │   -> Open DB  │  │ Registration │
           │ sessions      │  │   -> Migrate  │  │ Begin/Finish │
           └──────────────┘  │   -> Cache     │  │ Login        │
                           └──────────────┘  └──────────────┘
                                  │
                    ┌─────────────┼─────────────┐
                    ▼             ▼             ▼
           ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
           │  User 1 DB   │ │  User 2 DB   │ │  User N DB   │
           │invoices.db   │ │invoices.db   │ │invoices.db   │
           │              │ │              │ │              │
           │ entries      │ │ entries      │ │ entries      │
           │ rates        │ │ rates        │ │ rates        │
           │ invoices     │ │ invoices     │ │ invoices     │
           │ settings     │ │ settings     │ │ settings     │
           └──────────────┘ └──────────────┘ └──────────────┘
```

---

## Two-Tier Database Model

The application uses a **separation of concerns** between authentication data and business data.

### Tier 1: Auth DB (`data/auth.db`) — Shared

All users share a single SQLite database for authentication state.

```
┌─────────────────────────────────────────────┐
│              data/auth.db                   │
│                                             │
│  ┌──────────────────────────────────────┐  │
│  │              users                   │  │
│  │  id (PK)                             │  │
│  │  username (UNIQUE)                   │  │
│  │  display_name                        │  │
│  │  created_at                          │  │
│  └──────────────────────────────────────┘  │
│                                             │
│  ┌──────────────────────────────────────┐  │
│  │      webauthn_credentials            │  │
│  │  id (PK)                             │  │
│  │  user_id -> users.id                  │  │
│  │  credential_id (UNIQUE)              │  │
│  │  public_key                          │  │
│  │  attestation_type                    │  │
│  │  transports                          │  │
│  │  flags (JSON)                        │  │
│  │  authenticator (JSON)                │  │
│  └──────────────────────────────────────┘  │
│                                             │
│  ┌──────────────────────────────────────┐  │
│  │            sessions                  │  │
│  │  id (PK)                             │  │
│  │  user_id -> users.id                  │  │
│  │  token_hash (SHA256)                 │  │
│  │  expires_at                          │  │
│  └──────────────────────────────────────┘  │
└─────────────────────────────────────────────┘
```

### Tier 2: Per-User Data DB (`data/users/{userID}/invoices.db`) — Isolated

Each user gets their own SQLite file. This provides **true data isolation** without a complex database setup.

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ User 1 (ID: 1)  │    │ User 2 (ID: 2)  │    │ User 3 (ID: 3)  │
│                 │    │                 │    │                 │
│ data/users/1/   │    │ data/users/2/   │    │ data/users/3/   │
│  invoices.db    │    │  invoices.db    │    │  invoices.db    │
│                 │    │                 │    │                 │
│ - entries       │    │ - entries       │    │ - entries       │
│ - rates         │    │ - rates         │    │ - rates         │
│ - invoices      │    │ - invoices      │    │ - invoices      │
│ - invoice_items │    │ - invoice_items │    │ - invoice_items │
│ - settings      │    │ - settings      │    │ - settings      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

**Key design decision**: Rather than adding a `user_id` column to every table, we keep user data physically separate. This means:
- No risk of cross-user data leaks from a missing `WHERE user_id = ?`
- Easy backups per user
- Simpler data migration (just move a directory)

---

## Authentication Flow: WebAuthn (Passkeys)

This is a **passwordless** authentication system using the FIDO2/WebAuthn standard.

### Registration Flow

```
Browser                                          Server
─────────────────────────────────────────────────────────────────
   │                                                │
   │ POST /register/begin                           │
   │ { username: "alice", displayName: "Alice" }    │
   ├───────────────────────────────────────────────>│
   │                                                │
   │                                          ┌─────┴─────┐
   │                                          │ AuthDB    │
   │                                          │ CreateUser│
   │                                          │ -> ID: 7  │
   │                                          └─────┬─────┘
   │                                                │
   │                                          ┌─────┴─────┐
   │                                          │ WebAuthn  │
   │                                          │ BeginReg()│
   │                                          │ -> options│
   │                                          │ -> session│
   │                                          └─────┬─────┘
   │                                                │
   │ 200 OK                                         │
   │ { user_id: 7, options: { publicKey: {...} } } │
   │<───────────────────────────────────────────────│
   │                                                │
   │ [JavaScript: decode base64url -> ArrayBuffer] │
   │                                                │
   │ navigator.credentials.create(options)        │
   ├────────────────────────────────────────────────│
   │ (Browser prompts for fingerprint/YubiKey/etc)│
   │<────────────────────────────────────────────────
   │                                                │
   │ PublicKeyCredential {                          │
   │   rawId, response: {                           │
   │     clientDataJSON,                             │
   │     attestationObject                           │
   │   }                                             │
   │ }                                               │
   │<────────────────────────────────────────────────│
   │                                                │
   │ [JavaScript: encode ArrayBuffer -> base64url] │
   │                                                │
   │ POST /register/finish                          │
   │ { user_id: 7, id, rawId, type, response }    │
   ├───────────────────────────────────────────────>│
   │                                                │
   │                                          ┌─────┴─────┐
   │                                          │ WebAuthn  │
   │                                          │ FinishReg │
   │                                          │ -> verify │
   │                                          │ -> cred   │
   │                                          └─────┬─────┘
   │                                                │
   │                                          ┌─────┴─────┐
   │                                          │ AuthDB    │
   │                                          │ SaveCred  │
   │                                          └─────┬─────┘
   │                                                │
   │                                          ┌─────┴─────┐
   │                                          │ SessionMgr│
   │                                          │ Create()  │
   │                                          │ -> cookie │
   │                                          └─────┬─────┘
   │                                                │
   │ 200 OK { status: "ok" }                        │
   │ Set-Cookie: invoice_session=abc123...          │
   │<───────────────────────────────────────────────│
```

### Login Flow

```
Browser                                          Server
─────────────────────────────────────────────────────────────────
   │                                                │
   │ POST /login/begin                              │
   │ { username: "alice" }                          │
   ├───────────────────────────────────────────────>│
   │                                          ┌─────┴─────┐
   │                                          │ AuthDB    │
   │                                          │ LookupUser│
   │                                          │ -> ID: 7  │
   │                                          │ -> creds  │
   │                                          └─────┬─────┘
   │                                          ┌─────┴─────┐
   │                                          │ WebAuthn  │
   │                                          │ BeginLogin│
   │                                          │ -> options│
   │                                          │ -> session│
   │                                          └─────┬─────┘
   │ 200 OK                                         │
   │ { user_id: 7, options: { publicKey: {...} } } │
   │<───────────────────────────────────────────────│
   │                                                │
   │ [JavaScript: decode base64url -> ArrayBuffer] │
   │                                                │
   │ navigator.credentials.get(options)           │
   ├────────────────────────────────────────────────│
   │ (Browser prompts for passkey)                  │
   │<────────────────────────────────────────────────│
   │                                                │
   │ PublicKeyCredential {                          │
   │   rawId, response: {                           │
   │     clientDataJSON, authenticatorData,         │
   │     signature, userHandle                      │
   │   }                                             │
   │ }                                               │
   │<────────────────────────────────────────────────│
   │                                                │
   │ POST /login/finish                             │
   │ { user_id: 7, id, rawId, type, response }      │
   ├───────────────────────────────────────────────>│
   │                                          ┌─────┴─────┐
   │                                          │ WebAuthn  │
   │                                          │ FinishLog │
   │                                          │ -> verify │
   │                                          │ -> OK     │
   │                                          └─────┬─────┘
   │                                          ┌─────┴─────┐
   │                                          │ SessionMgr│
   │                                          │ Create()  │
   │                                          │ -> cookie │
   │                                          └─────┬─────┘
   │ 200 OK { status: "ok" }                        │
   │ Set-Cookie: invoice_session=abc123...          │
   │<───────────────────────────────────────────────│
```

---

## Passkeys Primer

Passkeys are asymmetric credentials created on a user's device and bound to a specific website (Relying Party). They replace passwords with public‑key cryptography.

- What gets created: a credential with a public key stored by the server and a private key kept by the authenticator (device, security key, or password manager synced across devices).
- Phishing resistance: the browser enforces origin and RP ID matching; the private key will not sign for the wrong site.
- Multi‑device: platform authenticators can sync passkeys via the OS account; cross‑device flows are handled by the browser/OS.

Terms you will see:
- RP ID: the effective domain for WebAuthn (e.g., `localhost` in dev, `example.com` in prod).
- Origin: scheme+host+port of the page performing WebAuthn (e.g., `https://example.com`). Must be allowed in config.
- Attestation: optional proof about the authenticator model. This project records attestation metadata but does not require a specific attestation policy.
- Discoverable credentials: allow username‑less login. This app prompts for username first and filters by that user's credentials.

---

## WebAuthn In This App

Code references:
- Server manager: `internal/auth/webauthn.go`
- HTTP handlers: `internal/handler/auth_handlers.go`
- Auth DB schema/CRUD: `internal/db/authdb.go`

Configuration (loaded in `cmd/server/main.go` → `AuthConfig`):
- `WEB_AUTHN_RP_ID` (RP ID / domain)
- `WEB_AUTHN_RP_ORIGINS` (allowed origins)
- `WEB_AUTHN_RP_DISPLAY` (friendly name shown in browser prompts)

Begin/Finish calls:
- Registration: `BeginRegistration(username, displayName)` then `FinishRegistration(sessionID, r)`
- Login: `BeginLogin(username)` then `FinishLogin(sessionID, r)`

Pending WebAuthn sessions:
- Stored in‑memory with a 5‑minute TTL cleanup. If a registration is abandoned and produced a user with no credentials, the cleanup loop deletes that orphaned user.

Attestation policy:
- The library defaults are used when calling `webauthn.BeginRegistration`. The app records `attestation_type` in the DB but does not enforce a specific attestation trust policy. This keeps UX simple; most consumer sites use attestation `none`.

Credential storage (`auth.db`):
- `webauthn_credentials` columns map to `webauthn.Credential` fields: `credential_id`, `public_key`, `attestation_type`, `transports`, `flags` (JSON), `authenticator` (JSON including counter/AAGUID when available).
- On login, the library verifies signature, RP ID hash, clientDataJSON type/origin/challenge, authenticator data, and counter semantics before returning success.

Why username‑first login:
- `BeginLogin(username)` returns an assertion filtered to that user's registered credential IDs. This avoids discoverable credentials complexity and keeps the UI predictable.

---

## Session Management

After WebAuthn success, the app issues a cookie session managed in `internal/auth/session.go` with backing rows in `auth.db`:

- Token generation: 32 random bytes, SHA‑256 hashed in DB. Only the base64url token is set in the cookie; server stores and looks up the hash.
- Cookie flags: `HttpOnly`, `SameSite=Lax`. `Secure` is determined per-request: enabled when `r.TLS != nil` or when `TRUSTED_PROXY=true` and `X-Forwarded-Proto: https` is present. In local dev over plain HTTP, it is disabled so browsers accept the cookie.
- TTL: `SESSION_TTL` (default 24h). Expiry is enforced on validation and sent as `Max‑Age`. A background ticker runs hourly calling `AuthDB.CleanupExpiredSessions()` to prune expired rows.
- Logout: double‑submit CSRF check on the form (`csrf_token` cookie and form value must match), then server clears session + CSRF cookies and deletes the DB row.
- CSRF protection: a global `CSRFMiddleware` validates the `csrf_token` cookie against a matching header (`X-CSRF-Token`) or form value on every POST/PUT/PATCH/DELETE request for non‑public paths. HTMX requests automatically include the token via a `htmx:configRequest` listener. Standard forms include a hidden `<input name="csrf_token">` populated by a script reading the cookie.

Route protection chain:
- `AuthMiddleware` skips `/login`, `/register`, `/static/*`.
- For other routes it loads the user from the session cookie and opens the per‑user SQLite via `MultiStore.ForUser(user.ID)`.
- Store and user are injected into the request context for handlers via `StoreFromContext` and `UserFromContext`.

Auth disabled mode:
- When `AUTH_ENABLED=false`, the middleware injects a legacy shared store and an anonymous user. All routes are public, and the old single‑DB layout is used.

---

## Security Properties And Defenses

- Phishing resistance: enforced by WebAuthn origin and RP ID checks; the authenticator will not sign for the wrong site.
- No passwords: eliminates credential stuffing and password database compromises.
- Session storage hardening: random tokens hashed with SHA‑256 in the DB; stolen DB rows do not reveal the bearer token.
- Cookie safety: `HttpOnly` prevents JS access; `SameSite=Lax` mitigates CSRF on state‑changing POSTs initiated by cross‑site navigations.
- CSRF on all unsafe routes: `CSRFMiddleware` validates a double‑submit token (cookie + header/form) on every POST/PUT/PATCH/DELETE to non‑public paths. HTMX requests auto‑inject the token, standard forms use a hidden input populated from the cookie.
- Security headers: all responses carry `X-Content-Type-Options: nosniff`, `Referrer-Policy`, `Permissions-Policy`, `X-Frame-Options: DENY`, and a Content‑Security‑Policy restricting scripts, styles, images, and objects to `'self'`.
- Cookie Secure hardening: `Secure` flag is set per‑request based on `r.TLS` or trusted proxy `X-Forwarded-Proto` header via `TRUSTED_PROXY` env var. No longer derived solely from static config origins.
- DB safety: per‑user SQLite files prevent cross‑tenant query bugs; there is no `user_id` filter to forget in business queries.
- MultiStore resource limits: per‑user DB handles are capped at 64, evicting the least‑recently‑used store when the limit is reached. Stores idle for >10 minutes are closed on a periodic sweep.
- Pending registration cleanup: abandoned registrations are pruned; orphaned users without credentials are removed automatically.
- Session cleanup: expired session rows are deleted hourly by a background ticker.

Considerations for production:
- HTTPS everywhere: ensure `WEB_AUTHN_RP_ORIGINS` are HTTPS so browsers allow passkeys.
- Set `TRUSTED_PROXY=true` when behind a reverse proxy that terminates TLS. The app will then trust the `X-Forwarded-Proto` header to set `Secure` cookies.
- Stable domain: RP ID must be a registrable domain you control; subdomain changes require careful RP ID choices.
- Reverse proxy: ensure the proxy strips incoming `X-Forwarded-Proto` from clients and sets it to the actual client‑facing protocol.
- Multi‑device passkeys: users may register multiple credentials; add a UI to manage and revoke passkeys if needed.

---

## Auth API Contract (Client ↔ Server)

Endpoints (JSON over POST):
- `POST /register/begin` → `{ username, displayName }` → `{ session_id, options }`
- `POST /register/finish` → body must include `{ session_id, ...webauthn response fields... }`
- `POST /login/begin` → `{ username }` → `{ session_id, user_id, options }`
- `POST /login/finish` → body must include `{ session_id, ...webauthn response fields... }`

Frontend responsibilities:
- Convert base64url to/from `ArrayBuffer` for `navigator.credentials.create` and `.get`.
- Send the browser‑returned `PublicKeyCredential` JSON back to `finish` endpoints unchanged (after encoding binary fields with base64url).
- On success, the server sets `invoice_session` and `csrf_token` cookies; then navigate to `/`.

Error handling:
- The server intentionally returns generic errors like `{"error":"login failed"}` to avoid leaking existence of usernames or verification details.

---

## Troubleshooting

- Passkey prompt not appearing:
  - Check browser console for WebAuthn errors.
  - Ensure `WEB_AUTHN_RP_ID` matches the effective domain and the page origin is listed in `WEB_AUTHN_RP_ORIGINS`.
  - In dev, use `http://localhost:8080`; many browsers restrict WebAuthn to secure contexts except for `localhost`.
- Cookie not set:
  - If using plain HTTP with a non‑localhost hostname, browsers may reject non‑secure cookies. Use HTTPS for custom domains.
  - Verify `Set‑Cookie` response headers and that `Secure` is correct for the environment.
- Stuck registration:
  - The in‑memory pending session expires after ~5 minutes. Re‑start from `/register`.
- Behind a proxy:
  - Make sure the external origin in `WEB_AUTHN_RP_ORIGINS` matches what the browser sees, not the internal service URL.

---

## Local Dev And Testing Tips

- Chrome DevTools → More Tools → WebAuthn: enable a virtual authenticator to test flows without hardware keys.
- Safari and Firefox also support platform authenticator testing on `localhost`.
- For cross‑device passkeys, test on a real phone signing into the same account and visiting the same RP ID domain.

---

## Further Reading

- WebAuthn Level 3 (W3C): https://www.w3.org/TR/webauthn-3/
- FIDO2 CTAP (client‑to‑authenticator): https://fidoalliance.org/specifications/
- Passkeys overview (passkeys.dev): https://passkeys.dev/
- Google Web Fundamentals: https://web.dev/passkeys/
- Yubico WebAuthn intro: https://www.yubico.com/learn/webauthn/
- go-webauthn library: https://github.com/go-webauthn/webauthn
- OWASP Session Management Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html


## Request Lifecycle

When a request hits the server, here is what happens:

```
┌─────────────┐
│ HTTP Request │
└──────┬──────┘
       │
       ▼
┌──────────────────┐
│ recoverMiddleware │  <- Catches panics, returns 500
└──────┬───────────┘
       │
       ▼
┌─────────────────────────────┐
│ SecurityHeadersMiddleware    │  <- CSP, X-Frame-Options, etc.
└──────┬──────────────────────┘
       │
       ▼
┌──────────────────────────┐
│ RequestLoggingMiddleware  │  <- Logs method, path, status, duration
└──────┬───────────────────┘
       │
       ▼
┌──────────────────┐
│ LoggerMiddleware  │  <- Injects logger into context
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│  AuthMiddleware   │
│                   │
│  Is public path?  │────Yes────┐
│      │                      │
│      No                     │
│      ▼                      │
│  Get cookie                 │
│      │                      │
│      ▼                      │
│  Validate token             │
│      │                      │
│      ▼                      │
│  Look up user               │
│      │                      │
│      ▼                      │
│  Open user's DB             │
│      │                      │
│      ▼                      │
│  Inject into context         │
└──────┬───────────┘          │
       │                      │
       ▼                      │
┌──────────────────┐          │
│  CSRFMiddleware   │<────────┘
│                   │
│  Is safe method?  │────Yes────┐
│  (GET/HEAD/OPT)? │          │
│      │                      │
│      No                     │
│      ▼                      │
│  Validate csrf_token │
│  cookie vs header/form│
└──────┬───────────┘          │
       │                      │
       ▼                      │
┌──────────────┐              │
│ Route Handler │<────────────┘
│               │
│ store :=      │
│ StoreFromCtx()│
│ user :=       │
│ UserFromCtx() │
│               │
│ store.Entries()│
│ store.Rates() │
│ etc.          │
└──────┬───────┘
       │
       ▼
┌─────────────┐
│ HTTP Response│
└─────────────┘
```

---

## Context-Based Dependency Injection

Instead of handlers holding a reference to a store, the store is injected into the request context by the auth middleware.

```go
// In auth middleware
ctx := context.WithValue(r.Context(), contextKeyStore, store)
ctx = context.WithValue(ctx, contextKeyUser, user)
next.ServeHTTP(w, r.WithContext(ctx))

// In any handler
func (a *App) dashboard(w http.ResponseWriter, r *http.Request) {
    store := StoreFromContext(r.Context())
    user := UserFromContext(r.Context())
    // ... use store and user
}
```

This design means:
- **No global state** — each request gets its own scoped dependencies
- **Easy testing** — `WithTestStore(ctx, mockStore)` allows injecting mocks
- **Thread-safe** — no shared *sql.DB across users

---

## Key Go Files

| File | Purpose |
|------|---------|
| `cmd/server/main.go` | Entrypoint: opens auth DB, wires managers, starts HTTP server, starts session cleanup ticker |
| `internal/handler/app.go` | Route setup, middleware chain (recover → security headers → logging → auth → CSRF → mux), context helpers, template rendering |
| `internal/handler/auth_handlers.go` | Login/register/logout HTTP handlers |
| `internal/handler/middleware_csrf.go` | CSRF validation middleware for all unsafe methods, extracts token from header or form field |
| `internal/handler/middleware_security.go` | Security headers middleware (CSP, X-Frame-Options, Referrer-Policy, etc.) |
| `internal/auth/webauthn.go` | WebAuthn Begin/Finish for registration and login |
| `internal/auth/session.go` | Cookie-based session manager (24h TTL), per-request Secure cookie detection via TLS or trusted proxy |
| `internal/auth/models.go` | WebAuthnUser wrapper, AuthConfig, SessionInfo |
| `internal/db/authdb.go` | AuthDB: user/credential/session CRUD in shared SQLite, session cleanup queries |
| `internal/db/multistore.go` | MultiStore: lazy-opens per-user DBs with LRU eviction (cap 64) and idle‑close sweep (10 min) |
| `cmd/migrate-to-user/main.go` | CLI tool to migrate legacy DB into a user's per-user DB |

---

## Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `ADDRESS` | `:8080` | Listen address |
| `AUTH_DB_PATH` | `data/auth.db` | Shared auth database |
| `USER_DB_DIR` | `data/users` | Per-user database directory |
| `AUTH_MIGRATIONS_DIR` | `auth-migrations` | Auth schema migrations |
| `MIGRATIONS_DIR` | `migrations` | Data schema migrations |
| `WEB_AUTHN_RP_ID` | `localhost` | WebAuthn domain |
| `WEB_AUTHN_RP_ORIGINS` | `http://localhost:8080` | Allowed origins |
| `WEB_AUTHN_RP_DISPLAY` | `Invoice App` | Display name |
| `SESSION_TTL` | `86400` | Session TTL in seconds |
| `TRUSTED_PROXY` | `false` | Set to `true` when behind a reverse proxy terminating TLS |
| `TYPST_BIN` | (auto-detect) | Typst binary for PDF generation |
| `SMTP_HOST` | (empty) | SMTP server (optional) |
| `OCR_ENABLED` | `false` | Enable OCR import |
