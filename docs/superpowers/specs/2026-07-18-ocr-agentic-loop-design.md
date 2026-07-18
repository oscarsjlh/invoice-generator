# OCR Agentic Loop Design

## Status

Proposed — awaiting implementation planning.

## Context

The invoice app has an optional OCR import flow. Users photograph handwritten timesheets, upload them to the Go web app, and the app calls a small Python OCR service. The OCR service sends the images to AWS Bedrock and returns structured draft entries that the user reviews before confirming.

Today the OCR service does a **single-shot** extraction:

1. Resize/normalize the image.
2. Send one prompt asking for pipe-separated lines: `DATE | CATEGORY | HOURS | NOTES`.
3. Parse the response with simple string splitting.

This has caused real problems:

- **iPhone/HEIC pictures** can exceed Bedrock request-body limits because they were not resized before base64 encoding.
- **Category drift**: the model receives the known categories as plain text but still hallucinates categories or confuses special cases like Sick/Holiday.
- **Manual review burden**: the user spends time fixing entries that the model could have gotten right with a second, focused pass.

## Goals

Replace the single-shot extraction with a small, explicit agent loop inside the OCR service that:

1. Extracts structured entries with a constrained category enum.
2. Validates every entry against the user’s fixed category list and basic business rules.
3. Re-prompts Bedrock only for flagged entries, with a focused correction prompt.
4. Finalizes results and marks remaining ambiguous entries for human review instead of silently guessing.

The Go ↔ OCR HTTP boundary must stay unchanged so the rest of the app is unaffected.

## Non-goals

- Do not introduce a heavy agent framework (LangChain, LangGraph, CrewAI, Pydantic AI). The loop is implemented in plain Python with Pydantic for typed models and validation.
- Do not change the upload UI or review UI beyond showing the new `review_reason` field.
- Do not perform cross-page reconciliation or multi-image reasoning in this iteration.
- Do not add autonomous decision-making that confirms entries without human approval.

## Design

### Boundary

```text
[Go invoice-service]  --multipart/image+json-->  [ocr-service]
                                                       |
                                                agent loop
                                                       |
                                                   Bedrock
```

The Go service still calls the OCR service once per image. The agent loop is internal to `ocr-service`.

### Data models

Introduce `pydantic` as a dependency in `ocr-service`.

```python
class ExtractedEntry(BaseModel):
    date_raw: str
    date_normalized: str           # YYYY-MM-DD
    category_raw: str
    category_normalized: str       # must match a known category
    hours_raw: str
    hours_normalized: str          # positive float as string
    notes_raw: str
    notes_normalized: str
    confidence: float              # 0.0 - 1.0
    needs_review: bool = False
    review_reason: str | None = None

class PageExtraction(BaseModel):
    entries: list[ExtractedEntry]

class OCRContext(BaseModel):
    categories: list[str]
    rates: list[RateHint]
    current_year: int
    page_number: int

class ValidationResult(BaseModel):
    valid: list[ExtractedEntry]
    flagged: list[ExtractedEntry]
```

### API changes

The OCR service request/response format is unchanged except for one new optional field.

**Request:** same multipart form with `request` JSON field containing `session_id`, `hints.categories`, `hints.send_rates`, `rates`, and `current_year`.

**Response:** same shape as today, but each entry may now include `review_reason`. The Go contract is extended:

```go
type OCRExtractedEntry struct {
    Date     OCRExtractedField
    Category OCRExtractedField
    Hours    OCRExtractedField
    Notes    OCRExtractedField
    ReviewReason string  // NEW
}
```

`ReviewReason` is stored on the draft entry and can be shown in the review table.

### Agent loop

```text
extract(image, context)
    -> PageExtraction
validate(page, context)
    -> ValidationResult
if flagged and attempt < MAX_ATTEMPTS:
    correct(image, context, flagged)
        -> PageExtraction
    validate again
finalize(valid + still-flagged)
    -> PageExtraction
```

- `MAX_ATTEMPTS` is 2: one initial extraction and one correction pass.
- Flagged entries that remain flagged after the correction pass are kept but marked `needs_review=True` with their reason.
- Each Bedrock call is wrapped in its own OpenTelemetry span.

### Prompts

#### Extraction prompt

