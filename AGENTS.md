# AGENTS.md

## Architecture

Single-binary Go web app (Go 1.26) using SQLite, `net/http`, `html/template`, and HTMX.

- **Entrypoint:** `cmd/server/main.go` — web server on `:8080` by default
- **CSV importer:** `cmd/migrate-csv/main.go` — one-shot migration from spreadsheet exports
- **Database:** SQLite via `modernc.org/sqlite` (pure Go, no CGO). Migrations run automatically at startup from `migrations/*.sql` (sorted by filename, no version tracking — use `IF NOT EXISTS`).
- **Templates:** Go `html/template` in `templates/`. `layout.html` is the base template; page templates are rendered through it via `renderPage`. HTMX partials use `renderPartial` instead.
- **PDF:** Generated via external `typst` binary (v0.13.1) using the template at `templates/invoice-maker.typ`. Typst must be installed separately for local dev.
- **Email:** SMTP via `github.com/go-mail/mail/v2`. Optional — only needed for the "Send Invoice" feature.

## Commands

```bash
# Run server locally
make run   # or: go run ./cmd/server

# Build binary
make build   # or: go build ./cmd/server

# Import CSV data
make migrate-csv   # or: go run ./cmd/migrate-csv --entries data/entries.csv --rates data/rates.csv

# Format code (only available lint/fmt tool)
make fmt   # or: go fmt ./...

# Docker
docker compose up --build
```

## Environment variables

All config is via env vars (no `.env` file loading):

| Variable | Default | Notes |
|---|---|---|
| `ADDRESS` | `:8080` | Listen address |
| `DATABASE_PATH` | `data/invoices.db` | SQLite file path |
| `TEMPLATES_DIR` | `templates` | |
| `MIGRATIONS_DIR` | `migrations` | |
| `TYPST_BIN` | (auto-detect) | Override typst binary path |
| `SMTP_HOST` | (empty) | SMTP disabled if blank |
| `SMTP_PORT` | `587` | |
| `SMTP_USER` | (empty) | |
| `SMTP_PASS` | (empty) | |
| `SMTP_FROM` | (empty) | |

## Go router notes

Uses Go 1.22+ enhanced `ServeMux` patterns: `GET /`, `POST /entries/{id}/delete`, `GET /invoices/{id}`, etc. Path parameters are extracted with `r.PathValue("id")`.

## Key constraints

- **No test suite exists.** There is no test command. `go fmt ./...` is the only code quality tool.
- **`typst` must be installed for PDF generation.** The Docker image bundles it at `/usr/local/bin/typst`. For local dev, install it separately or PDF endpoints will error. The `findTypst()` function checks `typst`, `typst-cli`, and common paths.
- **Invoice generation fails if any matching entries lack a rate.** The `CountUnratedEntries` check gates the transaction. If it returns > 0, the invoice is not created and an error is returned.
- **Invoice numbers** follow the pattern `INV-{YYYY-MM}-{SLUG}`. The slug is the uppercase category with non-alphanumerics stripped. If the number already exists, a sequential suffix (`-2`, `-3`, etc.) is appended.
- **Database pragmas** set on open: `WAL`, `foreign_keys = ON`, `busy_timeout = 5000`.
- **Address parsing** (`parseAddress` in `handler/invoices.go`) splits on newlines with a 3-line convention: street, city, postal code. Changing this logic affects the PDF template.
- **HTMX detection:** `isHTMX(r)` checks the `HX-Request` header. Deletes and rate updates return partial HTML for HTMX requests vs redirects for normal requests.
- **CSV date normalization** handles Excel serial date numbers (days since 1899-12-30), `YYYY-MM-DD`, `DD/MM/YYYY`, and `DD/MM/YY` formats.
