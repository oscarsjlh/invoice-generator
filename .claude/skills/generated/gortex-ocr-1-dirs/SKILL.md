---
name: gortex-ocr-1-dirs
description: "Work in the ocr +1 dirs area — 46 symbols across 4 files (91% cohesion)"
---

# ocr +1 dirs

46 symbols | 4 files | 91% cohesion

## When to Use

Use this skill when working on files in:
- `invoice-service/internal/ocr/client.go`
- `invoice-service/internal/ocr/client_test.go`
- `invoice-service/internal/ocr/contract.go`
- `invoice-service/internal/ocrimport/importer.go`

## Key Files

| File | Symbols |
|------|---------|
| `invoice-service/internal/ocr/client.go` | span, NewStubClient, recordSpanError, images, rates, ... |
| `invoice-service/internal/ocr/client_test.go` | t, closure@36, TestClientExtractInjectsTraceparent, closure@32 |
| `invoice-service/internal/ocr/contract.go` | Rate, SessionID, OCRRequest, Entries, Status, ... |
| `invoice-service/internal/ocrimport/importer.go` | categories, extract, rates, ctx, sessionID, ... |

## Entry Points

- `invoice-service/internal/ocr/client.go::Client.Extract`
- `invoice-service/internal/ocr/client_test.go::TestClientExtractInjectsTraceparent`

## Connected Communities

- **db +2 dirs** (1 cross-edges)

## How to Explore

```
get_communities with id: "community-71"
smart_context with task: "understand ocr +1 dirs", format: "gcx"
find_usages with id: "invoice-service/internal/ocr/client.go::Client.Extract", format: "gcx"
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
