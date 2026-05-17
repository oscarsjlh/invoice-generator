## ADDED Requirements

### Requirement: OCR runtime assessment records deployment feasibility
The system SHALL document whether the selected OCR stack can run locally, requires hosted execution, or supports multiple deployment modes.

#### Scenario: Local runtime is feasible
- **WHEN** the OCR stack can be executed on supported local hardware
- **THEN** the assessment SHALL record the local runtime approach, required components, and operational constraints

#### Scenario: Local runtime is not feasible
- **WHEN** the OCR stack cannot be executed locally with acceptable performance or support
- **THEN** the assessment SHALL record the reason and the required hosted or remote deployment alternative

### Requirement: OCR runtime assessment captures hardware requirements
The system SHALL document the compute and memory requirements needed to run the selected OCR stack, including VRAM requirements when GPU inference is required.

#### Scenario: GPU-backed runtime is required
- **WHEN** the chosen OCR stack depends on GPU inference
- **THEN** the assessment SHALL include the expected VRAM range and any model-specific runtime assumptions

#### Scenario: CPU or reduced local mode is available
- **WHEN** the chosen OCR stack supports a lower-resource local mode
- **THEN** the assessment SHALL describe its performance or accuracy trade-offs

### Requirement: OCR service contract accepts contextual category hints
The OCR service SHALL accept structured context from the invoice app so category and rate information can influence extraction.

#### Scenario: Request includes known categories
- **WHEN** the invoice app submits OCR work for a paper-entry import session
- **THEN** the OCR service SHALL accept known categories or rate metadata alongside the uploaded images

#### Scenario: OCR service returns normalized and raw values
- **WHEN** the OCR service completes extraction
- **THEN** it SHALL return both normalized candidate values and raw recognized text for fields that were influenced by contextual hints
