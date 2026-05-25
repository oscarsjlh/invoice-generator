---
name: gortex-ocrimport-confirmdrafts
description: "Work in the ocrimport · ConfirmDrafts area — 45 symbols across 2 files (85% cohesion)"
---

# ocrimport · ConfirmDrafts

45 symbols | 2 files | 85% cohesion

## When to Use

Use this skill when working on files in:
- `invoice-service/internal/ocrimport/importer.go`
- `invoice-service/internal/ocrimport/importer_test.go`

## Key Files

| File | Symbols |
|------|---------|
| `invoice-service/internal/ocrimport/importer.go` | ctx, date, Confirmed, Hours, SessionID, ... |
| `invoice-service/internal/ocrimport/importer_test.go` | category, UpdateDraftEntry, GetDraftEntry, confirmedID, ids, ... |

## Connected Communities

- **db +2 dirs** (1 cross-edges)

## How to Explore

```
get_communities with id: "community-78"
smart_context with task: "understand ocrimport · ConfirmDrafts", format: "gcx"
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
