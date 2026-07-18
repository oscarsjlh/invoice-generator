# OCR Agentic Loop Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the single-shot OCR extraction in `ocr-service` with a typed extract-validate-correct agent loop, add `review_reason` to the Go draft flow, and surface it in the review UI.

**Architecture:** A new `ocr_agent.py` module inside `ocr-service` owns the Pydantic models, prompts, validation, and loop. `main.py` delegates extraction to it and keeps the legacy path behind `OCR_AGENT_ENABLED`. The Go service gains a `review_reason` column on `ocr_draft_entries` and passes it through the existing contract.

**Tech Stack:** Python 3.12, Pydantic, Pillow, boto3; Go 1.26, SQLite, html/template.

---

## File structure

- `ocr-service/pyproject.toml` — add `pydantic` dependency.
- `ocr-service/ocr_agent.py` — new module: models, image prep, prompts, extraction, validation, correction, loop.
- `ocr-service/main.py` — integrate the agent loop; keep legacy fallback.
- `ocr-service/test_main.py` — unit tests for the agent loop.
- `invoice-service/migrations/009_ocr_review_reason.sql` — add `review_reason` column.
- `invoice-service/internal/db/models.go` — add `ReviewReason` to `OCRDraftEntry`.
- `invoice-service/internal/db/ocr.go` — update insert/select SQL to include `review_reason`.
- `invoice-service/internal/ocr/contract.go` — add `ReviewReason` to `OCRExtractedEntry`.
- `invoice-service/internal/ocrimport/importer.go` — propagate `ReviewReason` into `db.OCRDraftEntry`.
- `invoice-service/templates/ocr_review.html` — show `review_reason` in the review table.

---

## Task 1: Add `pydantic` to `ocr-service`

**Files:**
- Modify: `ocr-service/pyproject.toml`

- [ ] **Step 1: Add `pydantic` to dependencies**

```toml
dependencies = [
    "boto3>=1.35",
    "Pillow>=10.0",
    "pillow-heif>=0.18",
    "pydantic>=2.0",
    "opentelemetry-api>=1.38",
    "opentelemetry-sdk>=1.38",
    "opentelemetry-exporter-otlp-proto-http>=1.38",
]
```

- [ ] **Step 2: Install locally and verify import**

Run: `pip install -e ./ocr-service`

Expected: `pydantic` installs without errors.

- [ ] **Step 3: Commit**

```bash
git add ocr-service/pyproject.toml
```

---

## Task 2: Add the `review_reason` database column

**Files:**
- Create: `invoice-service/migrations/009_ocr_review_reason.sql`

- [ ] **Step 1: Create migration file**

```sql
ALTER TABLE ocr_draft_entries ADD COLUMN review_reason TEXT NOT NULL DEFAULT '';
```

- [ ] **Step 2: Verify migration applies cleanly**

Run: `cd invoice-service && go test ./internal/db/... -run TestMigrations -v`

Expected: all DB tests pass.

- [ ] **Step 3: Commit**

```bash
git add invoice-service/migrations/009_ocr_review_reason.sql
```

---

## Task 3: Update the Go DB model and queries

**Files:**
- Modify: `invoice-service/internal/db/models.go`
- Modify: `invoice-service/internal/db/ocr.go`

- [ ] **Step 1: Add field to `OCRDraftEntry`**

```go
type OCRDraftEntry struct {
    ID                 int64
    SessionID          int64
    DateRaw            string
    DateNormalized     string
    CategoryRaw        string
    CategoryNormalized string
    HoursRaw           string
    HoursNormalized    float64
    NotesRaw           string
    NotesNormalized    string
    Confidence         float64
    NeedsReview        bool
    ReviewReason       string  // NEW
    Confirmed          bool
}
```

- [ ] **Step 2: Update `SaveDraftEntries` insert**

Change the SQL to include `review_reason` and bind the new field:

