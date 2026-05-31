package ocr

import "time"

type ImportSessionState string

const (
	StateUploaded      ImportSessionState = "uploaded"
	StateProcessing    ImportSessionState = "processing"
	StateReviewReady   ImportSessionState = "review_ready"
	StateFailed        ImportSessionState = "failed"
	StateConfirmed     ImportSessionState = "confirmed"
	StateConfirmedPart ImportSessionState = "confirmed_part"
)

type ContextHint struct {
	Categories []string `json:"categories"`
	SendRates  bool     `json:"send_rates"`
}

type RateHint struct {
	Category  string  `json:"category"`
	StartDate string  `json:"start_date"`
	EndDate   string  `json:"end_date"`
	Rate      float64 `json:"rate"`
}

type OCRRequest struct {
	Hints       ContextHint `json:"hints"`
	Rates       []RateHint  `json:"rates,omitempty"`
	SessionID   int64       `json:"session_id"`
	CurrentYear int         `json:"current_year"`
}

type OCRExtractedField struct {
	Raw         string   `json:"raw"`
	Normalized  string   `json:"normalized"`
	Candidates  []string `json:"candidates,omitempty"`
	Confidence  float64  `json:"confidence"`
	NeedsReview bool     `json:"needs_review"`
}

type OCRExtractedEntry struct {
	Date     OCRExtractedField `json:"date"`
	Category OCRExtractedField `json:"category"`
	Hours    OCRExtractedField `json:"hours"`
	Notes    OCRExtractedField `json:"notes"`
}

type OCRResponse struct {
	SessionID int64               `json:"session_id"`
	Status    string              `json:"status"`
	Error     string              `json:"error,omitempty"`
	Entries   []OCRExtractedEntry `json:"entries,omitempty"`
	Metadata  OCRMetadata         `json:"metadata"`
}

type OCRMetadata struct {
	ModelUsed        string  `json:"model_used"`
	ProcessingTimeMs int64   `json:"processing_time_ms"`
	PagesProcessed   int     `json:"pages_processed"`
	AvgConfidence    float64 `json:"avg_confidence"`
}

type DraftEntry struct {
	ID                 int64   `json:"id"`
	SessionID          int64   `json:"session_id"`
	DateRaw            string  `json:"date_raw"`
	DateNormalized     string  `json:"date_normalized"`
	CategoryRaw        string  `json:"category_raw"`
	CategoryNormalized string  `json:"category_normalized"`
	HoursRaw           string  `json:"hours_raw"`
	HoursNormalized    float64 `json:"hours_normalized"`
	NotesRaw           string  `json:"notes_raw"`
	NotesNormalized    string  `json:"notes_normalized"`
	Confidence         float64 `json:"confidence"`
	NeedsReview        bool    `json:"needs_review"`
	Confirmed          bool    `json:"confirmed"`
}

type ImportSession struct {
	ID        int64              `json:"id"`
	State     ImportSessionState `json:"state"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
	ErrorMsg  string             `json:"error_msg,omitempty"`
	Images    []SessionImage     `json:"images,omitempty"`
	Drafts    []DraftEntry       `json:"drafts,omitempty"`
}

type SessionImage struct {
	ID        int64  `json:"id"`
	SessionID int64  `json:"session_id"`
	FileName  string `json:"file_name"`
	FilePath  string `json:"file_path"`
	FileSize  int64  `json:"file_size"`
}
