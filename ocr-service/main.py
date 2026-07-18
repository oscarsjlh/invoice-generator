"""
OCR Service for invoice-generator paper entry import.

Accepts images + context hints from the Go app, returns structured extraction results.
Uses AWS Bedrock with Claude for handwriting extraction.

Endpoints:
    POST /ocr/extract  - Accept multipart form with images + JSON request body

Environment variables:
    OCR_LISTEN                - Bind address (default: :8000)
    OCR_MAX_IMAGE_DIMENSION   - Maximum width/height in pixels before resizing (default: 2048)
    BEDROCK_REGION            - AWS region (default: us-east-1)
    BEDROCK_MODEL             - Model ID (default: us.anthropic.claude-3-5-haiku-20241022-v1:0)
"""

import base64
import io
import json
import os
import tempfile
import time
from datetime import datetime
from http.server import HTTPServer, BaseHTTPRequestHandler
from typing import Optional

import boto3
from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.propagate import extract
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.trace import SpanKind, Status, StatusCode
from PIL import Image

try:
    from pillow_heif import register_heif_opener

    register_heif_opener()
except ImportError:
    pass

import ocr_agent

tracer = trace.get_tracer("invoice-app/ocr-service")


def _agent_enabled() -> bool:
    return os.environ.get("OCR_AGENT_ENABLED", "true").lower() in ("1", "true", "yes")


def setup_tracing():
    provider = TracerProvider(resource=Resource.create({"service.name": "ocr-service"}))
    if os.environ.get("OTEL_EXPORTER_OTLP_ENDPOINT"):
        provider.add_span_processor(BatchSpanProcessor(OTLPSpanExporter()))
    trace.set_tracer_provider(provider)
    return provider.shutdown


def prepare_image_for_bedrock(path):
    img = Image.open(path)
    if img.mode in ("RGBA", "LA", "P"):
        img = img.convert("RGB")

    max_dim = int(os.environ.get("OCR_MAX_IMAGE_DIMENSION", "2048"))
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


def parse_multipart(body, boundary):
    parts = []
    delimiter = b"--" + boundary
    sections = body.split(delimiter)
    for section in sections:
        if not section or section == b"--":
            continue
        if section.startswith(b"--"):
            section = section[2:]
        header_end = section.find(b"\r\n\r\n")
        if header_end == -1:
            continue
        headers_raw = section[:header_end].decode("utf-8", errors="ignore")
        data = section[header_end + 4 :]
        if data.endswith(b"\r\n"):
            data = data[:-2]
        name = ""
        filename = ""
        for line in headers_raw.split("\r\n"):
            if line.lower().startswith("content-disposition:"):
                for part in line.split(";"):
                    part = part.strip()
                    if part.startswith('name="'):
                        name = part[6:].rstrip('"')
                    elif part.startswith('filename="'):
                        filename = part[10:].rstrip('"')
        if name:
            parts.append((name, filename, data))
    return parts


def normalize_date(raw, current_year: Optional[int] = None):
    if current_year is None:
        current_year = datetime.now().year
    raw = raw.strip()
    parts = [p.strip() for p in raw.replace("-", "/").split("/")]
    if len(parts) == 3 and len(parts[0]) == 4:
        return raw.replace("/", "-")
    if len(parts) == 2:
        parts.append(str(current_year))
    elif len(parts) == 3 and len(parts[2]) == 2:
        parts[2] = "20" + parts[2]
    if len(parts) == 3:
        return f"{int(parts[2]):04d}-{int(parts[1]):02d}-{int(parts[0]):02d}"

    months = {
        "jan": 1,
        "feb": 2,
        "mar": 3,
        "apr": 4,
        "may": 5,
        "jun": 6,
        "jul": 7,
        "aug": 8,
        "sep": 9,
        "oct": 10,
        "nov": 11,
        "dec": 12,
    }
    text = raw.lower().replace(",", "")
    for abbr, num in months.items():
        if abbr in text:
            day_part = text.replace(abbr, "").strip()
            try:
                day = int(day_part)
            except ValueError:
                continue
            return f"{current_year:04d}-{num:02d}-{day:02d}"

    return raw


