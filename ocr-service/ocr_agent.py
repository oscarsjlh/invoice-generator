"""OCR agentic loop: models, image preprocessing, and extraction helpers."""

from __future__ import annotations

import base64
import io
import os
from datetime import datetime
from typing import Optional

from PIL import Image, ImageOps
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


def prepare_image_for_bedrock(path: str, max_dim: int | None = None) -> str:
    """Load an image, correct EXIF orientation, resize, and return as a base64 JPEG."""
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
