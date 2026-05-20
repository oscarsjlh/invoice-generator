## ADDED Requirements

### Requirement: URL query parameters must use standard library escaping

The system SHALL use `net/url.QueryEscape` for all URL query parameter encoding. The custom `urlQueryEscape` function SHALL be removed.

#### Scenario: Notice message with special characters
- **WHEN** a redirect includes a notice message containing `&`, `=`, `+`, or `%` characters
- **THEN** the notice is properly URL-encoded in the redirect URL and decoded correctly on the target page

#### Scenario: Notice message with Unicode characters
- **WHEN** a redirect includes a notice message containing non-ASCII characters
- **THEN** the notice is properly percent-encoded in the redirect URL

### Requirement: HTTP error responses must not leak internal details

The system SHALL return generic error messages to clients for server-side errors. Internal error details (SQL errors, file paths, stack traces) SHALL be logged server-side but SHALL NOT be included in the HTTP response body.

#### Scenario: Database error returns generic message
- **WHEN** a database query fails with a SQL error
- **THEN** the HTTP response body contains a generic message like "Internal server error" and the detailed error is logged

#### Scenario: File operation error returns generic message
- **WHEN** a file operation fails (e.g., PDF generation, template parsing)
- **THEN** the HTTP response body contains a generic message and the detailed error is logged

### Requirement: Content-Disposition header values must be sanitized

The system SHALL strip or escape control characters (CR, LF, NUL) from values used in `Content-Disposition` headers. Invoice numbers and filenames used in download headers SHALL NOT contain characters that could inject additional HTTP headers.

#### Scenario: Invoice number with control characters
- **WHEN** an invoice number somehow contains a carriage return or line feed character
- **THEN** the control characters are stripped from the `Content-Disposition` header value

### Requirement: Dead code must be removed

The system SHALL not contain unused functions that could cause confusion or latent bugs. The `notFoundIfNoRows` function SHALL be removed as it passes `nil` as the ResponseWriter and would panic if called.

#### Scenario: Codebase contains no dead code paths
- **WHEN** a developer reviews the handler package
- **THEN** there are no functions that are defined but never called, or that contain obvious bugs (like passing `nil` to functions expecting a non-nil interface)
