---
name: gortex-auth-newwebauthnmanager
description: "Work in the auth · NewWebAuthnManager area — 38 symbols across 3 files (78% cohesion)"
---

# auth · NewWebAuthnManager

38 symbols | 3 files | 78% cohesion

## When to Use

Use this skill when working on files in:
- `invoice-service/internal/auth/auth_test.go`
- `invoice-service/internal/auth/models.go`
- `invoice-service/internal/auth/webauthn.go`

## Key Files

| File | Symbols |
|------|---------|
| `invoice-service/internal/auth/auth_test.go` | TestFinishLoginInvalidSession, TestBeginRegistrationUsernameTaken, t, TestBeginLoginUserNotFound, t, ... |
| `invoice-service/internal/auth/models.go` | RPDisplayName, SessionTTL, AuthConfig, RPID, RPOrigins |
| `invoice-service/internal/auth/webauthn.go` | UserID, mu, BeginRegistration, pendingSession, WebAuthnManager, ... |

## Connected Communities

- **handler +2 dirs · CreateUser** (9 cross-edges)
- **handler +2 dirs · authMiddleware** (4 cross-edges)
- **auth +1 dirs** (3 cross-edges)
- **db +2 dirs** (1 cross-edges)

## How to Explore

```
get_communities with id: "community-8"
smart_context with task: "understand auth · NewWebAuthnManager", format: "gcx"
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
