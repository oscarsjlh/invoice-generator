package db

type Entry struct {
	ID       int64
	Date     string
	Category string
	Hours    float64
	Notes    string
}

type Rate struct {
	ID        int64
	Category  string
	StartDate string
	EndDate   string
	Rate      float64
}

type MonthlySummary struct {
	Category    string
	Month       string
	TotalHours  float64
	TotalAmount float64
}

type InvoiceSummary struct {
	ID            int64
	InvoiceNumber string
	Month         string
	Category      string
	Total         float64
	InvoiceDate   string
	DueDate       string
	CreatedAt     string
}

type InvoiceLine struct {
	ID       int64
	Category string
	Hours    float64
	Rate     float64
	Amount   float64
}

type Invoice struct {
	ID              int64
	InvoiceNumber   string
	Month           string
	Category        string
	InvoiceDate     string
	DueDate         string
	Subtotal        float64
	Total           float64
	BusinessName    string
	BusinessAddress string
	BankName        string
	AccountName     string
	AccountNumber   string
	SortCode        string
	PaymentTerms    string
	Lines           []InvoiceLine
}

type OCRSession struct {
	ID        int64
	State     string
	ErrorMsg  string
	CreatedAt string
	UpdatedAt string
}

type OCRSessionImage struct {
	ID        int64
	SessionID int64
	FileName  string
	FilePath  string
	FileSize  int64
}

type OCRDraftEntry struct {
	ID                 int64
	SessionID          int64
	DateRaw            string
	DateNormalized     string
	CategoryRaw        string
	CategoryNormalized string
	HoursRaw           string
	HoursNormalized    float64
	NotesRaw           string
	NotesNormalized    string
	Confidence         float64
	NeedsReview        bool
	Confirmed          bool
}

type DraftReviewData struct {
	Session    OCRSession
	Images     []OCRSessionImage
	Drafts     []OCRDraftEntry
	Categories []string
	Notice     string
}

type Settings struct {
	BusinessName       string
	BusinessAddress    string
	BankName           string
	AccountName        string
	AccountNumber      string
	SortCode           string
	PaymentTerms       string
	DefaultDueDays     int
	CustomerName       string
	CustomerTitle      string
	CustomerEmail      string
	CustomerAddress    string
	CustomerPostalCode string
	CustomerCity       string
}
