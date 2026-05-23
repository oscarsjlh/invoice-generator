# Invoice App

Minimal self-hosted invoice application built with Go, SQLite, `net/http`, `html/template`, and HTMX.

## Features

- Time entry CRUD
- Rate management by category and date range
- Invoice generation with stored snapshots
- Printable HTML invoice previews
- Business settings page
- CSV importer for migrating spreadsheet exports

## Run Locally

```bash
cd invoice-app
go run ./cmd/server
```

Then open `http://localhost:8080`.

## CSV Migration

Export your Sheets tabs to CSV, then run:

```bash
cd invoice-app
go run ./cmd/migrate-csv --entries /path/to/entries.csv --rates /path/to/rates.csv
```

## Docker

```bash
cd invoice-app
docker compose up --build
```

## CI/CD Release (Dagger + Woodpecker)

Releases are driven by Dagger and triggered from Woodpecker on git tags matching `v*`.

### Required Woodpecker Secrets

- `registry_username`
- `registry_password`

These are exposed to Dagger as `REGISTRY_USERNAME` and `REGISTRY_PASSWORD`.

### Release Trigger

Create and push a version tag:

```bash
git tag v1.2.3
git push origin v1.2.3
```

The pipeline publishes:

- `registry.oscarcorner.com/invoices:v1.2.3` and `:latest`
- `registry.oscarcorner.com/invoices-ocr:v1.2.3` and `:latest`
