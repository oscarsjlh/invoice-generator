---
name: gortex-db-3-dirs
description: "Work in the db +3 dirs area — 50 symbols across 6 files (68% cohesion)"
---

# db +3 dirs

50 symbols | 6 files | 68% cohesion

## When to Use

Use this skill when working on files in:
- `invoice-service/cmd/migrate-to-user/main.go`
- `invoice-service/internal/db/authdb.go`
- `invoice-service/internal/db/db.go`
- `invoice-service/internal/db/multistore.go`
- `invoice-service/internal/ocr/preprocess.go`
- `invoice-service/internal/ocrimport/importer.go`

## Key Files

| File | Symbols |
|------|---------|
| `invoice-service/cmd/migrate-to-user/main.go` | closure@127, dst, closure@82, src, closure@36, ... |
| `invoice-service/internal/db/authdb.go` | path, OpenAuthDB |
| `invoice-service/internal/db/db.go` | DB |
| `invoice-service/internal/db/multistore.go` | Close, sweepIdle, closeAll |
| `invoice-service/internal/ocr/preprocess.go` | dstPath, closure@98, srcPath, closure@24, filePath, ... |
| `invoice-service/internal/ocrimport/importer.go` | closure@153, header, saveAndPreprocess |

## Entry Points

- `invoice-service/cmd/migrate-to-user/main.go::main`
- `invoice-service/cmd/migrate-to-user/main.go::migrateInvoices`

## Connected Communities

- **db · ForUser** (2 cross-edges)
- **handler +2 dirs · CreateUser** (2 cross-edges)
- **handler +4 dirs** (1 cross-edges)
- **config** (1 cross-edges)

## How to Explore

```
get_communities with id: "community-16"
smart_context with task: "understand db +3 dirs", format: "gcx"
find_usages with id: "invoice-service/cmd/migrate-to-user/main.go::main", format: "gcx"
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
