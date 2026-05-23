## ADDED Requirements

### Requirement: Handler tests cover updateEntry
The system SHALL have tests for the `updateEntry` handler that verify successful updates return a redirect and validation errors return the edit form with errors.

#### Scenario: Successful update redirects to entries page
- **WHEN** `updateEntry` is called with valid form data (date, category, hours, notes) for an existing entry
- **THEN** the entry is updated in the database and the response is a redirect (303 See Other) to `/entries`

#### Scenario: Update with missing category returns form with error
- **WHEN** `updateEntry` is called with a blank category
- **THEN** the response is a 200 OK rendering the edit form with an error message

### Requirement: Handler tests cover editEntryForm
The system SHALL have tests for the `editEntryForm` handler that verify the edit form is rendered for valid entries and a 404 is returned for invalid entry IDs.

#### Scenario: Edit form for valid entry
- **WHEN** `editEntryForm` is called for an existing entry ID
- **THEN** the response is 200 OK with the edit form populated with the entry's current data

#### Scenario: Edit form for nonexistent entry
- **WHEN** `editEntryForm` is called for a nonexistent entry ID
- **THEN** the response is 404 Not Found

### Requirement: Handler tests cover updateRate
The system SHALL have tests for the `updateRate` handler that verify successful updates return a redirect and validation errors return the edit form with errors.

#### Scenario: Successful rate update redirects to rates page
- **WHEN** `updateRate` is called with valid form data for an existing rate
- **THEN** the rate is updated in the database and the response is a redirect (303 See Other) to `/rates`

#### Scenario: Update with missing category returns form with error
- **WHEN** `updateRate` is called with a blank category
- **THEN** the response is a 200 OK rendering the edit form with an error message

### Requirement: Handler tests cover editRateForm
The system SHALL have tests for the `editRateForm` handler that verify the edit form is rendered for valid rates.

#### Scenario: Edit form for valid rate
- **WHEN** `editRateForm` is called for an existing rate ID
- **THEN** the response is 200 OK with the edit form populated with the rate's current data

#### Scenario: Edit form for nonexistent rate
- **WHEN** `editRateForm` is called for a nonexistent rate ID
- **THEN** the response is 404 Not Found

### Requirement: Handler tests cover dashboard
The system SHALL have a test for the `dashboard` handler that verifies the dashboard page renders successfully.

#### Scenario: Dashboard renders successfully
- **WHEN** `dashboard` is called with a store that has entries, rates, and invoices
- **THEN** the response is 200 OK and contains key dashboard content (e.g., "Dashboard" or entry summary text)
