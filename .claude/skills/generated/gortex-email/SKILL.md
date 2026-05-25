---
name: gortex-email
description: "Work in the email area — 41 symbols across 2 files (93% cohesion)"
---

# email

41 symbols | 2 files | 93% cohesion

## When to Use

Use this skill when working on files in:
- `invoice-service/internal/email/send.go`
- `invoice-service/internal/email/send_test.go`

## Key Files

| File | Symbols |
|------|---------|
| `invoice-service/internal/email/send.go` | Config, invoiceNumber, SMTPUser, NewDialer, DialAndSend, ... |
| `invoice-service/internal/email/send_test.go` | t, t, TestNewDialerInvalidPort, TestSendInvoiceBodyContainsRecipientName, t, ... |

## How to Explore

```
get_communities with id: "community-32"
smart_context with task: "understand email", format: "gcx"
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
