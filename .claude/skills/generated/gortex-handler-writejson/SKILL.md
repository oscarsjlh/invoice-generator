---
name: gortex-handler-writejson
description: "Work in the handler · writeJSON area — 29 symbols across 3 files (78% cohesion)"
---

# handler · writeJSON

29 symbols | 3 files | 78% cohesion

## When to Use

Use this skill when working on files in:
- `invoice-service/internal/handler/auth_handlers.go`
- `invoice-service/internal/handler/auth_handlers_test.go`
- `invoice-service/internal/handler/logger.go`

## Key Files

| File | Symbols |
|------|---------|
| `invoice-service/internal/handler/auth_handlers.go` | data, finishRegistration, beginRegistration, beginLoginRequest, finishLogin, ... |
| `invoice-service/internal/handler/auth_handlers_test.go` | TestBeginRegistrationSuccess, t, TestFinishLoginInvalidSession, TestFinishRegistrationInvalidSession, t, ... |
| `invoice-service/internal/handler/logger.go` | value, RedactEmail |

## Connected Communities

- **handler · LoggerFromContext** (4 cross-edges)
- **handler · newTestAppWithAuth** (3 cross-edges)
- **handler · Close** (2 cross-edges)
- **handler · WriteHeader** (1 cross-edges)

## How to Explore

```
get_communities with id: "community-37"
smart_context with task: "understand handler · writeJSON", format: "gcx"
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
