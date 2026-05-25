---
name: gortex-handler-2-dirs-authmiddleware
description: "Work in the handler +2 dirs · authMiddleware area — 48 symbols across 7 files (77% cohesion)"
---

# handler +2 dirs · authMiddleware

48 symbols | 7 files | 77% cohesion

## When to Use

Use this skill when working on files in:
- `invoice-service/internal/auth/auth_test.go`
- `invoice-service/internal/auth/session.go`
- `invoice-service/internal/handler/app.go`
- `invoice-service/internal/handler/csrf_cookie.go`
- `invoice-service/internal/handler/store_provider.go`
- `invoice-service/internal/handler/traced_store.go`
- `invoice-service/internal/testutil/helpers.go`

## Key Files

| File | Symbols |
|------|---------|
| `invoice-service/internal/auth/auth_test.go` | closure@212, closure@225, TestSessionManagerGetUserFromRequestNoCookie, TestSessionManagerIsSecure, t, ... |
| `invoice-service/internal/auth/session.go` | r, authDB, ttl, authDB, GetUserFromRequest, ... |
| `invoice-service/internal/handler/app.go` | authMiddleware, next, closure@217 |
| `invoice-service/internal/handler/csrf_cookie.go` | w, r, secure, ensureCSRFCookie |
| `invoice-service/internal/handler/store_provider.go` | r, user, ForRequest |
| `invoice-service/internal/handler/traced_store.go` | ctx, next, newTracedStore |
| `invoice-service/internal/testutil/helpers.go` | t, closure@92, NewTestAuthDB |

## Entry Points

- `invoice-service/internal/auth/auth_test.go::TestSessionManagerCreateAndDestroy`

## Connected Communities

- **handler +2 dirs · CreateUser** (3 cross-edges)
- **handler · WriteHeader** (2 cross-edges)
- **db +3 dirs** (2 cross-edges)
- **testutil** (1 cross-edges)
- **handler +4 dirs** (1 cross-edges)
- **handler · process** (1 cross-edges)
- **handler · LoggerFromContext** (1 cross-edges)

## How to Explore

```
get_communities with id: "community-6"
smart_context with task: "understand handler +2 dirs · authMiddleware", format: "gcx"
find_usages with id: "invoice-service/internal/auth/auth_test.go::TestSessionManagerCreateAndDestroy", format: "gcx"
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
