"""Unit tests for the OCR agent loop and image preprocessing."""

import base64
import contextlib
import io
import json
import tempfile
from pathlib import Path

import pytest
from botocore.exceptions import ClientError
from PIL import Image

import ocr_agent
from ocr_agent import (
    ExtractedEntry,
    OCRContext,
    PageExtraction,
    extract_page,
    prepare_image_for_bedrock,
    run_extraction_loop,
    validate_page,
)


class FakeBedrockClient:
    def __init__(self, responses):
        self.responses = iter(responses)

    def invoke_model(self, *, modelId, body):
        return {"body": io.BytesIO(json.dumps(next(self.responses)).encode())}


def make_test_image(path: str, size: tuple[int, int] = (300, 200), color: str = "white"):
    img = Image.new("RGB", size, color=color)
    img.save(path, "JPEG")


@contextlib.contextmanager
def make_test_image_path():
    """Create a temporary test image and yield its path.

    The temporary directory is removed when the context manager exits.
    """
    with tempfile.TemporaryDirectory() as tmpdir:
        path = Path(tmpdir) / "page.jpg"
        make_test_image(path)
        yield path


def _entry_dict(**overrides) -> dict:
    defaults = {
        "date_raw": "12/5",
        "date_normalized": "2026-05-12",
        "category_raw": "Consulting",
        "category_normalized": "Consulting",
        "hours_raw": "7.5",
        "hours_normalized": "7.5",
        "notes_raw": "",
        "notes_normalized": "",
        "confidence": 0.8,
        "needs_review": False,
    }
    defaults.update(overrides)
    return defaults


def test_extract_page_returns_structured_entries():
    responses = [
        {
            "choices": [
                {
                    "message": {
                        "content": json.dumps(
                            {"entries": [_entry_dict(category_raw="Consulting", notes_raw="client meeting", notes_normalized="client meeting")]}
                        )
                    }
                }
            ]
        }
    ]
    with tempfile.TemporaryDirectory() as tmpdir:
        path = Path(tmpdir) / "page.jpg"
        make_test_image(path)
        ctx = OCRContext(categories=["Consulting"], current_year=2026)
        page = extract_page(str(path), ctx, FakeBedrockClient(responses), "test")

        assert len(page.entries) == 1
        entry = page.entries[0]
        assert entry.category_normalized == "Consulting"
        assert entry.date_normalized == "2026-05-12"
        assert entry.hours_normalized == "7.5"
        assert entry.notes_normalized == "client meeting"


def test_extract_page_accepts_numeric_hours():
    # LLM sometimes returns hours as JSON numbers; the schema should coerce them to strings.
    raw = {
        "entries": [
            {
                "date_raw": "12/5",
                "date_normalized": "2026-05-12",
                "category_raw": "Consulting",
                "category_normalized": "Consulting",
                "hours_raw": 3.0,
                "hours_normalized": 3.0,
                "notes_raw": "",
                "notes_normalized": "",
                "confidence": 0.9,
                "needs_review": False,
            }
        ]
    }
    page = ocr_agent._parse_page_json(json.dumps(raw))
    assert len(page.entries) == 1
    assert page.entries[0].hours_raw == "3.0"
    assert page.entries[0].hours_normalized == "3.0"


def test_validate_page_flags_unknown_category():
    entry = ExtractedEntry.model_validate(
        _entry_dict(category_normalized="Cons")
    )
    ctx = OCRContext(categories=["Consulting"], current_year=2026)
    result = validate_page(PageExtraction(entries=[entry]), ctx)

    assert len(result.flagged) == 1
    assert len(result.valid) == 0
    assert result.flagged[0].needs_review is True
    assert "unknown_category" in result.flagged[0].review_reason


def test_validate_page_flags_invalid_date():
    entry = ExtractedEntry.model_validate(
        _entry_dict(date_normalized="not-a-date")
    )
    ctx = OCRContext(categories=["Consulting"], current_year=2026)
    result = validate_page(PageExtraction(entries=[entry]), ctx)

    assert len(result.flagged) == 1
    assert result.flagged[0].needs_review is True
    assert "invalid_date" in result.flagged[0].review_reason


