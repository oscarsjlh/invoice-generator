# Tracing

This project uses OpenTelemetry traces to connect browser requests, Go handler work, OCR background jobs, the OCR HTTP call, and the Python OCR service's Bedrock request.

## Local Setup

Run the compose stack:

```bash
docker compose up --build
```

Jaeger is available at `http://localhost:16686`. The Go and Python services export OTLP/HTTP traces to `http://jaeger:4318`, which is Jaeger's OTLP HTTP endpoint.

To create a cross-service trace:

1. Open `http://localhost:8080`.
2. Sign in or register if auth is enabled.
3. Start an OCR import.
4. In Jaeger, choose `invoice-service` or `ocr-service` and search recent traces.

## Trace Shape

A normal web request creates an `invoice-service` server span around the HTTP middleware and handler chain. Request logs still include the existing `request_id`, and when tracing is active they also include `trace_id` and `span_id` so logs and traces can be joined.

Database spans are created by the request-scoped store wrapper. They are named by stable application operations such as `db.entries.list`, `db.invoices.generate`, and `db.settings.load`. The spans intentionally do not include SQL text, customer details, invoice contents, or uploaded file paths.

PDF spans are emitted during invoice preview/download and email delivery. `invoice_delivery.render_pdf` or `invoice_delivery.send` wraps the user-facing workflow, and `pdf.render_invoice` covers Typst template loading, Typst content formatting, and PDF compilation.

An OCR import should include these spans in one trace:

- `invoice-service`: inbound HTTP request.
- `invoice-service`: database spans for session creation, image records, draft records, and invoice data lookups.
- `invoice-service`: `ocr.process_session` for the background OCR workflow.
- `invoice-service`: `ocr.extract` and the outbound `POST /ocr/extract` HTTP client span.
- `ocr-service`: inbound `POST /ocr/extract` server span.
- `ocr-service`: `ocr.prepare_images`.
- `ocr-service`: `bedrock.invoke_model`.

The app keeps sensitive values out of span attributes. Do not add invoice contents, uploaded image bytes, cookies, WebAuthn payloads, SMTP credentials, or AWS credentials to traces.

## Configuration

Tracing is disabled in the Go service unless `OTEL_EXPORTER_OTLP_ENDPOINT` is set. The Python service creates spans either way, but only exports them when the same endpoint is set.

Common local values:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
export OTEL_TRACES_SAMPLER=parentbased_traceidratio
export OTEL_TRACES_SAMPLER_ARG=1.0
```

Use `OTEL_SDK_DISABLED=true` to disable tracing during local debugging.

For production, point `OTEL_EXPORTER_OTLP_ENDPOINT` at an OpenTelemetry Collector or vendor OTLP endpoint. Prefer a collector for retries, batching, and routing.

## Troubleshooting

If Jaeger has no traces, check that `OTEL_EXPORTER_OTLP_ENDPOINT` is set for the service and that the endpoint is reachable from inside the container. In compose, services should use `http://jaeger:4318`, not `localhost`.

If only one service appears in a trace, check that the Go OCR client is using request contexts and that the Python service is receiving the `traceparent` header.

If logs have `request_id` but no `trace_id`, tracing is disabled or the request did not pass through the OpenTelemetry HTTP wrapper.

References:

- OpenTelemetry Go getting started: https://opentelemetry.io/docs/languages/go/getting-started/
- OpenTelemetry Python getting started: https://opentelemetry.io/docs/languages/python/getting-started/
- OTLP exporter spec: https://opentelemetry.io/docs/specs/otel/protocol/exporter/
- Jaeger deployment docs: https://www.jaegertracing.io/docs/next-release/deployment/
