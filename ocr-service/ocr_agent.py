"""OCR agentic loop: models, image preprocessing, and extraction helpers."""

from __future__ import annotations

import base64
import io
import json
import os
from datetime import datetime
from typing import Optional

from PIL import Image, ImageOps
from pydantic import BaseModel, Field, ValidationError


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


def _category_list(context: OCRContext) -> str:
    return "\n".join(f"- {c}" for c in context.categories)


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
        "Return a complete JSON page with ALL entries, correcting only the flagged ones. "
        "Keep valid entries unchanged. Use this schema:\n"
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


def _invoke_bedrock_json(client, model: str, body: str) -> dict:
    resp = client.invoke_model(modelId=model, body=body)
    result = json.loads(resp["body"].read())
    return result["choices"][0]["message"]["content"]


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


def extract_page(image_path: str, context: OCRContext, client, model: str) -> PageExtraction:
    b64 = prepare_image_for_bedrock(image_path)
    prompt = build_extraction_prompt(context)
    body = json.dumps(
        {
            "messages": [
                {
                    "role": "user",
                    "content": [
                        {"type": "image_url", "image_url": {"url": f"data:image/jpeg;base64,{b64}"}},
                        {"type": "text", "text": prompt},
                    ],
                }
            ],
            "max_tokens": 2048,
            "temperature": 0.1,
        }
    )
    text = _invoke_bedrock_json(client, model, body)
    return _parse_page_json(text)


def correct_entries(
    image_path: str, flagged: list[ExtractedEntry], context: OCRContext, client, model: str
) -> PageExtraction:
    b64 = prepare_image_for_bedrock(image_path)
    prompt = build_correction_prompt(context, flagged)
    body = json.dumps(
        {
            "messages": [
                {
                    "role": "user",
                    "content": [
                        {"type": "image_url", "image_url": {"url": f"data:image/jpeg;base64,{b64}"}},
                        {"type": "text", "text": prompt},
                    ],
                }
            ],
            "max_tokens": 2048,
            "temperature": 0.1,
        }
    )
    text = _invoke_bedrock_json(client, model, body)
    return _parse_page_json(text)
