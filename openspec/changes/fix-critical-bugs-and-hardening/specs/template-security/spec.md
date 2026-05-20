## ADDED Requirements

### Requirement: Typst template content must be fully escaped

The system SHALL escape all Typst-special characters in user-controlled fields before embedding them into generated Typst template source. The characters that MUST be escaped are: backslash (`\`), double quote (`"`), hash (`#`), at sign (`@`), and newline characters.

#### Scenario: Business name with hash character
- **WHEN** a business name contains `#show: link("https://evil.com")`
- **THEN** the generated Typst source contains the escaped literal `\#show: link(\"https://evil.com\")` and does not execute as Typst code

#### Scenario: Customer name with at sign
- **WHEN** a customer name contains `@label`
- **THEN** the generated Typst source contains the escaped literal `\@label` and does not resolve as a Typst label reference

#### Scenario: Payment terms with backslash
- **WHEN** payment terms contain a literal backslash character
- **THEN** the generated Typst source contains a double backslash `\\` and renders as a single backslash in the PDF

#### Scenario: Address with newline
- **WHEN** a business address contains a newline character
- **THEN** the newline is escaped as `\n` in the Typst string literal and does not break the template syntax

#### Scenario: Item description with double quote
- **WHEN** an item description (category name) contains a double quote character
- **THEN** the generated Typst source contains `\"` and the PDF renders the quote correctly