def test_validate_page_flags_invalid_hours():
    entry = ExtractedEntry.model_validate(
        _entry_dict(hours_normalized="-3")
    )
    ctx = OCRContext(categories=["Consulting"], current_year=2026)
    result = validate_page(PageExtraction(entries=[entry]), ctx)

    assert len(result.flagged) == 1
    assert result.flagged[0].needs_review is True
    assert "invalid_hours" in result.flagged[0].review_reason


@pytest.mark.parametrize("hours", ["NaN", "inf", "-inf"])
def test_validate_page_flags_non_finite_hours(hours):
    entry = ExtractedEntry.model_validate(_entry_dict(hours_normalized=hours))
    ctx = OCRContext(categories=["Consulting"], current_year=2026)

    result = validate_page(PageExtraction(entries=[entry]), ctx)

    assert len(result.flagged) == 1
    assert "invalid_hours" in result.flagged[0].review_reason


def test_validate_page_flags_entry_without_matching_rate_hint():
    entry = ExtractedEntry.model_validate(_entry_dict(date_normalized="2026-05-12"))
    ctx = OCRContext(
        categories=["Consulting"],
        rates=[
            ocr_agent.RateHint(
                category="Consulting",
                start_date="2026-06-01",
                end_date="",
                rate=100,
            )
        ],
        current_year=2026,
    )

    result = validate_page(PageExtraction(entries=[entry]), ctx)

    assert len(result.flagged) == 1
    assert "missing_rate" in result.flagged[0].review_reason


def test_validate_page_accepts_entry_with_matching_rate_hint():
    entry = ExtractedEntry.model_validate(_entry_dict(date_normalized="2026-05-12"))
    ctx = OCRContext(
        categories=["Consulting"],
        rates=[
            ocr_agent.RateHint(
                category="Consulting",
                start_date="2026-01-01",
                end_date="2026-05-31",
                rate=100,
            )
        ],
        current_year=2026,
    )

    result = validate_page(PageExtraction(entries=[entry]), ctx)

    assert result.valid == [entry]


def test_run_extraction_loop_corrects_unknown_category():
    responses = [
        {
            "choices": [
                {
                    "message": {
                        "content": json.dumps(
                            {"entries": [_entry_dict(category_raw="Cons", category_normalized="Cons", confidence=0.8)]}
                        )
                    }
                }
            ]
        },
        {
            "choices": [
                {
                    "message": {
                        "content": json.dumps(
                            {"entries": [_entry_dict(category_raw="Consulting", category_normalized="Consulting", confidence=0.9)]}
                        )
                    }
                }
            ]
        },
    ]
    with tempfile.TemporaryDirectory() as tmpdir:
        path = Path(tmpdir) / "page.jpg"
        make_test_image(path)
        ctx = OCRContext(categories=["Consulting"], current_year=2026)
        entries, meta = run_extraction_loop([str(path)], ctx, FakeBedrockClient(responses), "test")

        assert len(entries) == 1
        assert entries[0].category_normalized == "Consulting"
        assert entries[0].needs_review is False
        assert meta["rounds"] == 2


def test_run_extraction_loop_marks_persistent_flags_for_review():
    responses = [
        {
            "choices": [
                {
                    "message": {
                        "content": json.dumps(
                            {"entries": [_entry_dict(date_normalized="bad-date", confidence=0.8)]}
                        )
                    }
                }
            ]
        },
        {
            "choices": [
                {
                    "message": {
                        "content": json.dumps(
                            {"entries": [_entry_dict(date_normalized="still-bad", confidence=0.8)]}
                        )
                    }
                }
            ]
        },
    ]
    with tempfile.TemporaryDirectory() as tmpdir:
        path = Path(tmpdir) / "page.jpg"
        make_test_image(path)
        ctx = OCRContext(categories=["Consulting"], current_year=2026)
        entries, meta = run_extraction_loop([str(path)], ctx, FakeBedrockClient(responses), "test")

        assert len(entries) == 1
        assert entries[0].needs_review is True
        assert "invalid_date" in entries[0].review_reason
        assert meta["rounds"] == 2


