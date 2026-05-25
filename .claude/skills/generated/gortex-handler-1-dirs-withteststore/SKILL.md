---
name: gortex-handler-1-dirs-withteststore
description: "Work in the handler +1 dirs · WithTestStore area — 119 symbols across 12 files (86% cohesion)"
---

# handler +1 dirs · WithTestStore

119 symbols | 12 files | 86% cohesion

## When to Use

Use this skill when working on files in:
- `invoice-service/internal/handler/app.go`
- `invoice-service/internal/handler/e2e_test.go`
- `invoice-service/internal/handler/entries.go`
- `invoice-service/internal/handler/entries_test.go`
- `invoice-service/internal/handler/integration_test.go`
- `invoice-service/internal/handler/invoices_test.go`
- `invoice-service/internal/handler/ocr_test.go`
- `invoice-service/internal/handler/rates_test.go`
- `invoice-service/internal/handler/settings_test.go`
- `invoice-service/internal/handler/test_helpers.go`
- `invoice-service/internal/handler/traced_store.go`
- `invoice-service/internal/testutil/helpers.go`

## Key Files

| File | Symbols |
|------|---------|
| `invoice-service/internal/handler/app.go` | ctx, WithTestStore, store |
| `invoice-service/internal/handler/e2e_test.go` | t, createEntryViaHTTP, rate, t, endDate, ... |
| `invoice-service/internal/handler/entries.go` | r, w, entriesTable |
| `invoice-service/internal/handler/entries_test.go` | TestUpdateEntryMissingCategory, t, t, store, TestDeleteEntryRedirects, ... |
| `invoice-service/internal/handler/integration_test.go` | t, TestIntegrationDashboardReturns200, closure@48 |
| `invoice-service/internal/handler/invoices_test.go` | TestGenerateInvoiceSuccess, TestInvoicePreview404, TestDashboardRendersSuccessfully, t, t, ... |
| `invoice-service/internal/handler/ocr_test.go` | t, TestOCRUploadPageWhenDisabled |
| `invoice-service/internal/handler/rates_test.go` | t, TestDeleteRate, t, TestRatesPageReturns200, TestUpdateRateSuccess, ... |
| `invoice-service/internal/handler/settings_test.go` | TestSettingsPageReturnsSavedValues, t, t, TestSaveSettingsRedirects |
| `invoice-service/internal/handler/test_helpers.go` | urlencode, body, newFormRequest, url, values |
| `invoice-service/internal/handler/traced_store.go` | notes, SaveSettings, closure@57, closure@135, hours, ... |
| `invoice-service/internal/testutil/helpers.go` | SampleSettings |

## Entry Points

- `invoice-service/internal/handler/e2e_test.go::TestEntryCRUDViaHTTP`
- `invoice-service/internal/handler/invoices_test.go::TestInvoicePreviewReturns200`
- `invoice-service/internal/handler/e2e_test.go::TestRateManagementViaHTTP`
- `invoice-service/internal/handler/e2e_test.go::TestSettingsPersistenceViaHTTP`
- `invoice-service/internal/handler/invoices_test.go::TestDashboardRendersSuccessfully`

## Connected Communities

- **handler · LoggerFromContext** (40 cross-edges)
- **handler +4 dirs** (6 cross-edges)
- **handler · traceDBErr** (3 cross-edges)
- **db +1 dirs · traceDB** (1 cross-edges)
- **handler · Close** (1 cross-edges)
- **db · ForUser** (1 cross-edges)

## How to Explore

```
get_communities with id: "community-41"
smart_context with task: "understand handler +1 dirs · WithTestStore", format: "gcx"
find_usages with id: "invoice-service/internal/handler/e2e_test.go::TestEntryCRUDViaHTTP", format: "gcx"
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