def parse_handwritten_text(content, categories, current_year=None):
    entries = []
    for line in content.strip().split("\n"):
        line = line.strip()
        if not line:
            continue
        if "|" in line:
            parts = [p.strip() for p in line.split("|")]
        else:
            parts = line.split(None, 3)
        if len(parts) < 3:
            continue
        date_raw = parts[0]
        category_raw = parts[1]
        hours_raw = parts[2]
        notes_raw = parts[3] if len(parts) > 3 else ""

        category_normalized = category_raw
        confidence = 0.7
        needs_review = False

        for cat in categories:
            if category_raw.lower() == cat.lower():
                category_normalized = cat
                confidence = 0.95
                break
            if cat.lower().startswith(
                category_raw.lower()
            ) or category_raw.lower().startswith(cat.lower()):
                category_normalized = cat
                confidence = 0.80
                needs_review = True
                break

        entries.append(
            {
                "date": {
                    "raw": date_raw,
                    "normalized": normalize_date(date_raw, current_year),
                    "confidence": 0.85,
                    "needs_review": False,
                },
                "category": {
                    "raw": category_raw,
                    "normalized": category_normalized,
                    "confidence": confidence,
                    "needs_review": needs_review,
                },
                "hours": {
                    "raw": hours_raw,
                    "normalized": hours_raw,
                    "confidence": 0.90,
                    "needs_review": False,
                },
                "notes": {
                    "raw": notes_raw,
                    "normalized": notes_raw,
                    "confidence": 0.70,
                    "needs_review": False,
                },
            }
        )
    return entries


class BedrockBackend:
    def __init__(self):
        self.region = os.environ.get("BEDROCK_REGION", "eu-west-2")
        self.model = os.environ.get("BEDROCK_MODEL", "qwen.qwen3-vl-235b-a22b")
        access_key = os.getenv("AWS_ACCESS_KEY_ID") or os.getenv("AWS_ACCES_KEY_ID")
        secret_key = os.getenv("AWS_SECRET_ACCESS_KEY")
        self.client = boto3.client(
            "bedrock-runtime",
            region_name=self.region,
            aws_access_key_id=access_key,
            aws_secret_access_key=secret_key,
        )

    def _legacy_extract(self, image_paths, request_data):
        session_id = request_data.get("session_id", 0)
        categories = request_data.get("hints", {}).get("categories", [])
        current_year = request_data.get("current_year") or datetime.now().year
        cat_list = ", ".join(categories) if categories else "(no known categories)"

        prompt = (
            "Extract handwritten work-log entries from this page. "
            "Each line contains: date, category, hours, optional notes. "
            f"If a date omits the year, infer {current_year}. "
            f"Known categories (prefer these if the handwriting matches): {cat_list}. "
            "Output format — one entry per line: DATE | CATEGORY | HOURS | NOTES\n"
            "If a field is unreadable, use '?'."
        )

        with tracer.start_as_current_span("ocr.prepare_images") as span:
            span.set_attribute("ocr.session_id", session_id)
            span.set_attribute("ocr.image_count", len(image_paths))
            msg_content = []
            for path in image_paths:
                b64 = prepare_image_for_bedrock(path)
                msg_content.append(
                    {
                        "type": "image_url",
                        "image_url": {"url": f"data:image/jpeg;base64,{b64}"},
                    }
                )
        msg_content.append({"type": "text", "text": prompt})

        body = json.dumps(
            {
                "messages": [{"role": "user", "content": msg_content}],
                "max_tokens": 2048,
                "temperature": 0.1,
            }
        )

        with tracer.start_as_current_span("bedrock.invoke_model") as span:
            span.set_attribute("ocr.session_id", session_id)
            span.set_attribute("gen_ai.system", "aws.bedrock")
            span.set_attribute("gen_ai.request.model", self.model)
            span.set_attribute("aws.region", self.region)
            span.set_attribute("ocr.request_body_bytes", len(body))
            start = time.time()
            resp = self.client.invoke_model(
                modelId=self.model,
                body=body,
            )
            elapsed_ms = int((time.time() - start) * 1000)
            span.set_attribute("ocr.processing_time_ms", elapsed_ms)

        result = json.loads(resp["body"].read())
        text = result["choices"][0]["message"]["content"]

        entries = parse_handwritten_text(text, categories, current_year)

        input_tokens = result.get("usage", {}).get("input_tokens", 0)
        output_tokens = result.get("usage", {}).get("output_tokens", 0)

        return {
            "session_id": session_id,
            "status": "success",
            "entries": entries,
            "metadata": {
                "model_used": self.model,
                "processing_time_ms": elapsed_ms,
                "pages_processed": len(image_paths),
                "input_tokens": input_tokens,
                "output_tokens": output_tokens,
                "avg_confidence": sum(e["category"]["confidence"] for e in entries)
                / max(len(entries), 1),
            },
        }

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

    def extract(self, image_paths, request_data):
        if not _agent_enabled():
            return self._legacy_extract(image_paths, request_data)
        return self._agent_extract(image_paths, request_data)


