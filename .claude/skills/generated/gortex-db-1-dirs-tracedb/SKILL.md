---
name: gortex-db-1-dirs-tracedb
description: "Work in the db +1 dirs · traceDB area — 56 symbols across 2 files (74% cohesion)"
---

# db +1 dirs · traceDB

56 symbols | 2 files | 74% cohesion

## When to Use

Use this skill when working on files in:
- `invoice-service/internal/db/storer.go`
- `invoice-service/internal/handler/traced_store.go`

## Key Files

| File | Symbols |
|------|---------|
| `invoice-service/internal/db/storer.go` | RateStore, Storer, OCRStore, EntryStore, InvoiceStore, ... |
| `invoice-service/internal/handler/traced_store.go` | DB, T, closure@35, traceDB, month, ... |

## Connected Communities

- **handler · ConfirmDraftEntries** (2 cross-edges)

## How to Explore

```
get_communities with id: "community-66"
smart_context with task: "understand db +1 dirs · traceDB", format: "gcx"
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
