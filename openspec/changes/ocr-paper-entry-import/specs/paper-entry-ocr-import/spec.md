## ADDED Requirements

### Requirement: User can create an OCR import session from photographed paper notes
The system SHALL allow a user to upload one or more photographed paper timesheet images and create an OCR import session without creating final entries immediately.

#### Scenario: Upload creates draft import session
- **WHEN** the user submits one or more supported image files for paper-entry import
- **THEN** the system creates an import session in a non-final state and associates the uploaded images with that session

#### Scenario: Unsupported upload is rejected
- **WHEN** the user submits a file that is not a supported image type or cannot be processed
- **THEN** the system SHALL reject the upload and explain why the import session was not started

### Requirement: OCR import produces reviewable draft entries
The system SHALL convert OCR results into draft entries that can be reviewed and edited before persistence into the `entries` table.

#### Scenario: OCR results ready for review
- **WHEN** OCR processing completes successfully
- **THEN** the system SHALL present extracted draft entries with date, category, hours, notes, and confidence or ambiguity indicators

#### Scenario: User confirms reviewed entries
- **WHEN** the user confirms one or more draft entries after review
- **THEN** the system SHALL create normal persisted entries only for the confirmed drafts

### Requirement: OCR import uses known rates as category hints
The system SHALL use existing rate/category data to improve category interpretation while preserving the original OCR result when confidence is low.

#### Scenario: OCR text matches known category closely
- **WHEN** OCR processing returns category text that closely matches a known rate category
- **THEN** the system SHALL suggest the known category as the preferred match for that draft entry

#### Scenario: OCR text is ambiguous
- **WHEN** OCR processing cannot confidently match category text to a known rate category
- **THEN** the system SHALL show the raw OCR text and mark the draft entry as needing user review

### Requirement: OCR import exposes processing failures without losing the session
The system SHALL preserve the import session and report actionable errors when OCR processing fails.

#### Scenario: OCR backend fails
- **WHEN** the OCR service returns an error or times out
- **THEN** the system SHALL mark the import session as failed and provide an error message without deleting uploaded images or prior draft data
