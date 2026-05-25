---
name: gortex-handler-loggerfromcontext
description: "Work in the handler · LoggerFromContext area — 155 symbols across 11 files (80% cohesion)"
---

# handler · LoggerFromContext

155 symbols | 11 files | 80% cohesion

## When to Use

Use this skill when working on files in:
- `invoice-service/internal/handler/app.go`
- `invoice-service/internal/handler/auth_handlers.go`
- `invoice-service/internal/handler/dashboard.go`
- `invoice-service/internal/handler/entries.go`
- `invoice-service/internal/handler/helpers_test.go`
- `invoice-service/internal/handler/invoices.go`
- `invoice-service/internal/handler/logger.go`
- `invoice-service/internal/handler/ocr.go`
- `invoice-service/internal/handler/rates.go`
- `invoice-service/internal/handler/settings.go`
- `invoice-service/internal/handler/traced_store.go`

## Key Files

| File | Symbols |
|------|---------|
| `invoice-service/internal/handler/app.go` | value, User, parsePositiveInt, parseInt64Path, renderPage, ... |
| `invoice-service/internal/handler/auth_handlers.go` | registerPage, r, r, w, r, ... |
| `invoice-service/internal/handler/dashboard.go` | dashboard, r, w |
| `invoice-service/internal/handler/entries.go` | deleteEntry, entriesPage, r, w, w, ... |
| `invoice-service/internal/handler/helpers_test.go` | closure@81, TestValidateDate, closure@293, TestIsHTMX, t, ... |
| `invoice-service/internal/handler/invoices.go` | w, invoicePDF, r, sendInvoice, w, ... |
| `invoice-service/internal/handler/logger.go` | closure@124, LoggerFromContext, RequestLoggingMiddleware, addr, redactRemoteAddr, ... |
| `invoice-service/internal/handler/ocr.go` | ocrStartSession, ocrUploadPage, ocrDeleteSession, w, ocrSessionStatus, ... |
| `invoice-service/internal/handler/rates.go` | w, ratesPage, editRateForm, r, r, ... |
| `invoice-service/internal/handler/settings.go` | r, r, saveSettings, settingsPage, w, ... |
| `invoice-service/internal/handler/traced_store.go` | ListCategories, GetInvoice, closure@121, ListInvoices, id, ... |

## Entry Points

- `invoice-service/internal/handler/invoices.go::App.sendInvoice`
- `invoice-service/internal/handler/invoices.go::App.invoicePDF`
- `invoice-service/internal/handler/ocr.go::App.ocrStartSession`
- `invoice-service/internal/handler/app.go::App.Routes`
- `invoice-service/internal/handler/logger.go::RequestLoggingMiddleware`

## Connected Communities

- **db +1 dirs · traceDB** (15 cross-edges)
- **handler · traceDBErr** (4 cross-edges)
- **invoicedelivery +2 dirs** (4 cross-edges)
- **handler · process** (3 cross-edges)
- **handler +1 dirs · WithTestStore** (3 cross-edges)
- **db +2 dirs** (2 cross-edges)
- **handler · WriteHeader** (2 cross-edges)
- **handler +2 dirs · authMiddleware** (1 cross-edges)
- **handler · Page** (1 cross-edges)
- **handler · TestValidateMonth** (1 cross-edges)
- **handler · recoverMiddleware** (1 cross-edges)
- **handler · LoggerMiddleware** (1 cross-edges)
- **ocrimport · StartSession** (1 cross-edges)
- **handler +2 dirs · CreateUser** (1 cross-edges)
- **handler · ConfirmDraftEntries** (1 cross-edges)
- **handler · TestNormalizeCategory** (1 cross-edges)
- **handler · TestSecurityHeadersMiddleware** (1 cross-edges)
- **handler · Write** (1 cross-edges)

## How to Explore

```
get_communities with id: "community-49"
smart_context with task: "understand handler · LoggerFromContext", format: "gcx"
find_usages with id: "invoice-service/internal/handler/invoices.go::App.sendInvoice", format: "gcx"
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
