## ADDED Requirements

### Requirement: Structured request logging
The system SHALL log every HTTP request with structured fields including method, path, status code, duration, remote address, user agent, and request ID. The log entry SHALL be written at `info` level after the response is sent.

#### Scenario: Successful GET request
- **WHEN** a client sends `GET /entries` and the handler returns 200
- **THEN** an `info`-level log is written with `method: "GET"`, `path: "/entries"`, `status: 200`, `duration` in milliseconds, and `request_id`

#### Scenario: Failed POST request
- **WHEN** a client sends `POST /invoices/generate` and the handler returns 500
- **THEN** an `info`-level log is written with `status: 500` alongside the error details logged separately by the handler

#### Scenario: HTMX request detection
- **WHEN** a request includes the `HX-Request` header
- **THEN** the request log entry includes `hx_request: true` and `hx_target` (if `HX-Target` header is present)

#### Scenario: Response size tracking
- **WHEN** a response is sent
- **THEN** the log entry includes `bytes_written` (the response body size in bytes)

### Requirement: Request logging middleware ordering
The system SHALL place `RequestLoggingMiddleware` after `LoggerMiddleware` in the middleware chain so that the request logger has access to the context-injected logger with request ID.

#### Scenario: Middleware chain order
- **WHEN** the middleware chain is assembled
- **THEN** `LoggerMiddleware` wraps `RequestLoggingMiddleware`, which wraps the mux

### Requirement: Response writer wrapping
The system SHALL wrap `http.ResponseWriter` to capture the status code and bytes written for logging purposes. The wrapper SHALL implement `http.ResponseWriter`, `http.Flusher` (for streaming), and `http.Pusher` (if supported by the underlying writer).

#### Scenario: Status code capture
- **WHEN** a handler writes a 201 status code
- **THEN** the response writer wrapper records `201` for the log entry

#### Scenario: Flusher support
- **WHEN** a handler calls `Flush()` on the response writer
- **THEN** the wrapper delegates to the underlying writer's `Flush()` method

### Requirement: Duration measurement
The request logging middleware SHALL measure request duration from entry to response completion with millisecond precision.

#### Scenario: Duration logged
- **WHEN** a request takes 42ms to process
- **THEN** the log entry includes `duration_ms: 42` (or similar duration field)
