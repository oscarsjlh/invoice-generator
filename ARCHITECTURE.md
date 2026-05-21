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
| `cmd/server/main.go` | Entrypoint: opens auth DB, wires managers, starts HTTP server |
| `internal/handler/app.go` | Route setup, middleware chain, context helpers, template rendering |
| `internal/handler/auth_handlers.go` | Login/register/logout HTTP handlers |
| `internal/auth/webauthn.go` | WebAuthn Begin/Finish for registration and login |
| `internal/auth/session.go` | Cookie-based session manager (24h TTL) |
| `internal/auth/models.go` | WebAuthnUser wrapper, AuthConfig, SessionInfo |
| `internal/db/authdb.go` | AuthDB: user/credential/session CRUD in shared SQLite |
| `internal/db/multistore.go` | MultiStore: lazy-opens per-user DBs with caching |
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
| `TYPST_BIN` | (auto-detect) | Typst binary for PDF generation |
| `SMTP_HOST` | (empty) | SMTP server (optional) |
| `OCR_ENABLED` | `false` | Enable OCR import |
