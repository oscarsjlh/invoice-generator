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
