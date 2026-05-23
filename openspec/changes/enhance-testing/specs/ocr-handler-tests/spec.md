## ADDED Requirements

### Requirement: OCR handler tests use mock client
The system SHALL have tests for `ocrUploadPage`, `ocrStartSession`, `ocrSessionStatus`, `ocrConfirmDrafts`, and `ocrDeleteSession` using a mock OCR client interface.

#### Scenario: OCR upload page renders successfully
- **WHEN** `ocrUploadPage` is called
- **THEN** the response is 200 OK containing upload form content

#### Scenario: OCR confirm drafts page renders
- **WHEN** `ocrConfirmDrafts` is called with a valid session ID
- **THEN** the response is 200 OK
