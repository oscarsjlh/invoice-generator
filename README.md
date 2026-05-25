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

## Production Browser Gate

Playwright E2E tests live in `invoice-service/tests/e2e`. The production confidence gate runs Chromium against three Docker app services:

- `app-auth-on`: auth-enabled mode with e2e-only session setup helpers.
- `app-noauth-1` and `app-noauth-2`: no-auth compatibility mode with isolated legacy SQLite databases for two Playwright workers.

Run the same Chromium gate locally and in CI with:

```bash
cd invoice-service
npx pnpm install
npx pnpm test:e2e
```

The command builds only the e2e Docker image with `go build -tags=e2e`, starts isolated auth-on and no-auth app containers, runs two Playwright workers with one CI retry, then removes the containers and volumes. Test helper routes require `E2E_TEST_HELPERS=true` and are absent from normal Go builds.

To inspect labelled service logs after a failed run:

```bash
cd invoice-service
npx pnpm test:e2e:logs
```

Failure artifacts include the Playwright HTML report, traces, screenshots, video-on-failure, JSON retry results, browser console logs, and Docker logs for `app-auth-on`, `app-noauth-1`, `app-noauth-2`, and `playwright`.

Chromium is the required production gate for relevant PRs, `main`, and release tags. Firefox and WebKit are reserved for scheduled non-blocking jobs until a later change promotes them to blockers.

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
