## ADDED Requirements

### Requirement: Database CRUD for entries
The database layer SHALL support creating, reading, updating, and deleting entry records.

#### Scenario: Create an entry and retrieve it
- **WHEN** an entry is created via CreateEntry with date="2024-03-15", category="Consulting", hours=4.5, notes="Client meeting"
- **THEN** GetEntry returns the entry with matching fields and a valid auto-assigned ID

#### Scenario: List entries returns all records
- **WHEN** three entries are created sequentially
- **THEN** ListEntries returns all three entries sorted by date descending

#### Scenario: Update an entry modifies it in place
- **WHEN** an entry is updated via UpdateEntry with new hours=6.0 and notes="Updated meeting"
- **THEN** GetEntry returns the entry with the updated values and unchanged ID/date/category

#### Scenario: Delete an entry removes it permanently
- **WHEN** an entry is deleted via DeleteEntry by its ID
- **THEN** subsequent ListEntries does not include it, and GetEntry returns sql.ErrNoRows

#### Scenario: Create entry with empty category fails validation
- **WHEN** CreateEntry is called with an empty category string
- **THEN** the operation returns an error

### Requirement: Database CRUD for rates
The database layer SHALL support creating, reading, updating, and deleting rate records.

#### Scenario: Create a rate and retrieve it
- **WHEN** a rate is created via CreateRate with category="Consulting", startDate="2024-01-01", endDate="", rate=150.00
- **THEN** GetRate returns the rate with matching fields and a valid auto-assigned ID

#### Scenario: List rates returns all records
- **WHEN** multiple rates are created for different categories
- **THEN** ListRates returns all rates

#### Scenario: Update a rate modifies it in place
- **WHEN** a rate is updated via UpdateRate with new rate=175.00
- **THEN** GetRate returns the rate with the updated value

#### Scenario: Delete a rate removes it permanently
- **WHEN** a rate is deleted via DeleteRate by its ID
- **THEN** subsequent ListRates does not include it, and GetRate returns sql.ErrNoRows

### Requirement: Database CRUD for invoices
The database layer SHALL support creating and retrieving invoice records with line items.

#### Scenario: Create an invoice with line items
- **WHEN** an invoice is created via CreateInvoice with month="2024-03", category="All", lines containing three InvoiceLine entries
- **THEN** GetInvoice returns the invoice with all line items populated

#### Scenario: List invoices returns summaries
- **WHEN** multiple invoices are created
- **THEN** ListInvoices returns InvoiceSummary records with correct totals

#### Scenario: CountUnratedEntries returns zero when all entries have rates
- **WHEN** all entries in a given month have associated rates
- **THEN** CountUnratedEntries returns 0

#### Scenario: CountUnratedEntries returns count for unrated entries
- **WHEN** some entries in a given month lack rates
- **THEN** CountUnratedEntries returns the number of entries without matching rates

### Requirement: Database settings persistence
The database layer SHALL support saving and retrieving application settings.

#### Scenario: Save settings and retrieve them
- **WHEN** Settings are saved via SaveSettings with business name, address, and bank details
- **THEN** GetSettings returns all saved fields correctly

#### Scenario: Default settings when none saved
- **WHEN** no settings have been saved yet
- **THEN** GetSettings returns a zero-value Settings struct (all empty strings, zero ints)

### Requirement: Migration execution
The database layer SHALL execute all SQL migration files from the configured directory in sorted order.

#### Scenario: Fresh database runs all migrations
- **WHEN** Migrate is called on a newly opened database with migration files present
- **THEN** all tables (entries, rates, invoices, settings, etc.) are created without error

#### Scenario: Re-running migrations is idempotent
- **WHEN** Migrate is called twice on the same database
- **THEN** no errors occur (tables use IF NOT EXISTS)

### Requirement: HTTP handler — create entry
The entries handler SHALL accept POST requests to create new entries with validation.

#### Scenario: Valid entry creation returns redirect
- **WHEN** a POST to /entries is made with valid date, category, and hours
- **THEN** the response is a 303 redirect to /entries with a success notice

#### Scenario: Missing category returns 400
- **WHEN** a POST to /entries is made without a category field
- **THEN** the response status is 400 Bad Request

#### Scenario: Invalid hours returns 400
- **WHEN** a POST to /entries is made with hours="abc"
- **THEN** the response status is 400 Bad Request

#### Scenario: HTMX request returns partial HTML on update
- **WHEN** a POST to /entries/{id} is made with HX-Request header
- **THEN** the response contains an HTML partial (entries_table) instead of a redirect

### Requirement: HTTP handler — delete entry
The entries handler SHALL accept POST requests to delete entries.

#### Scenario: Delete existing entry returns redirect
- **WHEN** a POST to /entries/{id}/delete is made for an existing entry
- **THEN** the response is a 303 redirect to /entries with a success notice

#### Scenario: Delete non-existent entry returns error
- **WHEN** a POST to /entries/99999/delete is made for a non-existent entry
- **THEN** the response is an error (500)

### Requirement: HTTP handler — rates page
The rates handler SHALL serve rate management endpoints.

#### Scenario: GET /rates returns 200 with rate list
- **WHEN** a GET request is made to /rates
- **THEN** the response status is 200 and the body contains rate data

#### Scenario: POST /rates creates a new rate
- **WHEN** a POST to /rates is made with valid category, start date, and rate
- **THEN** the response is a 303 redirect to /rates with a success notice

### Requirement: HTTP handler — invoices page
The invoices handler SHALL serve invoice listing and generation endpoints.

#### Scenario: GET /invoices returns 200 with invoice list
- **WHEN** a GET request is made to /invoices
- **THEN** the response status is 200 and the body contains invoice data or an empty-state message

#### Scenario: POST /invoices/generate creates an invoice when rates exist
- **WHEN** a POST to /invoices/generate is made with valid month, category, and due days — and all entries have rates
- **THEN** the response is a 303 redirect to the new invoice preview page

#### Scenario: POST /invoices/generate fails when entries lack rates
- **WHEN** a POST to /invoices/generate is made but some entries in the selected month have no rate
- **THEN** the response is an error page with a message about missing rates

### Requirement: HTTP handler — settings page
The settings handler SHALL serve application settings endpoints.

#### Scenario: GET /settings returns saved settings
- **WHEN** a GET request is made to /settings after settings have been saved
- **THEN** the response status is 200 and the body contains the saved values in form fields

#### Scenario: POST /settings saves new settings
- **WHEN** a POST to /settings is made with updated business information
- **THEN** the response is a 303 redirect to /settings with a success notice, and subsequent GET returns the updated values

### Requirement: HTTP middleware — recovery
The recoverMiddleware SHALL catch panics and return a 500 error.

#### Scenario: Handler panic returns 500
- **WHEN** a handler panics during request processing
- **THEN** the response status is 500 Internal Server Error with an error message, and the server continues serving subsequent requests

### Requirement: HTTP middleware — HTMX detection
The isHTMX function SHALL correctly detect HTMX requests.

#### Scenario: HX-Request header set to true identifies HTMX
- **WHEN** a request has header "HX-Request: true"
- **THEN** isHTMX returns true

#### Scenario: Missing HX-Request header is not HTMX
- **WHEN** a request has no HX-Request header
- **THEN** isHTMX returns false
