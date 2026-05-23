## ADDED Requirements

### Requirement: SendInvoice accepts a Dialer interface
The system SHALL accept a `Dialer` interface as a parameter to `SendInvoice` that has a `DialAndSend` method matching `gomail.Dialer.DialAndSend`. The system SHALL provide a default dialer constructor for production use.

#### Scenario: SendInvoice with mock dialer
- **WHEN** `SendInvoice` is called with a mock dialer
- **THEN** `DialAndSend` is called on the mock with the composed email message (correct from, to, subject, body, and PDF attachment)

#### Scenario: SendInvoice constructs correct email
- **WHEN** `SendInvoice` is called with recipient, name, sender, invoice number, and PDF data
- **THEN** the composed message has the correct `From`, `To`, `Subject` headers, plain text body containing the recipient's name and invoice number, and a PDF attachment with the correct filename

### Requirement: Email attachment is valid PDF content
The system SHALL attach the provided PDF bytes to the email message with the filename `{invoiceNumber}.pdf`.

#### Scenario: PDF attachment included
- **WHEN** `SendInvoice` is called with PDF data bytes
- **THEN** the message includes an attachment with the filename matching the invoice number as a PDF