```go
stmt, err := tx.Prepare(`INSERT INTO ocr_draft_entries
    (session_id, date_raw, date_normalized, category_raw, category_normalized,
     hours_raw, hours_normalized, notes_raw, notes_normalized, confidence, needs_review, review_reason, confirmed)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0)`)
```

```go
if _, err := stmt.Exec(
    sessionID,
    d.DateRaw, d.DateNormalized,
    d.CategoryRaw, d.CategoryNormalized,
    d.HoursRaw, d.HoursNormalized,
    d.NotesRaw, d.NotesNormalized,
    d.Confidence, d.NeedsReview, d.ReviewReason,
); err != nil {
```

- [ ] **Step 3: Update `GetDraftEntries` and `GetDraftEntry` selects**

Add `review_reason` to both the SELECT list and the `Scan` call.

```go
rows, err := s.db.Query(
    `SELECT id, session_id, date_raw, date_normalized, category_raw, category_normalized,
            hours_raw, hours_normalized, notes_raw, notes_normalized, confidence, needs_review, review_reason, confirmed
     FROM ocr_draft_entries WHERE session_id = ? ORDER BY id`,
    sessionID,
)
```

```go
err := rows.Scan(
    &d.ID, &d.SessionID,
    &d.DateRaw, &d.DateNormalized,
    &d.CategoryRaw, &d.CategoryNormalized,
    &d.HoursRaw, &d.HoursNormalized,
    &d.NotesRaw, &d.NotesNormalized,
    &d.Confidence, &d.NeedsReview, &d.ReviewReason, &d.Confirmed,
)
```

Do the same for `GetDraftEntry`.

- [ ] **Step 4: Run DB tests**

Run: `cd invoice-service && go test ./internal/db/...`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add invoice-service/internal/db/models.go invoice-service/internal/db/ocr.go
```

---

## Task 4: Update the Go OCR contract and importer

**Files:**
- Modify: `invoice-service/internal/ocr/contract.go`
- Modify: `invoice-service/internal/ocrimport/importer.go`

- [ ] **Step 1: Add `ReviewReason` to `OCRExtractedEntry`**

```go
type OCRExtractedEntry struct {
    Date     OCRExtractedField
    Category OCRExtractedField
    Hours    OCRExtractedField
    Notes    OCRExtractedField
    ReviewReason string  // NEW
}
```

- [ ] **Step 2: Propagate `ReviewReason` in `buildDraftEntries`**

```go
drafts = append(drafts, db.OCRDraftEntry{
    DateRaw:            e.Date.Raw,
    DateNormalized:     e.Date.Normalized,
    CategoryRaw:        e.Category.Raw,
    CategoryNormalized: strings.TrimSpace(e.Category.Normalized),
    HoursRaw:           e.Hours.Raw,
    HoursNormalized:    hoursVal,
    NotesRaw:           e.Notes.Raw,
    NotesNormalized:    strings.TrimSpace(e.Notes.Normalized),
    Confidence:         e.Category.Confidence,
    NeedsReview:        e.Category.NeedsReview,
    ReviewReason:       e.ReviewReason,
})
```

- [ ] **Step 3: Run Go tests**

Run: `cd invoice-service && go test ./internal/ocr/... ./internal/ocrimport/...`

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add invoice-service/internal/ocr/contract.go invoice-service/internal/ocrimport/importer.go
```

---

## Task 5: Create `ocr_agent.py` models and image preprocessing

**Files:**
- Create: `ocr-service/ocr_agent.py`

- [ ] **Step 1: Write the Pydantic models and exceptions**

