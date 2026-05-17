## Context

The current app only supports manual entry creation through the `/entries` form and direct persistence into SQLite. The new change crosses UI, server handlers, OCR integration, and operational research because the user wants a separate service that can ingest photographed handwritten timesheets, evaluate `https://olmocr.allenai.org/` as the OCR foundation, and determine whether that stack can run locally with a local LLM or otherwise document VRAM requirements.

The provided JPEG samples show non-ideal capture conditions: rotated pages, a spiral notebook seam, mixed ink colors, and handwritten category names with circled hour values. This means the system cannot treat OCR output as authoritative. Existing rates and categories are the best local source of truth for post-processing because they constrain the likely category vocabulary and can help resolve OCR outputs like `Acro 4` vs `Acro4`, `Reception`, or abbreviated handwritten labels.

## Goals / Non-Goals

**Goals:**
- Add a paper-entry import path that accepts one or more photographed pages and produces draft entries  and creates a table of possible entries, that the end user can check and make changes if required of immediately creating final rows.
- Keep the OCR engine behind a separate service boundary so the Go app remains a thin orchestrator and can switch between hosted OLMOCR, a local runtime, or a fallback implementation.
- Use stored rates/categories as contextual hints during OCR normalization and matching.
- Record OCR confidence, ambiguous fields, and runtime metadata so the user can review and correct results.
- Document whether OLMOCR can be self-hosted with a local LLM and, if self-hosting is impractical, capture the minimum runtime and VRAM expectations needed to deploy it.

**Non-Goals:**
- Fully automatic entry creation without user review.
- Solving arbitrary document OCR beyond handwritten work-log style pages.
- Replacing the existing manual entry flow.
- Building a generalized model-training pipeline for handwriting fine-tuning.

## Decisions

### Decision: Add a separate OCR service instead of embedding OCR directly in the Go web app

The Go app should call a dedicated OCR service over HTTP, passing uploaded images plus contextual metadata such as known categories/rates. This isolates heavy OCR dependencies, model runtime setup, and hardware constraints from the single-binary web app.

Why this approach:
- The current app is intentionally simple and single-binary; model runtimes would complicate deploys and local development.
- OLMOCR feasibility is uncertain. A service boundary lets the app remain stable whether the service ends up using hosted OLMOCR, a local model, or a hybrid flow.
- Separate deployment makes it easier to scale GPU-backed OCR independently from the main app.

Alternatives considered:
- Embed OCR into the Go server process: rejected because it couples GPU/model lifecycle to the web app and increases operational risk.
- Client-side OCR in the browser: rejected because the sample images and handwriting quality require stronger models and server-side normalization.

### Decision: Produce draft OCR results that require review before persistence

OCR results should be stored as import sessions with candidate entries and confidence metadata. The user then reviews, edits, and confirms them into real `entries` rows.

Why this approach:
- The sample pages contain ambiguous handwriting and page orientation issues.
- Existing app behavior expects valid normalized dates, categories, and positive hours; draft review prevents low-confidence OCR from polluting the ledger.

Alternatives considered:
- Auto-save directly to `entries`: rejected because recognition errors would be hard to detect and could affect invoice generation.
- Keep results only in-memory: rejected because image processing may be slow and users need to revisit drafts after OCR finishes.

### Decision: Use rate/category data as OCR normalization hints, not as a hard validator

The app should send the list of known categories, associated rates, and likely aliases to the OCR service or normalization layer. The service uses them to rank candidate category matches, but still returns unmatched text when confidence is low.

Why this approach:
- The user explicitly wants rates to help decipher handwriting.
- Categories are the strongest structured vocabulary already available in the system.
- A hard validator would incorrectly force mismatches when handwriting introduces a new or mistyped category.

Alternatives considered:
- Ignore rates/categories during OCR: rejected because it loses obvious domain context.
- Force every recognized category to an existing rate: rejected because it hides uncertainty and could create incorrect billable entries.

### Decision: Capture runtime-assessment findings as part of the service contract and project docs

The OCR service design includes a runtime assessment output covering deployment mode, model/backend choice, memory expectations, and whether local LLM support is feasible. This is treated as a first-class deliverable rather than an informal note.

Why this approach:
- The user asked for both feasibility and VRAM requirements, not just feature behavior.
- Implementation choices depend on whether the OCR stack can run on local hardware.

Alternatives considered:
- Defer runtime research until after coding: rejected because it could invalidate the service architecture.

## Risks / Trade-offs

- [OLMOCR may not support local handwritten OCR or local LLM execution well] -> Mitigation: keep the app talking to an abstract OCR service interface and document fallback engines or hosted mode requirements.
- [Image uploads increase storage and privacy sensitivity] -> Mitigation: treat uploaded images as temporary import-session assets with explicit retention and deletion rules.
- [Category biasing may over-correct valid uncommon handwriting] -> Mitigation: preserve raw OCR text and expose confidence plus suggested match separately.
- [Asynchronous OCR adds workflow complexity] -> Mitigation: model import sessions with clear states such as uploaded, processing, review-ready, failed.
- [GPU/VRAM needs may exceed likely deployment targets] -> Mitigation: include a no-GPU or hosted fallback path in the runtime assessment and keep the service separately deployable.

## Migration Plan

1. Add database support for OCR import sessions, uploaded image references, draft extracted entries, and review status.
2. Introduce the OCR service contract and a stub/mock implementation so the UI flow can be built before final model integration.
3. Add entry-import UI for image upload, processing status, and draft review/confirmation.
4. Integrate the chosen OCR runtime and complete the local-vs-hosted runtime assessment.
5. Roll out behind a configuration flag so manual entry remains the default fallback during early validation.

Rollback strategy:
- Disable the OCR import routes via configuration and leave the existing manual entry workflow unchanged.
- Preserve imported draft data for audit/debugging, but stop exposing the feature in the UI.

## Open Questions

- Does `olmocr.allenai.org` expose a supported self-hosted path for this handwriting-heavy workflow, or is a different OCR backend required for acceptable accuracy?
- What image preprocessing is necessary for the sample notebook pages: rotation correction, cropping, contrast enhancement, or page splitting?
- Should uploaded images be stored in SQLite-backed metadata plus filesystem blobs, or only on disk with path references?
- How much of the category hinting should happen in the Go app versus inside the OCR service prompt or post-processing layer?
- What confidence threshold should require mandatory manual correction before an entry can be confirmed?
