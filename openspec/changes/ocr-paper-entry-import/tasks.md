## 1. OCR Runtime Research And Service Contract

- [x] 1.1 Evaluate `olmocr.allenai.org` and its related runtime options to determine whether the OCR stack can be self-hosted, can use a local LLM, or requires a hosted deployment
- [x] 1.2 Document the chosen OCR runtime's hardware profile, including GPU and VRAM requirements or the trade-offs of any CPU or reduced local mode
- [x] 1.3 Define the HTTP contract for the separate OCR service, including image upload payloads, contextual category/rate hints, import status, raw text, normalized candidates, and confidence fields
- [x] 1.4 Add configuration settings for enabling the OCR workflow and pointing the Go app at the OCR service endpoint

## 2. Persistence And Import Session Modeling

- [x] 2.1 Add database schema for OCR import sessions, uploaded image metadata, extracted draft entries, confidence markers, and processing status
- [x] 2.2 Implement store methods to create import sessions, save OCR draft results, list pending review items, mark failures, and confirm reviewed drafts into `entries`
- [x] 2.3 Define retention and cleanup behavior for uploaded images and failed or abandoned OCR sessions

## 3. App Integration And Review Workflow

- [x] 3.1 Add entry-page routes and handlers for starting an OCR import session, polling or viewing processing status, and loading the review screen
- [x] 3.2 Implement the OCR client integration in the Go app so requests include known categories/rates and responses preserve both raw OCR text and normalized suggestions
- [x] 3.3 Build templates and HTMX partials for image upload, draft entry review, ambiguity/confidence display, manual correction, and confirmation into final entries
- [x] 3.4 Handle OCR service failures and unsupported uploads with recoverable user-facing errors that preserve the import session state

## 4. Image Quality And Rollout Validation

- [x] 4.1 Add image preprocessing or normalization steps needed for the sample notebook photos, such as rotation correction, cropping, or contrast improvement
- [x] 4.2 Validate the workflow against `PXL_20260517_080526792.jpg` and `PXL_20260517_080527543.jpg`, recording where rate-based hints improve recognition and where manual review is still required
- [x] 4.3 Gate the feature behind configuration so the existing manual entry flow remains the fallback during rollout
- [x] 4.4 Update project documentation with OCR setup, deployment expectations, and operator guidance for local-vs-hosted runtime decisions
