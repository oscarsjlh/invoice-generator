# Invoice App Context

This file defines the domain language used by the application and architecture docs.

## Domain Terms

- **Entry**: A dated unit of billable work with a category, hours, and optional notes.
- **Rate**: A billable hourly rate for a category, optionally bounded by start and end dates.
- **Invoice**: A generated billing document for a month and category, built from entries and rates.
- **Invoice delivery**: The workflow that renders an invoice PDF and optionally sends it to the configured customer by email.
- **Settings**: User-owned business, bank, payment, invoice, and customer defaults.
- **OCR import session**: A user workflow for uploading paper timesheet images and turning extracted data into draft entries.
- **OCR draft entry**: A proposed entry produced by OCR that must be reviewed and confirmed before becoming a ledger entry.
- **User store**: A per-user SQLite store containing that user's invoice data.
- **Legacy store mode**: The no-auth compatibility mode where every request uses one shared store.

## Architecture Language

- **Module**: Anything with an interface and implementation, including packages, structs, and functions.
- **Interface**: Everything a caller must know to use a module, including types, invariants, errors, ordering, and config.
- **Implementation**: The code hidden behind an interface.
- **Depth**: Leverage at the interface; a deep module hides substantial behavior behind a small interface.
- **Seam**: A place where behavior can be altered without editing callers in place.
- **Adapter**: A concrete implementation of an interface at a seam.
- **Locality**: The degree to which related behavior, bugs, and changes are concentrated in one place.
