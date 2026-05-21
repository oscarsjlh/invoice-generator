# AGENTS.md

## Architecture

Single-binary Go web app (Go 1.26) using SQLite, `net/http`, `html/template`, and HTMX.

- **Entrypoint:** `invoice-service/cmd/server/main.go` — web server on `:8080` by default
- **CSV importer:** `invoice-service/cmd/migrate-csv/main.go` — one-shot migration from spreadsheet exports
- **Data migrator:** `invoice-service/cmd/migrate-to-user/main.go` — migrates legacy data into a user's per-user database
- **Database:** Multi-tenant SQLite via `modernc.org/sqlite` (pure Go, no CGO). Two tiers:
  - **Auth DB** (`data/auth.db`): Shared database with users, WebAuthn credentials, and sessions (only when `AUTH_ENABLED=true`)
  - **Per-user DB** (`data/users/{userID}/invoices.db`): Each user gets their own SQLite file, automatically created and migrated on first login (only when `AUTH_ENABLED=true`)
  - **Legacy DB** (`data/invoices.db`): Single database used when `AUTH_ENABLED=false`
- **Authentication:** WebAuthn (passkeys) via `github.com/go-webauthn/webauthn` with cookie-based sessions (24h TTL, HMAC-signed). Gated by `AUTH_ENABLED` env var (default `true`).
- **Authentication:** WebAuthn (passkeys) via `github.com/go-webauthn/webauthn` with cookie-based sessions (24h TTL, HMAC-signed)
- **Templates:** Go `html/template` in `invoice-service/templates/`. `layout.html` is the base template; page templates are rendered through it via `renderPage`. HTMX partials use `renderPartial` instead.
- **PDF:** Generated via external `typst` binary (v0.13.1) using the template at `invoice-service/templates/invoice-maker.typ`. Typst must be installed separately for local dev.
- **OCR:** Optional paper-entry import using AWS Bedrock (separate Python service in `ocr-service/`). See `docs/ocr-setup.md`. Disabled by default (`OCR_ENABLED=false`).
- **Email:** SMTP via `github.com/go-mail/mail/v2`. Optional — only needed for the "Send Invoice" feature.

## Auth flow

1. User visits `/register` → creates a passkey via WebAuthn → user + credential stored in auth DB → per-user data DB created under `data/users/{id}/invoices.db` → session cookie set
2. User visits `/login` → enters username → browser prompts for passkey → session cookie set
3. All protected routes check session via `AuthMiddleware` (in `handler/app.go`) → user's store injected into `r.Context()`
4. Handlers call `StoreFromContext(r.Context())` to get the user-specific database store
5. Public routes: `/login`, `/register`, `/static/*` — no auth required

## Commands

```bash
# Run server locally (from invoice-service/)
make -C invoice-service run   # or: go run ./invoice-service/cmd/server

# Build binary (from invoice-service/)
make -C invoice-service build   # or: go build ./invoice-service/cmd/server

# Import CSV data
make -C invoice-service migrate-csv   # or: go run ./invoice-service/cmd/migrate-csv --entries data/entries.csv --rates data/rates.csv

# Migrate legacy DB to a user
go run ./invoice-service/cmd/migrate-to-user --user-id 1 --from data/invoices.db

# Format code (only available lint/fmt tool)
make -C invoice-service fmt   # or: go fmt ./invoice-service/...

# Docker
docker compose up --build
```

## Environment variables

All config is via env vars (no `.env` file loading):

