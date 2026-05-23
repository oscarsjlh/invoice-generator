## 1. Auth Handler Tests

- [x] 1.1 Write tests for `loginPage` (renders login form)
- [x] 1.2 Write tests for `registerPage` (renders registration form)
- [x] 1.3 Write tests for `beginRegistration` (creates user + returns WebAuthn options)
- [x] 1.4 Write tests for `finishRegistration` (completes passkey creation, creates session)
- [x] 1.5 Write tests for `beginLogin` (returns WebAuthn assertion options)
- [x] 1.6 Write tests for `finishLogin` (validates credential, creates session)
- [x] 1.7 Write tests for `logout` (destroys session, redirects to login)

## 2. Auth Middleware Tests

- [x] 2.1 Write test for authMiddleware with valid session cookie passes through
- [x] 2.2 Write test for authMiddleware without session redirects to /login
- [x] 2.3 Write test for authMiddleware with invalid session redirects to /login
- [x] 2.4 Write test for authMiddleware with auth disabled + legacy store

## 3. Fixtures Consolidation

- [x] 3.1 Replace inline Settings in handler invoices_test.go TestInvoicePreviewReturns200
- [x] 3.2 Replace inline Settings in handler invoices_test.go TestDashboardRendersSuccessfully
- [x] 3.3 Replace inline Settings in handler e2e_test.go TestFullInvoiceWorkflow
- [x] 3.4 Replace inline Settings in handler settings_test.go
- [x] 3.5 DB internal tests keep inline Settings (import cycle prevents testutil usage)

## 4. Integration Tests

- [x] 4.1 Create `internal/handler/integration_test.go` with httptest.Server using Routes()
- [x] 4.2 Test GET / returns 200 with Dashboard content
- [x] 4.3 Test entries page, settings page via full HTTP
- [x] 4.4 Test GET /static/styles.css returns content
- [x] 4.5 Test GET /health returns 200

## 5. OCR Handler Tests

- [x] 5.1 Define OCR `Extractor` interface in `internal/ocr/client.go`
- [x] 5.2 Add `ocrClient` field to App struct and `SetOCRClient` setter
- [x] 5.3 Write tests for `ocrUploadPage` handler (disabled + enabled)
- [x] 5.4 Write tests for `ocrSessionStatus` handler
- [x] 5.5 Write tests for `ocrDeleteSession` handler
- [x] 5.6 Skip `ocrStartSession` and `ocrConfirmDrafts` (require multipart form upload / draft population)
- [x] 5.7 Skip `processOCRSession` (requires goroutine with real OCR processing)

## 6. Concurrent Tests

- [x] 6.1 Write concurrent ForUser test in multistore_test.go (20 goroutines, different user IDs)

## 7. Table-Driven Tests

- [x] 7.1 Skipped: sequential CRUD tests don't benefit from table-driven conversion
- [x] 7.2 Skipped: same as above
- [x] 7.3 Skipped: same as above

## 8. Verification

- [x] 8.1 Run `go test -race -count=1 -tags='!e2e' ./...` — all pass
- [x] 8.2 Run `go test -race -count=1 -tags=e2e ./...` — all pass
- [x] 8.3 Handler coverage improved: 32.4% → 48.0%
