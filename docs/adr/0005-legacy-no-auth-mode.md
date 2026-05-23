# ADR 0005: Legacy No-Auth Mode

## Status

Accepted

## Context

The application previously supported a single local invoice database without authentication. Existing deployments may still rely on this mode.

## Decision

Keep legacy no-auth mode as a compatibility adapter that maps every request to one shared store and an anonymous user identity.

## Consequences

Legacy mode should be hidden behind the same store-provider seam as authenticated per-user mode. Handlers should not branch on legacy mode or know whether the current store is shared or per-user.
