# ADR 0004: WebAuthn Authentication

## Status

Accepted

## Context

The application supports authenticated multi-user operation without password storage.

## Decision

Use WebAuthn for user registration and login. Store authentication data separately from per-user invoice data.

## Consequences

Authentication modules should remain separate from invoice workflow modules. Request handling should attach an authenticated user identity and let a store provider resolve the correct user store.
