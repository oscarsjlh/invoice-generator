from decimal import Decimal
from typing import List, Optional
from pydantic import BaseModel, ConfigDict, Field
from PIL import Image
import io

from pillow_heif import register_heif_opener

register_heif_opener()


class DetectedPage(BaseModel):
    model_config = ConfigDict(arbitrary_types_allowed=True)

    image: Image.Image
    source: str  # 'heic' or 'original'


class DetectedCell(BaseModel):
    text: str
    row: int
    col: int
    confidence: float


class DetectedRow(BaseModel):
    cells: List[DetectedCell]


class ExtractedPage(BaseModel):
    rows: List[DetectedRow]


class ValidatedEntry(BaseModel):
    date: str
    category: str
    hours: float
    notes: Optional[str] = None
    confidence: float
    needs_review: bool
    review_reason: Optional[str] = None


def load_image(bytes_data: bytes, filename: str) -> Image.Image:
    img = Image.open(io.BytesIO(bytes_data))
    if img.mode != 'RGB':
        img = img.convert('RGB')
    buf = io.BytesIO()
    img.save(buf, format='PNG')
    buf.seek(0)
    return Image.open(buf)


def preprocess_image(image: Image.Image, target_size: int = 1568) -> Image.Image:
    w, h = image.size
    if max(w, h) > target_size:
        scale = target_size / max(w, h)
        new_size = (int(round(w * scale)), int(round(h * scale)))
        image = image.resize(new_size, Image.LANCZOS)
    return image


if __name__ == "__main__":
    # Sanity check: ensure models and helpers are importable.
    sample = ValidatedEntry(
        date="2026-07-18",
        category="dev",
        hours=8.0,
        confidence=0.95,
        needs_review=False,
    )
    print(sample)