def test_prepare_image_converts_heic_to_jpeg():
    pytest.importorskip("pillow_heif")
    with tempfile.TemporaryDirectory() as tmpdir:
        heic_path = Path(tmpdir) / "sheet.heic"
        img = Image.new("RGB", (64, 64), color="white")
        img.save(heic_path, "HEIF")

        b64 = prepare_image_for_bedrock(str(heic_path), max_dim=1024)
        raw = base64.b64decode(b64)
        assert raw.startswith(b"\xff\xd8\xff")

        decoded = Image.open(io.BytesIO(raw))
        assert decoded.format == "JPEG"
        assert decoded.mode == "RGB"


def test_prepare_image_fixes_exif_orientation():
    try:
        from PIL import ExifTags
        orientation_tag = ExifTags.Base.Orientation
    except AttributeError:
        orientation_tag = 0x0112

    with tempfile.TemporaryDirectory() as tmpdir:
        path = Path(tmpdir) / "oriented.jpg"
        # Create a portrait image and tag it with orientation 6 (rotated 90 CCW).
        img = Image.new("RGB", (200, 300), color="white")
        exif = img.getexif()
        exif[orientation_tag] = 6
        img.save(path, "JPEG", exif=exif)

        b64 = prepare_image_for_bedrock(str(path), max_dim=1024)
        raw = base64.b64decode(b64)
        decoded = Image.open(io.BytesIO(raw))

        # After EXIF orientation correction the image should be landscape.
        assert decoded.width > decoded.height


def test_validate_page_flags_low_confidence_and_unreadable_field():
    entry = ocr_agent.ExtractedEntry(
        date_raw="?", date_normalized="",
        category_raw="?", category_normalized="Consulting",
        hours_raw="7.5", hours_normalized="7.5",
        confidence=0.4,
    )
    page = ocr_agent.PageExtraction(entries=[entry])
    ctx = ocr_agent.OCRContext(categories=["Consulting"], current_year=2026)
    result = ocr_agent.validate_page(page, ctx)
    assert len(result.flagged) == 1
    assert "low_confidence" in result.flagged[0].review_reason
    assert "unreadable_field" in result.flagged[0].review_reason


def test_run_extraction_loop_handles_parser_error():
    class BadClient:
        def invoke_model(self, *, modelId, body):
            raise ClientError(
                {"Error": {"Code": "500", "Message": "network failure"}},
                "InvokeModel",
            )

    with make_test_image_path() as path:
        ctx = ocr_agent.OCRContext(categories=["Consulting"], current_year=2026)
        entries, meta = ocr_agent.run_extraction_loop([str(path)], ctx, BadClient(), "test")
        assert len(entries) == 1
        assert entries[0].needs_review is True
        assert "parser_error" in entries[0].review_reason


def test_run_extraction_loop_handles_correction_mismatch():
    # First response has a flag, second correction returns a different count.
    responses = [
        {"choices": [{"message": {"content": '{"entries": [{"date_raw": "12/5", "date_normalized": "2026-05-12", "category_raw": "Cons", "category_normalized": "Cons", "hours_raw": "7.5", "hours_normalized": "7.5", "notes_raw": "", "notes_normalized": "", "confidence": 0.8, "needs_review": false}]}'}}]},
        {"choices": [{"message": {"content": '{"entries": []}'}}]},
    ]
    with make_test_image_path() as path:
        ctx = ocr_agent.OCRContext(categories=["Consulting"], current_year=2026)
        entries, meta = ocr_agent.run_extraction_loop([str(path)], ctx, FakeBedrockClient(responses), "test")
        assert len(entries) == 1
        assert entries[0].needs_review is True
        assert "correction_mismatch" in entries[0].review_reason


def test_parse_page_json_strips_markdown_fences():
    text = "```json\n{\"entries\": []}\n```"
    page = ocr_agent._parse_page_json(text)
    assert page.entries == []
