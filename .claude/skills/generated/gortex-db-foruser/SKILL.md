---
name: gortex-db-foruser
description: "Work in the db · ForUser area — 34 symbols across 3 files (75% cohesion)"
---

# db · ForUser

34 symbols | 3 files | 75% cohesion

## When to Use

Use this skill when working on files in:
- `invoice-service/internal/db/entries_test.go`
- `invoice-service/internal/db/multistore.go`
- `invoice-service/internal/db/multistore_test.go`

## Key Files

| File | Symbols |
|------|---------|
| `invoice-service/internal/db/entries_test.go` | findMigrationsDir, t |
| `invoice-service/internal/db/multistore.go` | dir, get, NewMultiStore, userID, userID, ... |
| `invoice-service/internal/db/multistore_test.go` | TestMultiStoreLRUEviction, t, closure@80, t, closure@109, ... |

## Connected Communities

- **db +3 dirs** (8 cross-edges)
- **handler +4 dirs** (2 cross-edges)
- **db · newMultiStore** (2 cross-edges)
- **db +2 dirs** (2 cross-edges)
- **db · userStoreCache** (1 cross-edges)

## How to Explore

```
get_communities with id: "community-21"
smart_context with task: "understand db · ForUser", format: "gcx"
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