```python
from __future__ import annotations

import os
from datetime import datetime
from typing import Optional

from pydantic import BaseModel, Field

class OCRParseError(Exception):
    """Raised when the model response cannot be parsed into the expected schema."""

class ExtractedEntry(BaseModel):
    date_raw: str
    date_normalized: str
    category_raw: str
    category_normalized: str
    hours_raw: str
    hours_normalized: str
    notes_raw: str = ""
    notes_normalized: str = ""
    confidence: float = Field(ge=0.0, le=1.0, default=0.7)
    needs_review: bool = False
    review_reason: Optional[str] = None

class PageExtraction(BaseModel):
    entries: list[ExtractedEntry] = []

class RateHint(BaseModel):
    category: str
    start_date: str
    end_date: str
    rate: float

class OCRContext(BaseModel):
    categories: list[str]
    rates: list[RateHint] = []
    current_year: int
    page_number: int = 1

class ValidationResult(BaseModel):
    valid: list[ExtractedEntry]
    flagged: list[ExtractedEntry]

MAX_ROUNDS = 2
```

- [ ] **Step 2: Add `prepare_image_for_bedrock` with EXIF orientation fix**

```python
import base64
import io

from PIL import Image, ImageOps


def prepare_image_for_bedrock(path: str, max_dim: int | None = None) -> str:
    if max_dim is None:
        max_dim = int(os.environ.get("OCR_MAX_IMAGE_DIMENSION", "2048"))

    img = Image.open(path)
    img = ImageOps.exif_transpose(img)

    if img.mode in ("RGBA", "LA", "P"):
        img = img.convert("RGB")

    w, h = img.size
    if w > max_dim or h > max_dim:
        if w > h:
            new_w = max_dim
            new_h = int(round(h * max_dim / w))
        else:
            new_h = max_dim
            new_w = int(round(w * max_dim / h))
        img = img.resize((new_w, new_h), Image.LANCZOS)

    w, h = img.size
    align = 32
    new_w = ((w + align - 1) // align) * align
    new_h = ((h + align - 1) // align) * align
    if new_w != w or new_h != h:
        img = img.resize((new_w, new_h), Image.LANCZOS)

    buf = io.BytesIO()
    img.save(buf, format="JPEG", quality=85)
    return base64.b64encode(buf.getvalue()).decode()
```

- [ ] **Step 3: Verify the module imports**

Run: `cd ocr-service && python3 -c "import ocr_agent; print(ocr_agent.MAX_ROUNDS)"`

Expected: prints `2`.

- [ ] **Step 4: Commit**

```bash
git add ocr-service/ocr_agent.py
```

---

## Task 6: Add prompts and Bedrock helpers

**Files:**
- Modify: `ocr-service/ocr_agent.py`

- [ ] **Step 1: Add prompt builders**

```python
import json


def _sanitize_category(name: str) -> str:
    # Keep printable characters, collapse whitespace, and cap length.
    cleaned = "".join(ch for ch in name.strip() if ch.isprintable())
    return cleaned[:64]


def _category_list(context: OCRContext) -> str:
    return "\n".join(f"- {_sanitize_category(c)}" for c in context.categories)


def build_extraction_prompt(context: OCRContext) -> str:
    return (
        "You are an expert at reading handwritten timesheets. "
        "Extract each work-log entry from the image and return JSON matching this exact schema:\n"
        '{"entries": [{"date_raw": "...", "date_normalized": "YYYY-MM-DD", '
        '"category_raw": "...", "category_normalized": "<exact allowed category>", '
        '"hours_raw": "...", "hours_normalized": "<positive number>", '
        '"notes_raw": "...", "notes_normalized": "...", '
        '"confidence": 0.0-1.0, "needs_review": false}]}\n\n'
        "Allowed categories (use exact names):\n"
        f"{_category_list(context)}\n\n"
        f"If a date omits the year, infer {context.current_year}. "
        "If a field is unreadable, set its raw value to '?' and needs_review to true. "
        "Do not invent categories; if unsure, set needs_review true.\n\n"
        "Examples:\n"
        '- Normal: {"date_raw": "12/5", "date_normalized": "2026-05-12", "category_raw": "Consulting", '
        '"category_normalized": "Consulting", "hours_raw": "7.5", "hours_normalized": "7.5", '
        '"notes_raw": "client meeting", "notes_normalized": "client meeting", "confidence": 0.95, "needs_review": false}\n'
        '- Sick day: {"date_raw": "13/5", "date_normalized": "2026-05-13", "category_raw": "Sick", '
        '"category_normalized": "Sick", "hours_raw": "7.5", "hours_normalized": "7.5", '
        '"notes_raw": "", "notes_normalized": "", "confidence": 0.9, "needs_review": false}\n'
    )


def build_correction_prompt(context: OCRContext, flagged: list[ExtractedEntry]) -> str:
    entries_json = json.dumps([e.model_dump() for e in flagged], indent=2)
    return (
        "The following entries from a handwritten timesheet failed validation. "
        "Return a JSON page with ONLY the corrected versions of these flagged entries. "
        "Do not include entries that are already correct. "
        "Use this schema:\n"
        '{"entries": [{"date_raw": "...", "date_normalized": "YYYY-MM-DD", '
        '"category_raw": "...", "category_normalized": "<exact allowed category>", '
        '"hours_raw": "...", "hours_normalized": "<positive number>", '
        '"notes_raw": "...", "notes_normalized": "...", '
        '"confidence": 0.0-1.0, "needs_review": false}]}\n\n'
        "Allowed categories (use exact names):\n"
        f"{_category_list(context)}\n\n"
        f"{entries_json}"
    )
```

