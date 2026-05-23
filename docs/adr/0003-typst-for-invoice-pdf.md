# ADR 0003: Typst For Invoice PDF Rendering

## Status

Accepted

## Context

Invoices need deterministic PDF output with a maintainable template.

## Decision

Use Typst for invoice PDF rendering. Keep Typst process execution and filesystem details inside a PDF rendering adapter.

## Consequences

Callers should request invoice PDF bytes from an invoice delivery or PDF renderer module. They should not need to know about temporary work directories, embedded template files, Typst source generation, or process execution.
