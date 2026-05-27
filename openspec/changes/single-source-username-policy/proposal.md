## Why

The username validation rule (regex `^[a-z0-9][a-z0-9._-]{2,31}$` and its error message) is defined identically in two modules across a network seam: `internal/auth/username.go` (Go) and `static/auth.js` (JavaScript). Neither module owns the rule — changes require manual synchronization. This is a pass-through: the client validates, the server validates, but the policy itself has no single home. Making the server the single source of truth deepens the module, gives the policy a seam with two adapters (server-side validation, client-side fetching), and prevents drift.

## What Changes

- Expose a `GET /auth/username-policy` endpoint returning JSON with the regex pattern, validation message, and normalization rules
- `static/auth.js` fetches the policy on first interaction (or at page load) instead of hardcoding the regex and message
- `internal/auth/username.go` remains the single canonical definition — the endpoint reads from it
- Client maintains a cached copy; validation runs locally with the fetched policy for instant feedback
- `ValidateUsername` and `NormalizeUsername` signatures are unchanged — backward compatible

## Capabilities

### New Capabilities

- `username-policy-endpoint`: A `GET /auth/username-policy` endpoint that returns the canonical username validation policy (regex, message, normalization rules) as JSON, making the server the single source of truth for client-side and server-side validation.

### Modified Capabilities

<!-- None — existing server-side validation behavior is preserved. Client-side behavior is improved (fetches policy) but the validation result is identical. -->

## Impact

- `internal/auth/username.go` — add `UsernamePolicy` struct and export function (minor addition)
- `internal/handler/` — new route handler for `GET /auth/username-policy` (new ~15-line handler)
- `static/auth.js` — replace hardcoded `usernamePattern` and `usernameMessage` consts with a fetch + cache pattern (~20 lines changed)
- `templates/login.html` and `templates/register.html` — need to ensure the policy is loaded before form interaction (may require inline `<script>` or `defer` reordering)
- No breaking changes to server-side validation; client fallback to existing hardcoded values if fetch fails
