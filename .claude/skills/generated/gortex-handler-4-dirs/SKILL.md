---
name: gortex-handler-4-dirs
description: "Work in the handler +4 dirs area — 70 symbols across 12 files (68% cohesion)"
---

# handler +4 dirs

70 symbols | 12 files | 68% cohesion

## When to Use

Use this skill when working on files in:
- `invoice-service/cmd/server/main.go`
- `invoice-service/internal/db/db.go`
- `invoice-service/internal/db/multistore.go`
- `invoice-service/internal/handler/app.go`
- `invoice-service/internal/handler/integration_test.go`
- `invoice-service/internal/handler/logger.go`
- `invoice-service/internal/handler/middleware_test.go`
- `invoice-service/internal/handler/ocr_jobs.go`
- `invoice-service/internal/handler/ocr_test.go`
- `invoice-service/internal/handler/store_provider.go`
- `invoice-service/internal/testutil/helpers.go`
- `invoice-service/internal/tracing/tracing.go`

## Key Files

| File | Symbols |
|------|---------|
| `invoice-service/cmd/server/main.go` | closure@27, closure@121, closure@46, main, closure@98 |
| `invoice-service/internal/db/db.go` | filename, filename, isMigrationApplied, path, Migrate, ... |
| `invoice-service/internal/db/multistore.go` | userID, userStoreFactory, open, s, migrationsDir, ... |
| `invoice-service/internal/handler/app.go` | WaitForOCR, App, closure@132, stores, ocrJobs, ... |
| `invoice-service/internal/handler/integration_test.go` | t, newTestServer |
| `invoice-service/internal/handler/logger.go` | includeSource, format, NewLogger, level |
| `invoice-service/internal/handler/middleware_test.go` | t, closure@331, TestAuthMiddlewareDisabledWithLegacyStore |
| `invoice-service/internal/handler/ocr_jobs.go` | Wait |
| `invoice-service/internal/handler/ocr_test.go` | t, TestOCRUploadPageWhenEnabled |
| `invoice-service/internal/handler/store_provider.go` | NewRequestStoreProvider, multiStore |
| `invoice-service/internal/testutil/helpers.go` | t, MigrationsDir, NewTestDB, closure@29, t |
| `invoice-service/internal/tracing/tracing.go` | Init, ctx, serviceName, configuredSampler, closure@23 |

## Entry Points

- `invoice-service/cmd/server/main.go::main`
- `invoice-service/internal/handler/middleware_test.go::TestAuthMiddlewareDisabledWithLegacyStore`
- `invoice-service/internal/tracing/tracing.go::Init`
- `invoice-service/internal/handler/ocr_test.go::TestOCRUploadPageWhenEnabled`

## Connected Communities

- **db +3 dirs** (5 cross-edges)
- **handler · LoggerFromContext** (5 cross-edges)
- **db · ForUser** (4 cross-edges)
- **handler +2 dirs · authMiddleware** (2 cross-edges)
- **handler +2 dirs · CreateUser** (2 cross-edges)
- **db · Migrate** (2 cross-edges)
- **config** (1 cross-edges)
- **handler · process** (1 cross-edges)
- **handler · WriteHeader** (1 cross-edges)
- **handler +1 dirs · WithTestStore** (1 cross-edges)
- **auth · NewWebAuthnManager** (1 cross-edges)
- **handler · Renderer** (1 cross-edges)
- **testutil** (1 cross-edges)

## How to Explore

```
get_communities with id: "community-20"
smart_context with task: "understand handler +4 dirs", format: "gcx"
find_usages with id: "invoice-service/cmd/server/main.go::main", format: "gcx"
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
