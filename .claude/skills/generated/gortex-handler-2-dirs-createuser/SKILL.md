---
name: gortex-handler-2-dirs-createuser
description: "Work in the handler +2 dirs · CreateUser area — 70 symbols across 6 files (78% cohesion)"
---

# handler +2 dirs · CreateUser

70 symbols | 6 files | 78% cohesion

## When to Use

Use this skill when working on files in:
- `invoice-service/internal/auth/webauthn.go`
- `invoice-service/internal/db/authdb.go`
- `invoice-service/internal/db/authdb_test.go`
- `invoice-service/internal/handler/auth_handlers_test.go`
- `invoice-service/internal/handler/logger.go`
- `invoice-service/internal/handler/middleware_test.go`

## Key Files

| File | Symbols |
|------|---------|
| `invoice-service/internal/auth/webauthn.go` | r, sessionID, FinishRegistration |
| `invoice-service/internal/db/authdb.go` | GetUserByUsernameForAuth, userID, id, c, GetUserByID, ... |
| `invoice-service/internal/db/authdb_test.go` | t, setupTestAuthDB, t, TestAuthDBCreateAndValidateSession, TestAuthDBValidateExpiredSession, ... |
| `invoice-service/internal/handler/auth_handlers_test.go` | t, TestLoginPageRedirectsIfAlreadyLoggedIn, t, TestRegisterPageRedirectsIfAlreadyLoggedIn, t, ... |
| `invoice-service/internal/handler/logger.go` | Flush |
| `invoice-service/internal/handler/middleware_test.go` | t, closure@295, TestAuthMiddlewareValidSessionPassesThrough |

## Entry Points

- `invoice-service/internal/handler/middleware_test.go::TestAuthMiddlewareValidSessionPassesThrough`
- `invoice-service/internal/handler/auth_handlers_test.go::TestLogoutRedirects`

## Connected Communities

- **db +2 dirs** (7 cross-edges)
- **handler · LoggerFromContext** (5 cross-edges)
- **handler · newTestAppWithAuth** (4 cross-edges)
- **handler +2 dirs · authMiddleware** (4 cross-edges)
- **db +3 dirs** (2 cross-edges)
- **handler · WriteHeader** (1 cross-edges)
- **handler +4 dirs** (1 cross-edges)
- **auth +1 dirs** (1 cross-edges)

## How to Explore

```
get_communities with id: "community-7"
smart_context with task: "understand handler +2 dirs · CreateUser", format: "gcx"
find_usages with id: "invoice-service/internal/handler/middleware_test.go::TestAuthMiddlewareValidSessionPassesThrough", format: "gcx"
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