class OCRHandler(BaseHTTPRequestHandler):
    backend = None

    def do_POST(self):
        parent_context = extract(self.headers)
        with tracer.start_as_current_span(
            "POST /ocr/extract", context=parent_context, kind=SpanKind.SERVER
        ) as span:
            span.set_attribute("http.request.method", "POST")
            span.set_attribute("url.path", self.path)

            if self.path != "/ocr/extract":
                span.set_attribute("http.response.status_code", 404)
                self.send_response(404)
                self.end_headers()
                return

            content_type = self.headers.get("Content-Type", "")
            if "multipart/form-data" not in content_type:
                span.set_attribute("http.response.status_code", 400)
                self.send_error(400, "expected multipart/form-data")
                return

            boundary_raw = content_type.split("boundary=")[1]
            boundary = boundary_raw.split(";")[0].strip().encode()
            body = self.rfile.read(int(self.headers["Content-Length"]))
            parts = parse_multipart(body, boundary)

            request_data = None
            image_paths = []
            for name, filename, data in parts:
                if name == "request":
                    request_data = json.loads(data.decode("utf-8"))
                elif name == "images":
                    tmpdir = tempfile.mkdtemp(prefix="ocr_")
                    img_path = os.path.join(tmpdir, filename or "image.jpg")
                    with open(img_path, "wb") as f:
                        f.write(data)
                    image_paths.append(img_path)

            session_id = request_data.get("session_id", 0) if request_data else 0
            span.set_attribute("ocr.session_id", session_id)
            span.set_attribute("ocr.image_count", len(image_paths))

            if not image_paths:
                span.set_attribute("http.response.status_code", 400)
                self.send_error(400, "no images provided")
                return

            try:
                if OCRHandler.backend is None:
                    OCRHandler.backend = BedrockBackend()
                result = OCRHandler.backend.extract(image_paths, request_data or {})
                span.set_attribute("http.response.status_code", 200)
                span.set_attribute(
                    "ocr.entries_extracted", len(result.get("entries", []))
                )
                self.send_response(200)
                self.send_header("Content-Type", "application/json")
                self.end_headers()
                self.wfile.write(json.dumps(result).encode())
            except Exception as e:
                span.record_exception(e)
                span.set_status(Status(StatusCode.ERROR, str(e)))
                span.set_attribute("http.response.status_code", 500)
                self.send_response(500)
                self.send_header("Content-Type", "application/json")
                self.end_headers()
                self.wfile.write(
                    json.dumps(
                        {"session_id": session_id, "status": "error", "error": str(e)}
                    ).encode()
                )


if __name__ == "__main__":
    shutdown_tracing = setup_tracing()
    listen = os.environ.get("OCR_LISTEN", ":8000")
    host, port = listen.rsplit(":", 1) if ":" in listen else ("", listen)
    port = int(port)

    print(f"OCR service listening on {host}:{port} (backend: bedrock)")
    try:
        HTTPServer((host, port), OCRHandler).serve_forever()
    finally:
        shutdown_tracing()
