---
name: gortex-config
description: "Work in the config area — 53 symbols across 2 files (98% cohesion)"
---

# config

53 symbols | 2 files | 98% cohesion

## When to Use

Use this skill when working on files in:
- `invoice-service/internal/config/config.go`
- `invoice-service/internal/config/config_test.go`

## Key Files

| File | Symbols |
|------|---------|
| `invoice-service/internal/config/config.go` | OCRServiceURL, fallback, DatabasePath, WebAuthnRPID, SMTPPort, ... |
| `invoice-service/internal/config/config_test.go` | TestOCREnabled, mustSetenv, t, key, t, ... |

## How to Explore

```
get_communities with id: "community-9"
smart_context with task: "understand config", format: "gcx"
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