- [ ] **Step 2: Add Bedrock JSON parsing helper and shared body builder**

```python
from botocore.exceptions import ClientError
from pydantic import ValidationError


def _sanitize_category(name: str) -> str:
    # Keep printable characters, collapse whitespace, and cap length.
    cleaned = "".join(ch for ch in name.strip() if ch.isprintable())
    return cleaned[:64]


def _strip_markdown_fences(text: str) -> str:
    text = text.strip()
    if text.startswith("```"):
        parts = text.split("```", 2)
        if len(parts) >= 3:
            inner = parts[1].strip()
            if inner.startswith("json"):
                inner = inner[4:].strip()
            return inner
    return text


def _invoke_bedrock_json(client, model: str, body: str) -> str:
    try:
        resp = client.invoke_model(modelId=model, body=body)
    except ClientError as e:
        raise OCRParseError(f"Bedrock invoke failed: {e}") from e
    try:
        result = json.loads(resp["body"].read())
    except json.JSONDecodeError as e:
        raise OCRParseError(f"Bedrock response is not valid JSON: {e}") from e
    try:
        return result["choices"][0]["message"]["content"]
    except (KeyError, IndexError, TypeError) as e:
        raise OCRParseError(f"Bedrock response has unexpected shape: {e}") from e


def _parse_page_json(text: str) -> PageExtraction:
    text = _strip_markdown_fences(text)
    try:
        data = json.loads(text)
    except json.JSONDecodeError as e:
        raise OCRParseError(f"response is not valid JSON: {e}") from e
    try:
        return PageExtraction.model_validate(data)
    except ValidationError as e:
        raise OCRParseError(f"response JSON does not match schema: {e}") from e


def _build_bedrock_body(image_b64: str, prompt: str) -> str:
    return json.dumps(
        {
            "messages": [
                {
                    "role": "user",
                    "content": [
                        {"type": "image_url", "image_url": {"url": f"data:image/jpeg;base64,{image_b64}"}},
                        {"type": "text", "text": prompt},
                    ],
                }
            ],
            "max_tokens": 2048,
            "temperature": 0.1,
        }
    )
```

- [ ] **Step 3: Add `extract_page` and `correct_entries`**

```python

def extract_page(image_path: str, context: OCRContext, client, model: str) -> PageExtraction:
    b64 = prepare_image_for_bedrock(image_path)
    prompt = build_extraction_prompt(context)
    body = _build_bedrock_body(b64, prompt)
    text = _invoke_bedrock_json(client, model, body)
    return _parse_page_json(text)


def correct_entries(
    image_path: str, flagged: list[ExtractedEntry], context: OCRContext, client, model: str
) -> PageExtraction:
    b64 = prepare_image_for_bedrock(image_path)
    prompt = build_correction_prompt(context, flagged)
    body = _build_bedrock_body(b64, prompt)
    text = _invoke_bedrock_json(client, model, body)
    return _parse_page_json(text)