- Role: expert at reading handwritten timesheets.
- Task: extract entries and return JSON matching `PageExtraction`.
- Allowed category enum (exact names from `context.categories`).
- Current-year inference rule.
- Examples: a normal entry, a Sick/Holiday entry, and an unreadable entry marked for review.
- Strict instruction: if a field cannot be read, set `needs_review=True`, set the raw value to `"?"`, and leave normalized values empty or best-effort.

#### Correction prompt

- Includes the flagged entries as JSON.
- Repeats the allowed category enum.
- Focused instruction: “Fix only these entries. Choose the closest known category, or mark as needs review if still unreadable.”

### Validation rules

For each extracted entry:

1. `category_normalized` must match a known category case-insensitively. If not, reason = `unknown_category`.
2. `date_normalized` must parse as `YYYY-MM-DD`. If not, reason = `invalid_date`.
3. `hours_normalized` must be a positive float. If not, reason = `invalid_hours`.
4. `confidence < 0.5` → reason = `low_confidence`.
5. Any raw field equal to `"?"` → reason = `unreadable_field`.
6. (Optional) If rates are provided and no rate exists for the category+date, reason = `missing_rate`.

An entry can accumulate multiple reasons; they are joined with `; `.

### Image preprocessing

Keep the existing `prepare_image_for_bedrock` behavior:

- Open with Pillow (HEIC supported via `pillow-heif`).
- Convert to RGB.
- Resize so the longest side is at most `OCR_MAX_IMAGE_DIMENSION` (default 2048).
- Align dimensions to 32px.
- Encode as JPEG quality 85.
- Return base64.

Add EXIF orientation correction:

- Apply `ImageOps.exif_transpose(img)` before resizing. This fixes sideways iPhone photos.

### Error handling

- **Image open/preprocess failure:** fail the session (existing behavior).
- **Bedrock API failure:** fail the session (existing behavior).
- **JSON parse failure or Pydantic validation failure:** treat as a flag. The first failure triggers the correction pass. A persistent parse failure marks the whole page with `needs_review` and reason `parser_error`.
- **Timeout:** the existing 120s Go client timeout covers all Bedrock calls in the loop.

### Observability

New OpenTelemetry span attributes:

- `ocr.extract.attempt` — 0-indexed attempt number.
- `ocr.validate.flagged_count` — number of entries flagged in each validation.
- `ocr.correct.attempt` — correction pass number.
- `ocr.agent.rounds` — total rounds used for the page.
- `ocr.request_body_bytes` — already present, continues to be logged.

Logs must never include image bytes, prompts containing personal data, or AWS credentials.

## Testing

Add `ocr-service/test_main.py` with a stub Bedrock backend.

Test cases:

1. Valid page → all entries pass validation.
2. Unknown category → re-prompt corrects it to a known category.
3. Invalid date → flagged with `invalid_date`.
4. Invalid hours → flagged with `invalid_hours`.
5. Sick/Holiday category → recognized and normalized correctly.
6. Low confidence → flagged with `low_confidence`.
7. Malformed JSON from model → handled gracefully, falls back to needs_review.
8. HEIC image → converted to JPEG and resized.
9. EXIF-rotated JPEG → corrected before extraction.

The Go handler tests with the stub extractor continue to work unchanged.

## Rollout

1. Implement the agent loop behind an `OCR_AGENT_ENABLED` env var, default `true` once stable.
2. Keep the legacy `parse_handwritten_text` path as a fallback for the first release.
3. Log metrics: rounds per page, flag counts, final review count.
4. After a short soak period, remove the legacy fallback in a follow-up change.

## Files changed

- `ocr-service/pyproject.toml` — add `pydantic` dependency.
- `ocr-service/main.py` — refactor extraction into the agent loop, add models, prompts, validators.
- `ocr-service/test_main.py` — new unit tests.
- `invoice-service/internal/ocr/contract.go` — add `ReviewReason` to `OCRExtractedField`.
- `invoice-service/internal/ocrimport/importer.go` — propagate `ReviewReason` into `db.OCRDraftEntry`.
- `invoice-service/templates/ocr_review.html` — optionally show `review_reason` in the review table.

## Open questions

1. Should `review_reason` be shown in the review table as a tooltip, badge, or extra column?
badge
2. Should the correction pass also receive the original image again, or only the flagged entries?
flagged entries
3. Should we cap the number of entries sent to the correction pass to avoid oversized prompts?
yes
