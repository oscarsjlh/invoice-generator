---
name: gortex-pdf
description: "Work in the pdf area — 38 symbols across 2 files (98% cohesion)"
---

# pdf

38 symbols | 2 files | 98% cohesion

## When to Use

Use this skill when working on files in:
- `invoice-service/internal/pdf/generate.go`
- `invoice-service/internal/pdf/generate_test.go`

## Key Files

| File | Symbols |
|------|---------|
| `invoice-service/internal/pdf/generate.go` | AccountName, InvoiceData, SortCode, PaymentTerms, CustomerTitle, ... |
| `invoice-service/internal/pdf/generate_test.go` | t, TestFormatInvoiceTypNoTitle, t, t, TestFormatInvoiceTypEscapesSpecialChars, ... |

## How to Explore

```
get_communities with id: "community-81"
smart_context with task: "understand pdf", format: "gcx"
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
