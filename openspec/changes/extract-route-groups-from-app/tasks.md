## 1. Extract AuthHandlers group

- [x] 1.1 Create `AuthHandlers` struct with fields for `*auth.WebAuthnManager`, `*auth.SessionCookie`, `*Renderer`, and `*AuthRateLimiter`
- [x] 1.2 Write `NewAuthHandlers(...)` constructor accepting only the dependencies auth handlers use
- [x] 1.3 Move 7 route handler methods (`loginPage`, `registerPage`, `beginLogin`, `finishLogin`, `beginRegistration`, `finishRegistration`, `logout`, `usernamePolicyHandler`) from `App` to `AuthHandlers`
- [x] 1.4 Write `AuthHandlers.Routes(mux *http.ServeMux)` registering all auth routes with conditional registration
- [x] 1.5 Pass `authEnabled` and `registrationEnabled` as constructor params (not config.Config)
- [x] 1.6 Move notice constants and helper functions to auth_handlers.go
- [x] 1.7 Update `auth_handlers_test.go` — added `ah *AuthHandlers` to test struct; tests updated
- [x] 1.8 Run `go test ./internal/handler/...` — all auth handler tests pass

## 2. Extract EntryHandlers group

- [x] 2.1 Create `EntryHandlers` struct with `*Renderer` field
- [x] 2.2 Move 6 route handler methods from `App` to `EntryHandlers`
- [x] 2.3 Write `EntryHandlers.Routes(mux)` registering all entry routes
- [x] 2.4 Update `entries_test.go` — added `eh *EntryHandlers` to `testApp`, tests updated
- [x] 2.5 Run `go test ./internal/handler/... -run Entry` — all entry handler tests pass

## 3. Extract RateHandlers group

- [x] 3.1 Create `RateHandlers` struct with `*Renderer` field
- [x] 3.2 Move 6 route handler methods from `App` to `RateHandlers`
- [x] 3.3 Write `RateHandlers.Routes(mux)` registering all rate routes
- [x] 3.4 Update `rates_test.go` — tests updated to use `ta.rh`
- [x] 3.5 Run `go test ./internal/handler/...` — all rate handler tests pass

## 4. Extract InvoiceHandlers group

- [x] 4.1 Create `InvoiceHandlers` struct with `*Renderer` and `config.Config`
- [x] 4.2 Move 5 route handler methods from `App` to `InvoiceHandlers`
- [x] 4.3 Write `InvoiceHandlers.Routes(mux)` registering all invoice routes
- [x] 4.4 Update `invoices_test.go` — tests updated to use `ta.ih`
- [x] 4.5 Run `go test ./internal/handler/...` — all invoice handler tests pass

## 5. Extract SettingsHandlers group

- [x] 5.1 Create `SettingsHandlers` struct with `*Renderer` field
- [x] 5.2 Move 2 route handler methods from `App` to `SettingsHandlers`
- [x] 5.3 Write `SettingsHandlers.Routes(mux)` registering settings routes
- [x] 5.4 Update `settings_test.go` — tests updated to use `ta.sh`
- [x] 5.5 Run `go test ./internal/handler/...` — all settings handler tests pass

## 6. Extract OCRHandlers group

- [x] 6.1 Create `OCRHandlers` struct with `*Renderer`, `*OCRJobRunner`, `config.Config`
- [x] 6.2 Move 5 route handler methods + `ocrImporter` + `mergeDraftCategories` from `App` to `OCRHandlers`
- [x] 6.3 Write `OCRHandlers.Routes(mux)` registering all OCR routes
- [x] 6.4 Update `ocr_test.go` — added `oh *OCRHandlers` to test structs, tests updated
- [x] 6.5 Run `go test ./internal/handler/...` — all OCR handler tests pass

## 7. Shrink App to wiring coordinator

- [x] 7.1 Removed all moved handler methods from `App`. App keeps: `authMiddleware`, `isPublicPath`, `health`, `dashboard`, `renderPage`/`renderPartial` (delegated), `redirect` (now package-level), `noticeFromRequest` (package-level), helper functions
- [x] 7.2 `App.Routes()` creates group instances, calls each group's `Routes(mux)`, applies middleware
- [x] 7.3 App fields retained — all are still used by middleware or group wiring
- [x] 7.4 Removed `StoreForTest()` and `SetOCRClient()` test helper methods (tests now construct groups directly)
- [x] 7.5 `cmd/server/main.go` unchanged — still passes deps to `handler.New()` which wires groups internally
- [x] 7.6 `integration_test.go` unchanged — uses `app.Routes()` which creates groups internally

## 8. Final verification

- [x] 8.1 Run `go test ./internal/handler/...` — all tests pass
- [x] 8.2 Run `go test ./...` — full test suite passes (all 9 packages)
- [x] 8.3 Run `go vet ./...` — no vet warnings
- [ ] 8.4 Run `golangci-lint run ./...` (if configured) — not configured
- [ ] 8.5 Manual smoke test: start server, log in with WebAuthn, navigate dashboard/entries/rates/invoices/settings/OCR — all routes work
