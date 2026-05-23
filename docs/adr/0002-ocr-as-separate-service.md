# ADR 0002: OCR As A Separate Service

## Status

Accepted

## Context

OCR import depends on image handling and external model calls. These concerns have different operational characteristics from the Go web application.

## Decision

Keep OCR extraction behind a service seam. The Go application owns the OCR import workflow, uploaded image lifecycle, draft review, and ledger confirmation. The OCR service owns extraction from supplied images.

## Consequences

The web application should depend on a small OCR extractor interface, not the OCR service implementation. The OCR import workflow should be a deep module in the Go application so handlers do not own session state transitions, normalization, or draft persistence.
