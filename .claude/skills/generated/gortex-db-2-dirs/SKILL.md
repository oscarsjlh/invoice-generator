---
name: gortex-db-2-dirs
description: "Work in the db +2 dirs area — 156 symbols across 13 files (83% cohesion)"
---

# db +2 dirs

156 symbols | 13 files | 83% cohesion

## When to Use

Use this skill when working on files in:
- `invoice-service/internal/db/db.go`
- `invoice-service/internal/db/entries.go`
- `invoice-service/internal/db/entries_test.go`
- `invoice-service/internal/db/invoices.go`
- `invoice-service/internal/db/invoices_test.go`
- `invoice-service/internal/db/models.go`
- `invoice-service/internal/db/multistore.go`
- `invoice-service/internal/db/rates.go`
- `invoice-service/internal/db/rates_test.go`
- `invoice-service/internal/handler/app.go`
- `invoice-service/internal/handler/helpers_test.go`
- `invoice-service/internal/handler/ocr.go`
- `invoice-service/internal/ocrimport/importer.go`

## Key Files

| File | Symbols |
|------|---------|
| `invoice-service/internal/db/db.go` | Close |
| `invoice-service/internal/db/entries.go` | id, category, CreateEntry, notes, UpdateEntry, ... |
| `invoice-service/internal/db/entries_test.go` | TestEntryCRUD, TestUpdateEntry, closure@107, closure@137, setupTestDB, ... |
| `invoice-service/internal/db/invoices.go` | closure@150, countUnratedEntriesTx, closure@410, settings, month, ... |
| `invoice-service/internal/db/invoices_test.go` | createInvoiceForMonth, t, TestFilteredSummary, t, t, ... |
| `invoice-service/internal/db/models.go` | Month, EndDate, StartDate, Category, Notes, ... |
| `invoice-service/internal/db/multistore.go` | len |
| `invoice-service/internal/db/rates.go` | closure@113, CreateRate, id, category, GetRate, ... |
| `invoice-service/internal/db/rates_test.go` | t, t, TestDeleteRate, closure@61, closure@120, ... |
| `invoice-service/internal/handler/app.go` | n, formatWithCommas |
| `invoice-service/internal/handler/helpers_test.go` | TestFormatWithCommas, t, closure@158 |
| `invoice-service/internal/handler/ocr.go` | w, drafts, r, mergeDraftCategories, ocrConfirmDrafts, ... |
| `invoice-service/internal/ocrimport/importer.go` | buildDraftEntries, categories, result |

## Entry Points

- `invoice-service/internal/handler/ocr.go::App.ocrConfirmDrafts`
- `invoice-service/internal/db/invoices_test.go::TestFilteredSummary`
- `invoice-service/internal/db/invoices_test.go::TestGetInvoiceWithLines`
- `invoice-service/internal/db/rates_test.go::TestListAvailableMonths`
- `invoice-service/internal/db/invoices_test.go::TestGenerateInvoice`

## Connected Communities

- **db +3 dirs** (19 cross-edges)
- **handler · LoggerFromContext** (6 cross-edges)
- **handler +4 dirs** (2 cross-edges)
- **db · UpdateRate** (2 cross-edges)
- **ocr · OCRExtractedField** (1 cross-edges)
- **db · nextInvoiceNumber** (1 cross-edges)
- **db · ForUser** (1 cross-edges)
- **handler · process** (1 cross-edges)
- **ocrimport · ConfirmDrafts** (1 cross-edges)
- **db · InvoiceLine** (1 cross-edges)

## How to Explore

```
get_communities with id: "community-18"
smart_context with task: "understand db +2 dirs", format: "gcx"
find_usages with id: "invoice-service/internal/handler/ocr.go::App.ocrConfirmDrafts", format: "gcx"
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
