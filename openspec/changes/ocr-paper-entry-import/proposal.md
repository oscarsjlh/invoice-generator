## Why

Entering handwritten timesheet entries by hand is slow and error-prone, especially when the user already has paper records they can photograph. The app needs a way to ingest scanned notes into draft entries while evaluating whether `olmocr.allenai.org` can run as a local service or, if not, documenting its VRAM and deployment requirements.

## What Changes

- Add a paper-entry OCR ingestion workflow that accepts photographed handwritten entry sheets and returns draft invoice entries for review before save.
- Introduce a separate OCR service integration layer that can call OLMOCR-compatible processing and record whether it runs locally, requires a remote service, or needs a different model/runtime.
- Use existing rate/category data to improve OCR interpretation by biasing category recognition and surfacing confidence when handwriting does not map cleanly to known rates.
- Support the sample JPEG-based workflow already represented by `PXL_20260517_080526792.jpg` and `PXL_20260517_080527543.jpg` so the feature is grounded in real input quality.
- Document hardware assumptions, including local LLM feasibility and VRAM requirements, as part of the delivered service design.

## Capabilities

### New Capabilities
- `paper-entry-ocr-import`: Upload paper-entry photos, extract draft entries, and present them for user review before persistence.
- `ocr-service-runtime-assessment`: Determine whether the OCR stack can run locally with a local LLM and capture deployment or VRAM requirements when it cannot.

### Modified Capabilities

None.

## Impact

- Affected code: entry handlers, templates for entry creation/review, store methods for draft-to-entry persistence, and new OCR integration code.
- Affected systems: external or local OCR runtime, image upload/storage pipeline, and rate lookup logic used for OCR post-processing.
- Dependencies: OLMOCR investigation, possible new OCR or LLM runtime dependencies, image parsing support, and operational documentation for model hosting requirements.
