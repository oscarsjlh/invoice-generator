## Context

The username validation policy — regex `^[a-z0-9][a-z0-9._-]{2,31}$` and the message "Use 3-32 characters: lowercase letters, numbers, dots, underscores, or hyphens. Start with a letter or number." — is defined in two places:

- `internal/auth/username.go:8-10` (Go, server-side)
- `static/auth.js:2-3` (JavaScript, client-side)

Both perform identical validation. The server is authoritative (client-side validation is UX, not security), but neither module "owns" the policy — changing it requires manual sync across languages and a network boundary.

The app uses no build step (no Vite, no Webpack, no TypeScript), so there is no opportunity to share a source-of-truth file at build time.

## Goals / Non-Goals

**Goals:**
- The server is the single source of truth for the username validation policy
- The client fetches the policy from the server and applies it for instant feedback
- The client gracefully degrades if the fetch fails (uses hardcoded fallback values)
- The policy endpoint is public (no auth required) since it's used on the login/register pages
- Existing server-side `ValidateUsername` and `NormalizeUsername` functions are unchanged

**Non-Goals:**
- Changing the validation rules themselves (same regex, same message)
- Adding a build step or code generation to sync the policy
- Supporting runtime policy updates without page reload (policy is fetched once per page load)
- Exposing any other auth configuration via the endpoint

## Decisions

**Decision 1: Publish the policy as a JSON endpoint, not embedded in an HTML template**

Rationale: A JSON endpoint is consumable by the client JS and by potential future API consumers. Embedding the policy in the template as a `<script>` variable would couple the template to the client JS implementation. The endpoint keeps the seam clean: the server publishes the policy; any client can consume it.

Alternative considered: Embed the policy in `login.html` / `register.html` via a `<script>` tag with `window.__usernamePolicy = {...}`. Rejected because it couples templates to JS implementation and doesn't serve non-browser consumers.

**Decision 2: Fetch the policy lazily on first form interaction, not at page load**

Rationale: The username input is the first thing the user interacts with on the login/register page. Fetching on page load would add a blocking request to every page visit. Fetching on first `submit` or `input` event defers the request until it's needed, keeping the initial page render fast.

Alternative considered: Fetch at page load via `DOMContentLoaded`. Rejected because it adds a request to every visitor, including those who bounce before interacting with the form. The lazy fetch happens at most once per page visit.

**Decision 3: Client caches the policy in a module-level variable, fetched once**

The JS module stores the result of the first successful fetch. Subsequent `validatedUsername` calls use the cached policy. If the fetch fails, the module falls back to the current hardcoded regex and message (which are identical to the server's values). This ensures the page works even if the network is slow or the endpoint is temporarily unavailable.

**Decision 4: The endpoint returns the regex as a string, not a compiled regex**

The server sends the regex pattern as a JSON string. The client compiles it into a `RegExp` locally. This avoids the complexity of sending pre-compiled regex flags across the wire. The normalization rules are described in plain English (the client already implements `toLowerCase` + `trim`).

## Risks / Trade-offs

**[Risk] Additional HTTP request on first form interaction** — the policy fetch adds ~50ms of latency before the first validation runs.
→ **Mitigation**: The endpoint response is tiny (~100 bytes JSON). The client can pre-fetch on `focus` of the username input to hide latency. If the request fails, the fallback values provide equivalent validation instantly.

**[Risk] Client and server validation drift if the endpoint changes but the client hasn't re-fetched** — a user could have a stale page open after a deploy.
→ **Mitigation**: The policy change is extremely rare (the regex hasn't changed since it was introduced). If it does change, the server-side validation is authoritative — the client is only for UX feedback. A stale client still produces a valid username, or an invalid one that the server rejects with the correct message.

**[Trade-off] Added complexity for a small DRY win** — the duplicated code is 2 lines of regex + 1 line of message.
→ Acceptable: the real value is establishing the pattern. If the app later adds email validation, password rules, or other cross-seam policies, the infrastructure exists.
