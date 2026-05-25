---
name: gortex-handler-tracedberr
description: "Work in the handler · traceDBErr area — 40 symbols across 1 files (82% cohesion)"
---

# handler · traceDBErr

40 symbols | 1 files | 82% cohesion

## When to Use

Use this skill when working on files in:
- `invoice-service/internal/handler/traced_store.go`

## Key Files

| File | Symbols |
|------|---------|
| `invoice-service/internal/handler/traced_store.go` | id, category, UpdateEntry, sessionID, notes, ... |

## Connected Communities

- **handler · ConfirmDraftEntries** (2 cross-edges)

## How to Explore

```
get_communities with id: "community-64"
smart_context with task: "understand handler · traceDBErr", format: "gcx"
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