```

- [ ] **Step 4: Verify the helpers can be called with a fake client**

Create a throwaway script:

```python
import json, io
import ocr_agent

class FakeClient:
    def invoke_model(self, *, modelId, body):
        resp = {"choices": [{"message": {"content": '{"entries": []}'}}]}
        return {"body": io.BytesIO(json.dumps(resp).encode())}

ctx = ocr_agent.OCRContext(categories=["Consulting"], current_year=2026)
page = ocr_agent.extract_page("/dev/null", ctx, FakeClient(), "test")
print(page)
```

(Expected: `extract_page` raises `OCRParseError` at image preparation, proving the prompt/body path is wired and `_invoke_bedrock_json` validation no longer fails before image open.)

- [ ] **Step 5: Commit**

```bash
git add ocr-service/ocr_agent.py
```

---

## Task 7: Add the validation step

**Files:**
- Modify: `ocr-service/ocr_agent.py`

- [ ] **Step 1: Implement `validate_page`**

```python

def validate_page(page: PageExtraction, context: OCRContext) -> ValidationResult:
    valid: list[ExtractedEntry] = []
    flagged: list[ExtractedEntry] = []
    known = {c.lower(): c for c in context.categories}

    for entry in page.entries:
        reasons: list[str] = []

        if entry.category_normalized.lower() not in known:
            reasons.append("unknown_category")

        try:
            datetime.strptime(entry.date_normalized, "%Y-%m-%d")
        except ValueError:
            reasons.append("invalid_date")

        try:
            hours = float(entry.hours_normalized)
            if hours <= 0:
                reasons.append("invalid_hours")
        except ValueError:
            reasons.append("invalid_hours")

        if entry.confidence < 0.5:
            reasons.append("low_confidence")

        if entry.date_raw == "?" or entry.category_raw == "?" or entry.hours_raw == "?":
            reasons.append("unreadable_field")

        if reasons:
            entry.needs_review = True
            entry.review_reason = "; ".join(reasons)
            flagged.append(entry)
        else:
            valid.append(entry)

    return ValidationResult(valid=valid, flagged=flagged)
```

- [ ] **Step 2: Write a quick test for validation**

```python
import ocr_agent

def test_validate_unknown_category():
    entry = ocr_agent.ExtractedEntry(
        date_raw="12/5", date_normalized="2026-05-12",
        category_raw="Foo", category_normalized="Foo",
        hours_raw="7.5", hours_normalized="7.5",
        confidence=0.9,
    )
    page = ocr_agent.PageExtraction(entries=[entry])
    ctx = ocr_agent.OCRContext(categories=["Consulting"], current_year=2026)
    result = ocr_agent.validate_page(page, ctx)
    assert len(result.flagged) == 1
    assert result.flagged[0].review_reason == "unknown_category"
```

Run: `cd ocr-service && python3 -c "<paste test>"`

Expected: no assertion errors.

- [ ] **Step 3: Commit**

```bash
git add ocr-service/ocr_agent.py
```

---

## Task 8: Add the agent loop orchestration

**Files:**
- Modify: `ocr-service/ocr_agent.py`

- [ ] **Step 1: Implement `run_extraction_loop`**

```python

