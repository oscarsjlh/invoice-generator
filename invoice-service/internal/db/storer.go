package db

import "database/sql"

type EntryStore interface {
	ListEntries() ([]Entry, error)
	ListEntriesFiltered(year, month string) ([]Entry, error)
	CreateEntry(date, category string, hours float64, notes string) error
	GetEntry(id int64) (Entry, error)
	UpdateEntry(id int64, date, category string, hours float64, notes string) error
	DeleteEntry(id int64) error
}

type RateStore interface {
	ListRates() ([]Rate, error)
	ListActiveRates(today string) ([]Rate, error)
	CreateRate(category, startDate, endDate string, rate float64) error
	DeleteRate(id int64) error
	GetRate(id int64) (Rate, error)
	UpdateRate(id int64, category, startDate, endDate string, rate float64) error
}

type InvoiceStore interface {
	ListMonthlySummary() ([]MonthlySummary, error)
	ListAvailableYears() ([]string, error)
	ListAvailableMonths(year string) ([]string, error)
	FilteredSummary(year, month string) ([]MonthlySummary, float64, float64, error)
	ListInvoices() ([]InvoiceSummary, error)
	CountUnratedEntries(month, category string) (int, error)
	GenerateInvoice(month, category, invoiceDate string, dueDays int, settings Settings) (int64, error)
	GetInvoice(id int64) (Invoice, error)
}

type CategoryStore interface {
	ListCategories() ([]string, error)
}

type SettingsStore interface {
	LoadSettings() (Settings, error)
	SaveSettings(settings Settings) error
}

type OCRStore interface {
	CreateOCRSession() (int64, error)
	GetOCRSession(id int64) (OCRSession, error)
	UpdateOCRSessionState(id int64, state string, errorMsg string) error
	ListOCRSessions() ([]OCRSession, error)
	AddSessionImage(sessionID int64, fileName, filePath string, fileSize int64) error
	GetSessionImages(sessionID int64) ([]OCRSessionImage, error)
	SaveDraftEntries(sessionID int64, drafts []OCRDraftEntry) error
	GetDraftEntries(sessionID int64) ([]OCRDraftEntry, error)
	GetDraftEntry(id int64) (OCRDraftEntry, error)
	UpdateDraftEntry(id int64, date, category, hours, notes string) error
	DeleteDraftEntry(sessionID, draftID int64) error
	ConfirmDraftEntries(sessionID int64, ids []int64) (confirmed []int64, skipped []int64, err error)
	DeleteOCRSession(id int64) error
	CleanupStaleOCRSessions(maxAgeHours int) (int, error)
}

type StoreLifecycle interface {
	DB() *sql.DB
	Close() error
	Migrate(dir string) error
}

// Storer is the compatibility interface for handlers that still need multiple
// persistence capabilities. New workflow modules should depend on the smallest
// interface they need instead of this composed surface.
type Storer interface {
	EntryStore
	RateStore
	InvoiceStore
	CategoryStore
	SettingsStore
	OCRStore
	StoreLifecycle
}
