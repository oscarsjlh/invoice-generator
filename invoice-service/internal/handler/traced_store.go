package handler

import (
	"context"
	"database/sql"

	"invoice-app/internal/db"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type tracedStore struct {
	ctx  context.Context
	next db.Storer
}

func newTracedStore(ctx context.Context, next db.Storer) db.Storer {
	return &tracedStore{ctx: ctx, next: next}
}

func (s *tracedStore) ListEntries() ([]db.Entry, error) {
	return traceDB(s.ctx, "entries.list", s.next.ListEntries)
}

func (s *tracedStore) ListEntriesFiltered(year, month, rate string) ([]db.Entry, error) {
	return traceDB(s.ctx, "entries.filtered.list", func() ([]db.Entry, error) {
		return s.next.ListEntriesFiltered(year, month, rate)
	})
}

func (s *tracedStore) CreateEntry(date, category string, hours float64, notes string) error {
	return traceDBErr(s.ctx, "entries.create", func() error {
		return s.next.CreateEntry(date, category, hours, notes)
	})
}

func (s *tracedStore) GetEntry(id int64) (db.Entry, error) {
	return traceDB(s.ctx, "entries.get", func() (db.Entry, error) {
		return s.next.GetEntry(id)
	})
}

func (s *tracedStore) UpdateEntry(id int64, date, category string, hours float64, notes string) error {
	return traceDBErr(s.ctx, "entries.update", func() error {
		return s.next.UpdateEntry(id, date, category, hours, notes)
	})
}

func (s *tracedStore) DeleteEntry(id int64) error {
	return traceDBErr(s.ctx, "entries.delete", func() error {
		return s.next.DeleteEntry(id)
	})
}

func (s *tracedStore) ListRates() ([]db.Rate, error) {
	return traceDB(s.ctx, "rates.list", s.next.ListRates)
}

func (s *tracedStore) ListActiveRates(today string) ([]db.Rate, error) {
	return traceDB(s.ctx, "rates.list_active", func() ([]db.Rate, error) {
		return s.next.ListActiveRates(today)
	})
}

func (s *tracedStore) CreateRate(category, startDate, endDate string, rate float64) error {
	return traceDBErr(s.ctx, "rates.create", func() error {
		return s.next.CreateRate(category, startDate, endDate, rate)
	})
}

func (s *tracedStore) DeleteRate(id int64) error {
	return traceDBErr(s.ctx, "rates.delete", func() error {
		return s.next.DeleteRate(id)
	})
}

func (s *tracedStore) GetRate(id int64) (db.Rate, error) {
	return traceDB(s.ctx, "rates.get", func() (db.Rate, error) {
		return s.next.GetRate(id)
	})
}

func (s *tracedStore) UpdateRate(id int64, category, startDate, endDate string, rate float64) error {
	return traceDBErr(s.ctx, "rates.update", func() error {
		return s.next.UpdateRate(id, category, startDate, endDate, rate)
	})
}

func (s *tracedStore) ListMonthlySummary() ([]db.MonthlySummary, error) {
	return traceDB(s.ctx, "summary.monthly.list", s.next.ListMonthlySummary)
}

func (s *tracedStore) ListAvailableYears() ([]string, error) {
	return traceDB(s.ctx, "entries.years.list", s.next.ListAvailableYears)
}

func (s *tracedStore) ListAvailableMonths(year string) ([]string, error) {
	return traceDB(s.ctx, "entries.months.list", func() ([]string, error) {
		return s.next.ListAvailableMonths(year)
	})
}

func (s *tracedStore) FilteredSummary(year, month string) ([]db.MonthlySummary, float64, float64, error) {
	_, span := startDBSpan(s.ctx, "summary.filtered")
	defer span.End()
	summary, totalHours, totalAmount, err := s.next.FilteredSummary(year, month)
	if err != nil {
		recordTraceError(span, err)
	}
	return summary, totalHours, totalAmount, err
}

func (s *tracedStore) ListInvoices() ([]db.InvoiceSummary, error) {
	return traceDB(s.ctx, "invoices.list", s.next.ListInvoices)
}

func (s *tracedStore) CountUnratedEntries(month, category string) (int, error) {
	return traceDB(s.ctx, "entries.unrated.count", func() (int, error) {
		return s.next.CountUnratedEntries(month, category)
	})
}

func (s *tracedStore) CountUnratedEntriesFiltered(year, month string) (int, error) {
	return traceDB(s.ctx, "entries.unrated.count_filtered", func() (int, error) {
		return s.next.CountUnratedEntriesFiltered(year, month)
	})
}

func (s *tracedStore) GenerateInvoice(month, category, invoiceDate string, dueDays int, invoiceNumber string, settings db.Settings) (int64, error) {
	return traceDB(s.ctx, "invoices.generate", func() (int64, error) {
		return s.next.GenerateInvoice(month, category, invoiceDate, dueDays, invoiceNumber, settings)
	})
}

func (s *tracedStore) GetInvoice(id int64) (db.Invoice, error) {
	return traceDB(s.ctx, "invoices.get", func() (db.Invoice, error) {
		return s.next.GetInvoice(id)
	})
}

func (s *tracedStore) DeleteInvoice(id int64) error {
	return traceDBErr(s.ctx, "invoices.delete", func() error {
		return s.next.DeleteInvoice(id)
	})
}

func (s *tracedStore) ListCategories() ([]string, error) {
	return traceDB(s.ctx, "categories.list", s.next.ListCategories)
}

func (s *tracedStore) LoadSettings() (db.Settings, error) {
	return traceDB(s.ctx, "settings.load", s.next.LoadSettings)
}

func (s *tracedStore) SaveSettings(settings db.Settings) error {
	return traceDBErr(s.ctx, "settings.save", func() error {
		return s.next.SaveSettings(settings)
	})
}

func (s *tracedStore) CreateOCRSession() (int64, error) {
	return traceDB(s.ctx, "ocr_sessions.create", s.next.CreateOCRSession)
}

func (s *tracedStore) GetOCRSession(id int64) (db.OCRSession, error) {
	return traceDB(s.ctx, "ocr_sessions.get", func() (db.OCRSession, error) {
		return s.next.GetOCRSession(id)
	})
}

func (s *tracedStore) UpdateOCRSessionState(id int64, state string, errorMsg string) error {
	return traceDBErr(s.ctx, "ocr_sessions.state.update", func() error {
		return s.next.UpdateOCRSessionState(id, state, errorMsg)
	})
}

func (s *tracedStore) ListOCRSessions() ([]db.OCRSession, error) {
	return traceDB(s.ctx, "ocr_sessions.list", s.next.ListOCRSessions)
}

func (s *tracedStore) AddSessionImage(sessionID int64, fileName, filePath string, fileSize int64) error {
	return traceDBErr(s.ctx, "ocr_session_images.add", func() error {
		return s.next.AddSessionImage(sessionID, fileName, filePath, fileSize)
	})
}

func (s *tracedStore) GetSessionImages(sessionID int64) ([]db.OCRSessionImage, error) {
	return traceDB(s.ctx, "ocr_session_images.list", func() ([]db.OCRSessionImage, error) {
		return s.next.GetSessionImages(sessionID)
	})
}

func (s *tracedStore) SaveDraftEntries(sessionID int64, drafts []db.OCRDraftEntry) error {
	return traceDBErr(s.ctx, "ocr_drafts.save", func() error {
		return s.next.SaveDraftEntries(sessionID, drafts)
	})
}

func (s *tracedStore) GetDraftEntries(sessionID int64) ([]db.OCRDraftEntry, error) {
	return traceDB(s.ctx, "ocr_drafts.list", func() ([]db.OCRDraftEntry, error) {
		return s.next.GetDraftEntries(sessionID)
	})
}

func (s *tracedStore) GetDraftEntry(id int64) (db.OCRDraftEntry, error) {
	return traceDB(s.ctx, "ocr_drafts.get", func() (db.OCRDraftEntry, error) {
		return s.next.GetDraftEntry(id)
	})
}

func (s *tracedStore) UpdateDraftEntry(id int64, date, category, hours, notes string) error {
	return traceDBErr(s.ctx, "ocr_drafts.update", func() error {
		return s.next.UpdateDraftEntry(id, date, category, hours, notes)
	})
}

func (s *tracedStore) DeleteDraftEntry(sessionID, draftID int64) error {
	return traceDBErr(s.ctx, "ocr_drafts.delete", func() error {
		return s.next.DeleteDraftEntry(sessionID, draftID)
	})
}

func (s *tracedStore) ConfirmDraftEntries(sessionID int64, ids []int64) ([]int64, []int64, error) {
	_, span := startDBSpan(s.ctx, "ocr_drafts.confirm")
	defer span.End()
	confirmed, skipped, err := s.next.ConfirmDraftEntries(sessionID, ids)
	if err != nil {
		recordTraceError(span, err)
	}
	span.SetAttributes(
		attribute.Int("db.rows_confirmed", len(confirmed)),
		attribute.Int("db.rows_skipped", len(skipped)),
	)
	return confirmed, skipped, err
}

func (s *tracedStore) DeleteOCRSession(id int64) error {
	return traceDBErr(s.ctx, "ocr_sessions.delete", func() error {
		return s.next.DeleteOCRSession(id)
	})
}

func (s *tracedStore) CleanupStaleOCRSessions(maxAgeHours int) (int, error) {
	return traceDB(s.ctx, "ocr_sessions.cleanup", func() (int, error) {
		return s.next.CleanupStaleOCRSessions(maxAgeHours)
	})
}

func (s *tracedStore) DB() *sql.DB {
	return s.next.DB()
}

func (s *tracedStore) Close() error {
	return s.next.Close()
}

func (s *tracedStore) Migrate(dir string) error {
	return s.next.Migrate(dir)
}

func traceDB[T any](ctx context.Context, operation string, fn func() (T, error)) (T, error) {
	_, span := startDBSpan(ctx, operation)
	defer span.End()
	value, err := fn()
	if err != nil {
		recordTraceError(span, err)
	}
	return value, err
}

func traceDBErr(ctx context.Context, operation string, fn func() error) error {
	_, span := startDBSpan(ctx, operation)
	defer span.End()
	err := fn()
	if err != nil {
		recordTraceError(span, err)
	}
	return err
}

func startDBSpan(ctx context.Context, operation string) (context.Context, trace.Span) {
	return otel.Tracer("invoice-app/db").Start(ctx, "db."+operation, trace.WithAttributes(
		attribute.String("db.system.name", "sqlite"),
		attribute.String("db.operation.name", operation),
	))
}

func recordTraceError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}