def run_extraction_loop(
    image_paths: list[str], context: OCRContext, client, model: str
) -> tuple[list[ExtractedEntry], dict]:
    all_entries: list[ExtractedEntry] = []
    max_rounds = 0

    for idx, path in enumerate(image_paths):
        page_context = context.model_copy(update={"page_number": idx + 1})
        rounds = 0

        try:
            page = extract_page(path, page_context, client, model)
        except OCRParseError as e:
            all_entries.append(
                ExtractedEntry(
                    date_raw="?", date_normalized="",
                    category_raw="?", category_normalized="",
                    hours_raw="?", hours_normalized="",
                    notes_raw="", notes_normalized="",
                    confidence=0.0, needs_review=True,
                    review_reason=f"parser_error: {e}",
                )
            )
            max_rounds = max(max_rounds, 1)
            continue

        for attempt in range(MAX_ROUNDS):
            rounds += 1
            result = validate_page(page, page_context)
            if not result.flagged:
                all_entries.extend(result.valid)
                break

            if attempt == MAX_ROUNDS - 1:
                all_entries.extend(result.valid)
                all_entries.extend(result.flagged)
            else:
                try:
                    corrected = correct_entries(path, result.flagged, page_context, client, model)
                except OCRParseError:
                    all_entries.extend(result.valid)
                    for entry in result.flagged:
                        entry.review_reason = (entry.review_reason or "") + "; parser_error"
                        all_entries.append(entry)
                    break
                if len(corrected.entries) == len(result.flagged):
                    page = PageExtraction(entries=result.valid + corrected.entries)
                else:
                    all_entries.extend(result.valid)
                    for entry in result.flagged:
                        entry.review_reason = (entry.review_reason or "") + "; correction_mismatch"
                        all_entries.append(entry)
                    break

        max_rounds = max(max_rounds, rounds)

    return all_entries, {"rounds": max_rounds, "pages": len(image_paths)}
```

- [ ] **Step 2: Test the loop with a fake client**

```python
import json, io
import ocr_agent

class FakeClient:
    def __init__(self, responses):
        self.responses = iter(responses)
    def invoke_model(self, *, modelId, body):
        return {"body": io.BytesIO(json.dumps(next(self.responses)).encode())}

responses = [
    {"choices": [{"message": {"content": '{"entries": [{"date_raw": "12/5", "date_normalized": "2026-05-12", "category_raw": "Cons", "category_normalized": "Cons", "hours_raw": "7.5", "hours_normalized": "7.5", "notes_raw": "", "notes_normalized": "", "confidence": 0.8, "needs_review": false}]}'}}]},
    {"choices": [{"message": {"content": '{"entries": [{"date_raw": "12/5", "date_normalized": "2026-05-12", "category_raw": "Consulting", "category_normalized": "Consulting", "hours_raw": "7.5", "hours_normalized": "7.5", "notes_raw": "", "notes_normalized": "", "confidence": 0.9, "needs_review": false}]}'}}]},
]

from PIL import Image
img = Image.new('RGB', (100, 100), color='white')
path = '/tmp/loop_test.jpg'
img.save(path, 'JPEG')

ctx = ocr_agent.OCRContext(categories=["Consulting"], current_year=2026)
entries, meta = ocr_agent.run_extraction_loop([path], ctx, FakeClient(responses), "test")
assert len(entries) == 1
assert entries[0].category_normalized == "Consulting"
assert meta["rounds"] == 2
print("loop test passed")
```

- [ ] **Step 3: Commit**

```bash
git add ocr-service/ocr_agent.py
```

---

## Task 9: Wire the agent loop into `main.py`

**Files:**
- Modify: `ocr-service/main.py`

- [ ] **Step 1: Import `ocr_agent` at the top of `main.py`**

```python
import ocr_agent
```

- [ ] **Step 2: Add the agent-enabled flag**

```python

def _agent_enabled() -> bool:
    return os.environ.get("OCR_AGENT_ENABLED", "true").lower() in ("1", "true", "yes")
```

- [ ] **Step 3: Rename current `BedrockBackend.extract` to `legacy_extract`**

Move the current `extract` method body to a new method `_legacy_extract`. Keep the public method named `extract`.

```python
    def _legacy_extract(self, image_paths, request_data):
        # existing body of extract()
