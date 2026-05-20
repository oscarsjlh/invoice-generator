## ADDED Requirements

### Requirement: Templates must be compiled once at startup

The system SHALL parse and compile all Go HTML templates during application initialization. The compiled template set SHALL be stored on the application struct and reused for all subsequent requests.

#### Scenario: Application startup compiles templates
- **WHEN** the application starts
- **THEN** all template files are parsed and compiled before the HTTP server begins listening

#### Scenario: Page rendering uses cached templates
- **WHEN** a request is made for any page (dashboard, entries, rates, invoices, settings, OCR)
- **THEN** the response is rendered using the pre-compiled templates without re-parsing

#### Scenario: HTMX partial rendering uses cached templates
- **WHEN** an HTMX request is made for a partial (entries table, rates table, invoice list)
- **THEN** the partial is rendered using the pre-compiled templates without re-parsing

### Requirement: Template compilation errors prevent startup

The system SHALL fail to start if any template file cannot be parsed or compiled. The error message SHALL identify the specific file and parsing error.

#### Scenario: Missing template file prevents startup
- **WHEN** a template file referenced by the layout is missing from the embedded filesystem
- **THEN** the application exits with a non-zero status and logs the missing file name

#### Scenario: Invalid template syntax prevents startup
- **WHEN** a template file contains invalid Go template syntax
- **THEN** the application exits with a non-zero status and logs the parse error
