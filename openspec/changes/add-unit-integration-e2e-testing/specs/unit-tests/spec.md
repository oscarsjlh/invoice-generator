## ADDED Requirements

### Requirement: Parse positive float validation
The `parsePositiveFloat` function SHALL correctly parse positive floating-point numbers and reject invalid input.

#### Scenario: Valid positive float parses successfully
- **WHEN** the string "123.45" is passed to parsePositiveFloat
- **THEN** it returns 123.45 and nil error

#### Scenario: Zero is rejected
- **WHEN** the string "0" is passed to parsePositiveFloat
- **THEN** it returns 0 and an error indicating the value must be greater than zero

#### Scenario: Negative number is rejected
- **WHEN** the string "-5.0" is passed to parsePositiveFloat
- **THEN** it returns 0 and an error indicating the value must be greater than zero

#### Scenario: Non-numeric string is rejected
- **WHEN** the string "abc" is passed to parsePositiveFloat
- **THEN** it returns 0 and a parsing error

#### Scenario: Whitespace is trimmed before parsing
- **WHEN** the string " 10.5 " is passed to parsePositiveFloat
- **THEN** it returns 10.5 and nil error

### Requirement: Parse positive int validation
The `parsePositiveInt` function SHALL correctly parse positive integers and reject invalid input.

#### Scenario: Valid positive int parses successfully
- **WHEN** the string "7" is passed to parsePositiveInt
- **THEN** it returns 7 and nil error

#### Scenario: Zero is rejected
- **WHEN** the string "0" is passed to parsePositiveInt
- **THEN** it returns 0 and an error indicating the value must be greater than zero

#### Scenario: Non-numeric string is rejected
- **WHEN** the string "xyz" is passed to parsePositiveInt
- **THEN** it returns 0 and a parsing error

### Requirement: Date validation
The `validateDate` function SHALL accept only YYYY-MM-DD format dates.

#### Scenario: Valid date passes through
- **WHEN** "2024-03-15" is passed to validateDate
- **THEN** it returns "2024-03-15" and nil error

#### Scenario: Invalid date format is rejected
- **WHEN** "15/03/2024" is passed to validateDate
- **THEN** it returns an empty string and a parsing error

#### Scenario: Whitespace is trimmed
- **WHEN** " 2024-03-15 " is passed to validateDate
- **THEN** it returns "2024-03-15" and nil error

### Requirement: Month validation
The `validateMonth` function SHALL accept only YYYY-MM format months.

#### Scenario: Valid month passes through
- **WHEN** "2024-03" is passed to validateMonth
- **THEN** it returns "2024-03" and nil error

#### Scenario: Full date format is rejected
- **WHEN** "2024-03-15" is passed to validateMonth
- **THEN** it returns an empty string and a parsing error

### Requirement: Category normalization
The `normalizeCategory` function SHALL normalize category strings.

#### Scenario: Non-empty category returns trimmed value
- **WHEN** " Consulting " is passed to normalizeCategory
- **THEN** it returns "Consulting"

#### Scenario: Empty string returns default
- **WHEN** "" is passed to normalizeCategory
- **THEN** it returns "All"

### Requirement: Number formatting with commas
The `formatWithCommas` function SHALL format integers with thousand separators.

#### Scenario: Small numbers pass through unchanged
- **WHEN** 42 is passed to formatWithCommas
- **THEN** it returns "42"

#### Scenario: Thousands are separated
- **WHEN** 1000 is passed to formatWithCommas
- **THEN** it returns "1,000"

#### Scenario: Millions are separated
- **WHEN** 1234567 is passed to formatWithCommas
- **THEN** it returns "1,234,567"

#### Scenario: Negative numbers preserve sign
- **WHEN** -9999 is passed to formatWithCommas
- **THEN** it returns "-9,999"

### Requirement: Money formatting
The `money` function SHALL format floats as two-decimal-place strings.

#### Scenario: Whole number formats with .00
- **WHEN** 100.0 is passed to money
- **THEN** it returns "100.00"

#### Scenario: Decimal number preserves precision
- **WHEN** 45.67 is passed to money
- **THEN** it returns "45.67"

### Requirement: Numfmt formatting
The `numfmt` function SHALL format floats with comma-separated thousands and two decimal places.

#### Scenario: Simple decimal formats correctly
- **WHEN** 45.50 is passed to numfmt
- **THEN** it returns "45.50"

#### Scenario: Large number formats with commas
- **WHEN** 1234.50 is passed to numfmt
- **THEN** it returns "1,234.50"

### Requirement: Date label formatting
The `dateLabel` function SHALL convert date strings to human-readable format.

#### Scenario: YYYY-MM-DD format converts correctly
- **WHEN** "2024-03-15" is passed to dateLabel
- **THEN** it returns "15 Mar 2024"

#### Scenario: RFC3339 format converts correctly
- **WHEN** "2024-03-15T10:30:00Z" is passed to dateLabel
- **THEN** it returns "15 Mar 2024"

#### Scenario: Unparseable string passes through unchanged
- **WHEN** "not-a-date" is passed to dateLabel
- **THEN** it returns "not-a-date"

### Requirement: URL query escaping
The `urlQueryEscape` function SHALL escape special characters for use in URL query parameters.

#### Scenario: Spaces are escaped
- **WHEN** "hello world" is passed to urlQueryEscape
- **THEN** it returns "hello%20world"

#### Scenario: Ampersands are escaped
- **WHEN** "a&b" is passed to urlQueryEscape
- **THEN** it returns "a%26b"

#### Scenario: Percent signs are double-escaped
- **WHEN** "100%" is passed to urlQueryEscape
- **THEN** it returns "100%25"

### Requirement: Config loading with environment variables
The config package SHALL load configuration from environment variables with documented defaults.

#### Scenario: Default values are used when env vars are unset
- **WHEN** no relevant environment variables are set
- **THEN** the loaded config returns Address=":8080", LogLevel="info", LogFormat="json"

#### Scenario: Environment variables override defaults
- **WHEN** ADDRESS=":9090" and LOG_LEVEL="debug" are set
- **THEN** the loaded config returns Address=":9090", LogLevel="debug"

#### Scenario: OCR_ENABLED parses boolean correctly
- **WHEN** OCR_ENABLED="true" is set
- **THEN** the loaded config returns OCREnabled=true

#### Scenario: OCR_ENABLED false or unset disables OCR
- **WHEN** OCR_ENABLED="" (empty) is set
- **THEN** the loaded config returns OCREnabled=false
