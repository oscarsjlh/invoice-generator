## 1. Server-side policy endpoint

- [x] 1.1 Define `UsernamePolicy` struct in `internal/auth/username.go` with fields: `Pattern string`, `Message string`, `Normalization string`
- [x] 1.2 Add a package-level `var DefaultUsernamePolicy = UsernamePolicy{...}` pre-populated from the existing regex and message constants
- [x] 1.3 Add `usernamePolicyHandler` method on `App` that returns the policy as JSON
- [x] 1.4 Register `GET /auth/username-policy` route — added to both `Routes()` and `isPublicPath`/`isPublicPath` (standalone)
- [x] 1.5 Add test for `GET /auth/username-policy` — `TestUsernamePolicyEndpoint` verifies 200, JSON, correct fields; `TestUsernamePolicyEndpointIsPublic` verifies it passes auth middleware without session
- [x] 1.6 Run `go test ./internal/handler/... -run Policy` — policy endpoint tests pass
- [x] 1.7 Run `go test ./internal/auth/...` — username validation tests still pass

## 2. Client-side policy consumption

- [x] 2.1 In `static/auth.js`, added module-level `let cachedPolicy = null` variable
- [x] 2.2 Added `async function fetchUsernamePolicy()` — fetches `GET /auth/username-policy`, caches the result, falls back to hardcoded `fallbackPattern`/`fallbackMessage` on failure
- [x] 2.3 Updated `validatedUsername(input)` to `async` — calls `fetchUsernamePolicy()` on first invocation, validates against cached policy
- [x] 2.4 Removed hardcoded `const usernameMessage` and `const usernamePattern` from module top-level — replaced with `fallbackPattern`/`fallbackMessage` used only as fallback inside `fetchUsernamePolicy`
- [x] 2.5 `normalizeUsername` unchanged — policy `normalization` field describes same behavior (trim + lowercase)
- [ ] 2.6 Test manually: open login page, open DevTools network tab, verify a single `GET /auth/username-policy` request fires on first form interaction

## 3. Template updates

- [x] 3.1 Verified `login.html` and `register.html` — `<script src="/static/auth.js" defer>` unchanged; JS handles fetch internally
- [x] 3.2 `defer` loading and lazy fetch: IIFE attaches listeners on DOM parse, policy fetched on first `validatedUsername` call (form submit) — no timing issue
- [x] 3.3 No template changes needed

## 4. Verification

- [x] 4.1 Run `go test ./...` — full test suite passes (all 9 packages)
- [ ] 4.2 Manual test: open login page, submit an invalid username, verify the error message matches the server's `UsernameValidationMessage`
- [ ] 4.3 Manual test (regression): enter valid username, complete WebAuthn login — flow works identically
- [ ] 4.4 Manual test (fallback): block the `/auth/username-policy` endpoint in DevTools, verify validation still works with fallback values
- [ ] 4.5 Manual test: open register page, verify username validation works with fetched policy
