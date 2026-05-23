# ADR 0001: Per-User SQLite Stores

## Status

Accepted

## Context

The application stores invoice, entry, rate, settings, and OCR import data. Authentication can be enabled, and each authenticated user needs isolated invoice data while preserving a simpler local deployment model.

## Decision

Use a separate SQLite database per user for invoice data. Keep authentication data in a separate auth database. Preserve legacy no-auth mode through an adapter that resolves all requests to one shared store.

## Consequences

Per-user stores keep tenant data isolation simple and make backup/export of a single user's invoice data straightforward. Store resolution, migration, caching, and lifecycle management must be treated as an explicit module rather than leaking into handlers.
