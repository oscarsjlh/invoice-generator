## ADDED Requirements

### Requirement: Database error logging in handlers
The system SHALL log all database errors returned from store operations at `error` level with structured fields identifying the operation, relevant resource IDs, and the error message.

#### Scenario: Entry insert failure
- **WHEN** `InsertEntry` returns an error in the entry creation handler
- **THEN** an `error`-level log is written with `operation: "insert_entry"`, `error` (the error message), and relevant entry fields

#### Scenario: Invoice generation failure
- **WHEN** `GenerateInvoice` returns an error in the invoice generation handler
- **THEN** an `error`-level log is written with `operation: "generate_invoice"`, `month` (the invoice month), `category` (the category), and `error` (the error message)

#### Scenario: Rate update failure
- **WHEN** `UpdateRate` returns an error in the rate update handler
- **THEN** an `error`-level log is written with `operation: "update_rate"`, `rate_id`, and `error`

#### Scenario: Entry deletion failure
- **WHEN** `DeleteEntry` returns an error
- **THEN** an `error`-level log is written with `operation: "delete_entry"`, `entry_id`, and `error`

### Requirement: PDF generation error logging
The system SHALL log Typst PDF generation failures at `error` level with the invoice ID and error message.

#### Scenario: PDF generation failure
- **WHEN** `pdf.Generate()` returns an error
- **THEN** an `error`-level log is written with `operation: "generate_pdf"`, `invoice_id`, and `error`

### Requirement: Email send error logging
The system SHALL log SMTP email send failures at `error` level with the invoice ID, recipient address, and error message.

#### Scenario: Email send failure
- **WHEN** `email.Send()` returns an error
- **THEN** an `error`-level log is written with `operation: "send_email"`, `invoice_id`, `recipient`, and `error`

### Requirement: Validation error logging
The system SHALL log form validation errors at `warn` level with structured fields identifying the validation context and the error.

#### Scenario: Missing required field
- **WHEN** a form submission is missing a required field
- **THEN** a `warn`-level log is written with `operation` (e.g., `"create_entry"`), `validation_error` (the specific field), and the submitted form values (excluding sensitive data)

#### Scenario: Invalid rate value
- **WHEN** a rate form submission contains a non-numeric value
- **THEN** a `warn`-level log is written with `operation: "update_rate"` and `validation_error: "invalid_rate_value"`

### Requirement: No sensitive data in logs
The system SHALL NOT log email bodies, passwords, API keys, or full Typst stderr output. Error messages from underlying libraries SHALL be logged, but not the raw content that triggered them.

#### Scenario: Email send log excludes body
- **WHEN** an email send fails
- **THEN** the log entry does NOT contain the email body content

### Requirement: Info-level logging for successful operations
The system SHALL log successful resource mutations at `info` level to provide an audit trail of data changes.

#### Scenario: Entry created successfully
- **WHEN** an entry is created successfully
- **THEN** an `info`-level log is written with `operation: "create_entry"`, `entry_id`, and `category`

#### Scenario: Invoice created successfully
- **WHEN** an invoice is generated successfully
- **THEN** an `info`-level log is written with `operation: "generate_invoice"`, `invoice_id`, `invoice_number`, and `month`

#### Scenario: Entry deleted successfully
- **WHEN** an entry is deleted successfully
- **THEN** an `info`-level log is written with `operation: "delete_entry"` and `entry_id`
