---
name: gortex-handler-process
description: "Work in the handler · process area — 30 symbols across 3 files (75% cohesion)"
---

# handler · process

30 symbols | 3 files | 75% cohesion

## When to Use

Use this skill when working on files in:
- `invoice-service/internal/handler/ocr.go`
- `invoice-service/internal/handler/ocr_jobs.go`
- `invoice-service/internal/handler/store_provider.go`

## Key Files

| File | Symbols |
|------|---------|
| `invoice-service/internal/handler/ocr.go` | store, ocrImporter |
| `invoice-service/internal/handler/ocr_jobs.go` | wg, userID, importer, sessionID, OCRJobRunner, ... |
| `invoice-service/internal/handler/store_provider.go` | userID, SetLegacyStore, RequestStoreProvider, multiStore, StoreProvider, ... |

## Connected Communities

- **handler +2 dirs · authMiddleware** (1 cross-edges)
- **ocrimport · ProcessSession** (1 cross-edges)
- **ocr +1 dirs** (1 cross-edges)

## How to Explore

```
get_communities with id: "community-60"
smart_context with task: "understand handler · process", format: "gcx"
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