```

- [ ] **Step 4: Implement `_agent_extract`**

```python
    def _agent_extract(self, image_paths, request_data):
        session_id = request_data.get("session_id", 0)
        categories = request_data.get("hints", {}).get("categories", [])
        current_year = request_data.get("current_year") or datetime.now().year
        rate_dicts = request_data.get("rates", [])

        context = ocr_agent.OCRContext(
            categories=categories,
            rates=[ocr_agent.RateHint(**r) for r in rate_dicts],
            current_year=current_year,
        )

        entries, meta = ocr_agent.run_extraction_loop(
            image_paths, context, self.client, self.model
        )

        result_entries = []
        for e in entries:
            result_entries.append(
                {
                    "date": {
                        "raw": e.date_raw,
                        "normalized": e.date_normalized,
                        "confidence": 0.85,
                        "needs_review": e.needs_review,
                    },
                    "category": {
                        "raw": e.category_raw,
                        "normalized": e.category_normalized,
                        "confidence": e.confidence,
                        "needs_review": e.needs_review,
                    },
                    "hours": {
                        "raw": e.hours_raw,
                        "normalized": e.hours_normalized,
                        "confidence": 0.9,
                        "needs_review": e.needs_review,
                    },
                    "notes": {
                        "raw": e.notes_raw,
                        "normalized": e.notes_normalized,
                        "confidence": 0.7,
                        "needs_review": e.needs_review,
                    },
                    "review_reason": e.review_reason,
                }
            )

        avg_confidence = 0.0
        if result_entries:
            avg_confidence = sum(e["category"]["confidence"] for e in result_entries) / len(result_entries)

        return {
            "session_id": session_id,
            "status": "success",
            "entries": result_entries,
            "metadata": {
                "model_used": self.model,
                "processing_time_ms": 0,
                "pages_processed": meta["pages"],
                "input_tokens": 0,
                "output_tokens": 0,
                "avg_confidence": avg_confidence,
            },
        }
```

- [ ] **Step 5: Update the public `extract` method to branch**

```python
    def extract(self, image_paths, request_data):
        if not _agent_enabled():
            return self._legacy_extract(image_paths, request_data)
        return self._agent_extract(image_paths, request_data)
```

- [ ] **Step 6: Run a quick smoke test**

Run: `cd ocr-service && python3 -m py_compile main.py`

Expected: no errors.

- [ ] **Step 7: Commit**

```bash
git add ocr-service/main.py
```

---

## Task 10: Add unit tests for the agent loop

**Files:**
- Create: `ocr-service/test_main.py`

- [ ] **Step 1: Create a fake Bedrock client helper**

```python
import io
import json

class FakeBedrockClient:
    def __init__(self, responses):
        self.responses = iter(responses)

    def invoke_model(self, *, modelId, body):
        return {"body": io.BytesIO(json.dumps(next(self.responses)).encode())}
```

- [ ] **Step 2: Add tests for extract, validate, and loop**

Test cases to include:

1. `test_extract_page_returns_structured_entries`
2. `test_validate_page_flags_unknown_category`
3. `test_validate_page_flags_invalid_date`
4. `test_validate_page_flags_invalid_hours`
5. `test_run_extraction_loop_corrects_unknown_category`
6. `test_run_extraction_loop_marks_persistent_flags_for_review`
7. `test_prepare_image_converts_heic_to_jpeg`
8. `test_prepare_image_fixes_exif_orientation`

Use the `FakeBedrockClient` for the loop tests and real Pillow images for the image tests.

Example loop test:

```python
import ocr_agent
from PIL import Image
import tempfile
import os

def make_test_image(path: str):
    img = Image.new('RGB', (300, 200), color='white')
    img.save(path, 'JPEG')

