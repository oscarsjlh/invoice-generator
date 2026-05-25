---
name: gortex-invoicedelivery-2-dirs
description: "Work in the invoicedelivery +2 dirs area — 81 symbols across 4 files (89% cohesion)"
---

# invoicedelivery +2 dirs

81 symbols | 4 files | 89% cohesion

## When to Use

Use this skill when working on files in:
- `invoice-service/internal/db/models.go`
- `invoice-service/internal/invoicedelivery/delivery.go`
- `invoice-service/internal/invoicedelivery/delivery_test.go`
- `invoice-service/internal/pdf/generate.go`

## Key Files

| File | Symbols |
|------|---------|
| `invoice-service/internal/db/models.go` | Month, AccountName, PaymentTerms, ID, AccountNumber, ... |
| `invoice-service/internal/invoicedelivery/delivery.go` | err, ctx, ctx, Renderer, closure@187, ... |
| `invoice-service/internal/invoicedelivery/delivery_test.go` | TestSendRequiresCustomerEmail, err, fakeRenderer, invoice, TestRenderPDF, ... |
| `invoice-service/internal/pdf/generate.go` | GenerateInvoicePDF, workDir, findTypst, templateBytes, typContent |

## Entry Points

- `invoice-service/internal/pdf/generate.go::GenerateInvoicePDF`

## Connected Communities

- **db +2 dirs** (4 cross-edges)
- **pdf** (1 cross-edges)
- **pdf +1 dirs** (1 cross-edges)

## How to Explore

```
get_communities with id: "community-70"
smart_context with task: "understand invoicedelivery +2 dirs", format: "gcx"
find_usages with id: "invoice-service/internal/pdf/generate.go::GenerateInvoicePDF", format: "gcx"
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
