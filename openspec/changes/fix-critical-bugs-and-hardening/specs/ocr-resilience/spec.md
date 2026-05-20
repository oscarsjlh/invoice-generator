## ADDED Requirements

### Requirement: OCR upload failures must clean up orphaned resources

The system SHALL delete the created OCR session and remove any uploaded files when an error occurs during the upload loop. Cleanup SHALL occur before returning the error response to the client.

#### Scenario: Upload failure cleans up session and files
- **WHEN** an image fails validation during the upload loop
- **THEN** the OCR session is deleted and all previously uploaded files for that session are removed from disk

#### Scenario: Successful upload leaves no orphaned files
- **WHEN** all images in an upload request pass validation and are saved
- **THEN** the session remains in the database with state `uploaded` and all files exist on disk

### Requirement: Background OCR processing must support graceful shutdown

The system SHALL track all background OCR processing goroutines using a `sync.WaitGroup`. The application SHALL wait for all in-flight goroutines to complete before exiting during shutdown.

#### Scenario: Server shutdown waits for OCR processing
- **WHEN** the server receives a shutdown signal while OCR processing is in progress
- **THEN** the server waits for the processing goroutine to complete before exiting

#### Scenario: Multiple concurrent OCR sessions complete on shutdown
- **WHEN** multiple OCR sessions are processing concurrently and the server shuts down
- **THEN** all processing goroutines complete before the process exits

### Requirement: ConfirmDraftEntries must report skipped entries

The system SHALL return a list of draft entry IDs that were skipped during confirmation (due to missing date, category, or zero hours) alongside the list of confirmed IDs. The handler SHALL display a notice to the user indicating how many entries were confirmed and how many were skipped.

#### Scenario: All drafts are valid
- **WHEN** a user confirms 5 draft entries that all have valid date, category, and hours
- **THEN** all 5 entries are created and the notice shows "Confirmed 5 entries"

#### Scenario: Some drafts are invalid
- **WHEN** a user confirms 5 draft entries where 2 have missing dates
- **THEN** 3 entries are created and the notice shows "Confirmed 3 entries, 2 skipped (missing data)"
