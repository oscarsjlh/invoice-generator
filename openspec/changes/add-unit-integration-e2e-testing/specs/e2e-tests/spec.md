## ADDED Requirements

### Requirement: Happy-path invoice generation workflow
The system SHALL support a complete invoice generation flow from entry creation through PDF availability.

#### Scenario: Create entries and generate an invoice
- **WHEN** a user creates three time entries for month "2024-03" across different categories, sets rates for each category, then submits the invoice generation form
- **THEN** the system creates an invoice with the correct total (sum of hours × rate per line), redirects to the invoice preview page, and the preview shows all line items

#### Scenario: Invoice PDF is generated successfully
- **WHEN** a user views an invoice preview page for a valid invoice
- **THEN** clicking "Download PDF" triggers a PDF response with correct content type (application/pdf) and the invoice details rendered in the document

### Requirement: Entry management workflow
The system SHALL support creating, editing, and deleting entries through the web interface.

#### Scenario: Create an entry via the web form
- **WHEN** a user submits the entry creation form with date="2024-03-15", category="Consulting", hours=4.5, notes="Client call"
- **THEN** the system saves the entry, redirects to /entries with a success notice, and the entry appears in the entries table

#### Scenario: Edit an entry via HTMX inline editing
- **WHEN** a user clicks "edit" on an existing entry row (HTMX request)
- **THEN** the system returns an HTML partial with an editable form row

#### Scenario: Delete an entry via HTMX action
- **WHEN** a user clicks "delete" on an existing entry row (HTMX request)
- **THEN** the system removes the entry and returns an updated entries table partial without the deleted row

### Requirement: Rate management workflow
The system SHALL support setting and updating rates for categories.

#### Scenario: Set a rate for a category
- **WHEN** a user submits the rate creation form with category="Consulting", start date="2024-01-01", rate=150.00
- **THEN** the system saves the rate, redirects to /rates with a success notice, and the rate appears in the rates table

#### Scenario: Update an existing rate
- **WHEN** a user edits an existing rate to change the value from 150.00 to 175.00
- **THEN** the system updates the rate and subsequent invoice generation uses the new rate value

### Requirement: Settings persistence workflow
The system SHALL support saving and retrieving application settings through the web interface.

#### Scenario: Save business and client settings
- **WHEN** a user fills in the settings form with business name, address, bank details, and client information
- **THEN** the system saves all fields, redirects back to /settings with a success notice, and subsequent page loads display the saved values

### Requirement: Multi-step invoice preview and send
The system SHALL support previewing an invoice and sending it via email (when SMTP is configured).

#### Scenario: Preview an invoice before generating PDF
- **WHEN** a user navigates to an invoice's preview page
- **THEN** the page displays the invoice number, month, category, all line items with hours/rate/amount, subtotal, total, and payment details (business info, client info, bank details)

#### Scenario: Generate PDF for an existing invoice
- **WHEN** a user requests the PDF endpoint for a valid invoice
- **THEN** the system returns the generated PDF file with HTTP 200 status

### Requirement: Dashboard summary workflow
The system SHALL display accurate financial summaries on the dashboard.

#### Scenario: Dashboard shows correct monthly summaries
- **WHEN** entries and rates exist for multiple months
- **THEN** the dashboard displays monthly summaries with correct total hours and total amounts per month, based on entries × their category rates

#### Scenario: Dashboard filters by year and month
- **WHEN** a user selects a specific year and month on the dashboard
- **THEN** the displayed summary and recent invoices reflect only the selected time period