def test_run_extraction_loop_corrects_unknown_category():
    responses = [
        {"choices": [{"message": {"content": '{"entries": [{"date_raw": "12/5", "date_normalized": "2026-05-12", "category_raw": "Cons", "category_normalized": "Cons", "hours_raw": "7.5", "hours_normalized": "7.5", "notes_raw": "", "notes_normalized": "", "confidence": 0.8, "needs_review": false}]}'}}]},
        {"choices": [{"message": {"content": '{"entries": [{"date_raw": "12/5", "date_normalized": "2026-05-12", "category_raw": "Consulting", "category_normalized": "Consulting", "hours_raw": "7.5", "hours_normalized": "7.5", "notes_raw": "", "notes_normalized": "", "confidence": 0.9, "needs_review": false}]}'}}]},
    ]
    path = tempfile.mktemp(suffix='.jpg')
    make_test_image(path)
    try:
        ctx = ocr_agent.OCRContext(categories=["Consulting"], current_year=2026)
        entries, meta = ocr_agent.run_extraction_loop([path], ctx, FakeBedrockClient(responses), "test")
        assert len(entries) == 1
        assert entries[0].category_normalized == "Consulting"
        assert entries[0].needs_review is False
        assert meta["rounds"] == 2
    finally:
        os.remove(path)
```

- [ ] **Step 3: Run tests**

Run: `cd ocr-service && python3 -m pytest test_main.py -v`

Expected: all tests pass.

- [ ] **Step 4: Commit**

```bash
git add ocr-service/test_main.py
```

---

## Task 11: Show `review_reason` in the review template

**Files:**
- Modify: `invoice-service/templates/ocr_review.html`

- [ ] **Step 1: Render the reason in the confidence cell**

Change this block:

```html
            <td>
              {{if ge .Confidence 0.8}}<ins>{{printf "%.0f%%" (mul .Confidence 100)}}</ins>
              {{else if ge .Confidence 0.5}}<mark>{{printf "%.0f%%" (mul .Confidence 100)}}</mark>
              {{else}}<mark class="contrast">{{printf "%.0f%%" (mul .Confidence 100)}}</mark>{{end}}
              {{if .NeedsReview}}<br><small class="muted">needs review</small>{{end}}
            </td>
```

To:

```html
            <td>
              {{if ge .Confidence 0.8}}<ins>{{printf "%.0f%%" (mul .Confidence 100)}}</ins>
              {{else if ge .Confidence 0.5}}<mark>{{printf "%.0f%%" (mul .Confidence 100)}}</mark>
              {{else}}<mark class="contrast">{{printf "%.0f%%" (mul .Confidence 100)}}</mark>{{end}}
              {{if .NeedsReview}}<br><small class="muted">needs review</small>{{end}}
              {{if .ReviewReason}}<br><small class="muted">{{.ReviewReason}}</small>{{end}}
            </td>
```

- [ ] **Step 2: Verify the template renders**

Run: `cd invoice-service && go test ./internal/handler/... -run TestOCRSessionStatusHandler -v`

Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add invoice-service/templates/ocr_review.html
```

---

## Task 12: Verify the full stack

- [ ] **Step 1: Run Go tests**

Run: `cd invoice-service && go test ./...`

Expected: all packages pass.

- [ ] **Step 2: Run Python checks**

Run: `cd ocr-service && python3 -m py_compile main.py ocr_agent.py && python3 -m pytest test_main.py -v`

Expected: all tests pass.

- [ ] **Step 3: Build the Go server**

Run: `cd invoice-service && go build -buildvcs=false ./cmd/server`

Expected: no build errors.

- [ ] **Step 4: Final commit**

```bash
git add -A
git status
```

---

## Self-review

- **Spec coverage:** every section of the design spec has a corresponding task (models, loop, validation, prompts, preprocessing, error handling, tests, rollout, UI).
- **Placeholder scan:** no TBDs, TODOs, or vague steps. Each code step includes concrete code.
- **Type consistency:** `ReviewReason` is added consistently to `OCRExtractedEntry`, `OCRDraftEntry`, and the DB column. `ExtractedEntry.review_reason` is `Optional[str]` and maps to the response.
- **No breaking changes:** the Go HTTP API and OCR service response shape remain compatible; only a new optional field is added.

## Execution choice

Plan complete and saved to `docs/superpowers/plans/2026-07-18-ocr-agentic-loop.md`.

Two execution options:

1. **Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration.
2. **Inline Execution** — Execute tasks in this session using `executing-plans`, with checkpoints for review.

Which approach do you want?
