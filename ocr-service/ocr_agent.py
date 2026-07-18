"""OCR agentic loop: models, image preprocessing, and extraction helpers."""

from __future__ import annotations

import base64
import io
import json
import os
from datetime import datetime
from typing import Optional

from botocore.exceptions import ClientError
from PIL import Image, ImageOps
from pydantic import BaseModel, Field, ValidationError

try:
    from pillow_heif import register_heif_opener

    register_heif_opener()
except ImportError:
    pass


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


def prepare_image_for_bedrock(path: str, max_dim: int | None = None) -> str:
    """Load an image, correct EXIF orientation, resize, and return as a base64 JPEG."""
    if max_dim is None:
        max_dim = int(os.environ.get("OCR_MAX_IMAGE_DIMENSION", "2048"))

    try:
        with Image.open(path) as img:
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
    except (OSError, ValueError) as e:
        raise OCRParseError(f"failed to prepare image {path!r}: {e}") from e


def _sanitize_category(name: str) -> str:
    # Keep printable characters, trim edges, and cap length.
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
    except Exception as e:
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


def validate_page(page: PageExtraction, context: OCRContext) -> ValidationResult:
    """Classify extracted entries as valid or flagged.

    Each entry is checked against the provided context: the normalized category
    must be known, the normalized date must parse as YYYY-MM-DD, hours must be a
    positive number, confidence must be at least 0.5, and no raw field may be a
    literal "?" marker. Flagged entries are mutated in place; their
    ``needs_review`` field is set to ``True`` and ``review_reason`` is set to a
    semicolon-separated list of failure reasons.

    This function is intended to be called by the agentic extraction loop after a
    page has been extracted and before any correction/refinement round.
    """
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
