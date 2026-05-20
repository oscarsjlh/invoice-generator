## ADDED Requirements

### Requirement: Invoice generation must be atomic

The system SHALL execute the entire invoice generation flow — including the unrated entries check, invoice line query, invoice record insertion, and invoice line insertion — within a single database transaction. If any step fails, the transaction SHALL be rolled back and no partial invoice data SHALL persist.

#### Scenario: Successful invoice creation is atomic
- **WHEN** a user generates an invoice for a month/category with valid entries and rates
- **THEN** the invoice record and all invoice lines are committed together in a single transaction

#### Scenario: Unrated entries prevent invoice creation within transaction
- **WHEN** a user attempts to generate an invoice for a month/category where some entries lack a matching rate
- **THEN** the transaction is rolled back and an error is returned indicating the count of unrated entries

#### Scenario: Concurrent invoice generation is serialized
- **WHEN** two requests attempt to generate invoices for the same month/category simultaneously
- **THEN** one request completes and the second request generates a unique invoice number (no duplicate invoice numbers are created)

#### Scenario: No invoiceable entries prevents creation
- **WHEN** a user attempts to generate an invoice for a month/category with no matching entries
- **THEN** the transaction is rolled back and an error is returned

### Requirement: Invoice number uniqueness is enforced within the transaction

The system SHALL generate unique invoice numbers by checking for existence within the same transaction used for invoice insertion. If a collision is detected, a sequential suffix (`-2`, `-3`, etc.) SHALL be appended until a unique number is found.

#### Scenario: First invoice gets base number
- **WHEN** generating the first invoice for `2024-03` / `Consulting`
- **THEN** the invoice number is `INV-2024-03-CONSULTING`

#### Scenario: Duplicate invoice gets suffixed number
- **WHEN** generating a second invoice for `2024-03` / `Consulting` after one already exists
- **THEN** the invoice number is `INV-2024-03-CONSULTING-2`