| Variable | Default | Notes |
|---|---|---|
| `ADDRESS` | `:8080` | Listen address |
| `DATABASE_PATH` | `data/invoices.db` | No longer used for new data; legacy path for migration tool |
| `TEMPLATES_DIR` | `templates` | |
| `MIGRATIONS_DIR` | `migrations` | |
| `TYPST_BIN` | (auto-detect) | Override typst binary path |
| `SMTP_HOST` | (empty) | SMTP disabled if blank |
| `SMTP_PORT` | `587` | |
| `SMTP_USER` | (empty) | |
| `SMTP_PASS` | (empty) | |
| `SMTP_FROM` | (empty) | |
| `OCR_ENABLED` | `false` | Set to `true` to enable OCR import routes |
| `OCR_SERVICE_URL` | (empty) | OCR service endpoint (e.g., `http://localhost:8000`) |
| `OCR_UPLOAD_DIR` | `data/ocr-uploads` | Directory for uploaded images |
| `BEDROCK_REGION` | `us-east-1` | AWS region for Bedrock (OCR service only) |
| `BEDROCK_MODEL` | `us.anthropic.claude-3-5-haiku-20241022-v1:0` | Bedrock model ID (OCR service only) |
| `LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | `json` | Log format: `json` (structured) or `text` (human-readable) |
| `AUTH_ENABLED` | `true` | Set to `false` to disable auth and use legacy single-user mode |
| `AUTH_DB_PATH` | `data/auth.db` | Path to shared auth database |
| `USER_DB_DIR` | `data/users` | Directory for per-user SQLite databases |
| `WEB_AUTHN_RP_ID` | `localhost` | WebAuthn Relying Party ID (domain) |
| `WEB_AUTHN_RP_ORIGINS` | `http://localhost:8080` | WebAuthn allowed origin(s) |
| `WEB_AUTHN_RP_DISPLAY` | `Invoice App` | WebAuthn Relying Party display name |
| `SESSION_TTL` | `86400` | Session TTL in seconds (default 24h) |

## Go router notes

Uses Go 1.22+ enhanced `ServeMux` patterns: `GET /`, `POST /entries/{id}/delete`, `GET /invoices/{id}`, etc. Path parameters are extracted with `r.PathValue("id")`.

When `AUTH_ENABLED=true` (default):
- **Public** (no auth): `/login`, `/register`, `/static/*`
- **Protected** (requires session): all other routes wrapped in `AuthMiddleware`

When `AUTH_ENABLED=false`:
- All routes are accessible without authentication
- Auth routes (`/login`, `/register`, `/logout`) are not registered
- A single legacy database (`data/invoices.db`) is used for all data

Handlers access the per-user store via `StoreFromContext(r.Context())` instead of a shared `a.store` field.

## Internal packages

- `internal/db/authdb.go` — `AuthDB` type for user/credential/session management in the shared auth database
- `internal/db/multistore.go` — `MultiStore` type for opening per-user data databases on demand (lazy open + cache)
- `internal/auth/` — WebAuthn flow manager, session manager (cookie-based), and user types implementing `webauthn.User` interface
- `internal/handler/auth_handlers.go` — Login/register/logout handler functions

## Key constraints

- **No test suite exists.** There is no test command. `go fmt ./...` is the only code quality tool.
- **`typst` must be installed for PDF generation.** The Docker image bundles it at `/usr/local/bin/typst`. For local dev, install it separately or PDF endpoints will error. The `findTypst()` function checks `typst`, `typst-cli`, and common paths.
- **Invoice generation fails if any matching entries lack a rate.** The `CountUnratedEntries` check gates the transaction. If it returns > 0, the invoice is not created and an error is returned.
- **Invoice numbers** follow the pattern `INV-{YYYY-MM}-{SLUG}`. The slug is the uppercase category with non-alphanumerics stripped. If the number already exists, a sequential suffix (`-2`, `-3`, etc.) is appended.
- **Database pragmas** set on open: `WAL`, `foreign_keys = ON`, `busy_timeout = 5000`.
- **Address parsing** (`parseAddress` in `internal/handler/invoices.go`) splits on newlines with a 3-line convention: street, city, postal code. Changing this logic affects the PDF template.
- **HTMX detection:** `isHTMX(r)` checks the `HX-Request` header. Deletes and rate updates return partial HTML for HTMX requests vs redirects for normal requests.
- **CSV date normalization** handles Excel serial date numbers (days since 1899-12-30), `YYYY-MM-DD`, `DD/MM/YYYY`, and `DD/MM/YY` formats.
