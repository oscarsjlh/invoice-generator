---
name: gortex-handler-writeheader
description: "Work in the handler · WriteHeader area — 43 symbols across 5 files (88% cohesion)"
---

# handler · WriteHeader

43 symbols | 5 files | 88% cohesion

## When to Use

Use this skill when working on files in:
- `invoice-service/internal/handler/app.go`
- `invoice-service/internal/handler/logger.go`
- `invoice-service/internal/handler/middleware_csrf.go`
- `invoice-service/internal/handler/middleware_test.go`
- `invoice-service/internal/handler/renderer.go`

## Key Files

| File | Symbols |
|------|---------|
| `invoice-service/internal/handler/app.go` | isPublicPath, path |
| `invoice-service/internal/handler/logger.go` | code, WriteHeader |
| `invoice-service/internal/handler/middleware_csrf.go` | isSafeMethod, closure@8, method, CSRFMiddleware |
| `invoice-service/internal/handler/middleware_test.go` | t, t, t, closure@124, TestCSRFMiddlewarePOSTWithMismatchedToken, ... |
| `invoice-service/internal/handler/renderer.go` | status, data, name, files, w, ... |

## Entry Points

- `invoice-service/internal/handler/middleware_test.go::TestIsPublicPath`

## Connected Communities

- **handler · extractCSRFToken** (1 cross-edges)

## How to Explore

```
get_communities with id: "community-53"
smart_context with task: "understand handler · WriteHeader", format: "gcx"
find_usages with id: "invoice-service/internal/handler/middleware_test.go::TestIsPublicPath", format: "gcx"
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
