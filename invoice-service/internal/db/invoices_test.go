package db

import (
	"testing"
)

func TestGenerateInvoice(t *testing.T) {
	t.Parallel()
	store := setupTestDB(t)
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	if err := store.CreateEntry("2024-03-15", "Consulting", 4.5, ""); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}
	if err := store.CreateRate("Consulting", "2024-01-01", "", 150.00); err != nil {
		t.Fatalf("CreateRate: %v", err)
	}

	settings := Settings{
		BusinessName:    "Test Business Ltd",
		BusinessAddress: "123 Test St\nLondon\nSW1A 1AA",
		BankName:        "Test Bank",
		AccountName:     "Test Account",
		AccountNumber:   "12345678",
		SortCode:        "12-34-56",
		PaymentTerms:    "Payment due within 30 days.",
	}

	id, err := store.GenerateInvoice("2024-03", "All", "2024-04-01", 30, settings)
	if err != nil {
		t.Fatalf("GenerateInvoice: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive invoice ID, got %d", id)
	}

	invoice, err := store.GetInvoice(id)
	if err != nil {
		t.Fatalf("GetInvoice: %v", err)
	}
	if len(invoice.Lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(invoice.Lines))
	}
	if invoice.Total != 675.00 {
		t.Errorf("total = %v, want 675.00", invoice.Total)
	}
	if invoice.InvoiceNumber == "" {
		t.Error("InvoiceNumber should not be empty")
	}
}

func TestGenerateInvoiceFailsWhenUnrated(t *testing.T) {
	t.Parallel()
	store := setupTestDB(t)
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	if err := store.CreateEntry("2024-03-15", "Consulting", 4.5, ""); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}

	settings := Settings{BusinessName: "Test"}

	_, err := store.GenerateInvoice("2024-03", "All", "2024-04-01", 30, settings)
	if err == nil {
		t.Fatal("expected error when entries lack rates, got nil")
	}
}

func TestGetInvoiceWithLines(t *testing.T) {
	t.Parallel()
	store := setupTestDB(t)
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	if err := store.CreateEntry("2024-03-15", "Consulting", 4.5, ""); err != nil {
		t.Fatalf("CreateEntry 1: %v", err)
	}
	if err := store.CreateEntry("2024-03-16", "Design", 2.0, ""); err != nil {
		t.Fatalf("CreateEntry 2: %v", err)
	}
	if err := store.CreateRate("Consulting", "2024-01-01", "", 150.00); err != nil {
		t.Fatalf("CreateRate 1: %v", err)
	}
	if err := store.CreateRate("Design", "2024-01-01", "", 120.00); err != nil {
		t.Fatalf("CreateRate 2: %v", err)
	}

	settings := Settings{
		BusinessName:    "Test Business Ltd",
		BusinessAddress: "123 Test St\nLondon\nSW1A 1AA",
		BankName:        "Test Bank",
		AccountName:     "Test Account",
		AccountNumber:   "12345678",
		SortCode:        "12-34-56",
		PaymentTerms:    "Payment due within 30 days.",
	}

	id, err := store.GenerateInvoice("2024-03", "All", "2024-04-01", 30, settings)
	if err != nil {
		t.Fatalf("GenerateInvoice: %v", err)
	}

	invoice, err := store.GetInvoice(id)
	if err != nil {
		t.Fatalf("GetInvoice: %v", err)
	}
	if len(invoice.Lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(invoice.Lines))
	}
	if invoice.Total != 915.00 {
		t.Errorf("total = %v, want 915.00", invoice.Total)
	}
}

func TestListInvoices(t *testing.T) {
	t.Parallel()
	store := setupTestDB(t)
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	if err := store.CreateRate("Consulting", "2024-01-01", "", 150.00); err != nil {
		t.Fatalf("CreateRate: %v", err)
	}

	createInvoiceForMonth(t, store, "2024-03", 150.00)
	createInvoiceForMonth(t, store, "2024-02", 200.00)

	invoices, err := store.ListInvoices()
	if err != nil {
		t.Fatalf("ListInvoices: %v", err)
	}
	if len(invoices) != 2 {
		t.Fatalf("expected 2 invoices, got %d", len(invoices))
	}
	if invoices[0].Month != "2024-02" {
		t.Errorf("first invoice month = %q, want %q", invoices[0].Month, "2024-02")
	}
}

func TestCountUnratedEntries(t *testing.T) {
	t.Parallel()
	store := setupTestDB(t)
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	if err := store.CreateEntry("2024-03-15", "Consulting", 4.5, ""); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}
	if err := store.CreateRate("Consulting", "2024-01-01", "", 150.00); err != nil {
		t.Fatalf("CreateRate: %v", err)
	}

	count, err := store.CountUnratedEntries("2024-03", "All")
	if err != nil {
		t.Fatalf("CountUnratedEntries: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}

	if err := store.CreateEntry("2024-03-16", "Design", 2.0, ""); err != nil {
		t.Fatalf("CreateEntry 2: %v", err)
	}

	count, err = store.CountUnratedEntries("2024-03", "All")
	if err != nil {
		t.Fatalf("CountUnratedEntries: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestInvoiceNumberGeneration(t *testing.T) {
	t.Parallel()
	store := setupTestDB(t)
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	settings := Settings{
		BusinessName:    "Test Business Ltd",
		BusinessAddress: "123 Test St\nLondon\nSW1A 1AA",
		BankName:        "Test Bank",
		AccountName:     "Test Account",
		AccountNumber:   "12345678",
		SortCode:        "12-34-56",
		PaymentTerms:    "Payment due within 30 days.",
	}

	if err := store.CreateEntry("2024-03-15", "Consulting", 4.5, ""); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}
	if err := store.CreateRate("Consulting", "2024-01-01", "", 150.00); err != nil {
		t.Fatalf("CreateRate: %v", err)
	}

	id1, err := store.GenerateInvoice("2024-03", "Consulting", "2024-04-01", 30, settings)
	if err != nil {
		t.Fatalf("GenerateInvoice 1: %v", err)
	}
	id2, err := store.GenerateInvoice("2024-03", "Consulting", "2024-04-01", 30, settings)
	if err != nil {
		t.Fatalf("GenerateInvoice 2: %v", err)
	}

	invoice1, err := store.GetInvoice(id1)
	if err != nil {
		t.Fatalf("GetInvoice 1: %v", err)
	}
	invoice2, err := store.GetInvoice(id2)
	if err != nil {
		t.Fatalf("GetInvoice 2: %v", err)
	}

	if invoice1.InvoiceNumber == invoice2.InvoiceNumber {
		t.Errorf("invoice numbers should differ: %q vs %q", invoice1.InvoiceNumber, invoice2.InvoiceNumber)
	}
}

func TestListMonthlySummary(t *testing.T) {
	t.Parallel()
	store := setupTestDB(t)
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	if err := store.CreateEntry("2024-03-15", "Consulting", 4.5, ""); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}
	if err := store.CreateRate("Consulting", "2024-01-01", "", 150.00); err != nil {
		t.Fatalf("CreateRate: %v", err)
	}

	summary, err := store.ListMonthlySummary()
	if err != nil {
		t.Fatalf("ListMonthlySummary: %v", err)
	}
	if len(summary) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summary))
	}
	if summary[0].Month != "2024-03" || summary[0].Category != "Consulting" {
		t.Errorf("summary = %+v, want month=2024-03 category=Consulting", summary[0])
	}
	if summary[0].TotalAmount != 675.00 {
		t.Errorf("total_amount = %v, want 675.00", summary[0].TotalAmount)
	}
}

func TestFilteredSummary(t *testing.T) {
	t.Parallel()
	store := setupTestDB(t)
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	if err := store.CreateEntry("2024-03-15", "Consulting", 4.5, ""); err != nil {
		t.Fatalf("CreateEntry 1: %v", err)
	}
	if err := store.CreateEntry("2024-03-16", "Design", 2.0, ""); err != nil {
		t.Fatalf("CreateEntry 2: %v", err)
	}
	if err := store.CreateRate("Consulting", "2024-01-01", "", 150.00); err != nil {
		t.Fatalf("CreateRate 1: %v", err)
	}
	if err := store.CreateRate("Design", "2024-01-01", "", 120.00); err != nil {
		t.Fatalf("CreateRate 2: %v", err)
	}

	summary, totalHours, totalAmount, err := store.FilteredSummary("2024", "03")
	if err != nil {
		t.Fatalf("FilteredSummary: %v", err)
	}
	if len(summary) != 2 {
		t.Fatalf("expected 2 summary rows, got %d", len(summary))
	}
	if totalHours != 6.5 || totalAmount != 915.00 {
		t.Errorf("totalHours = %v, want 6.5; totalAmount = %v, want 915.00", totalHours, totalAmount)
	}

	// Filter for non-existent month
	noSummary, noHours, noAmount, err := store.FilteredSummary("2024", "99")
	if err != nil {
		t.Fatalf("FilteredSummary empty: %v", err)
	}
	if len(noSummary) != 0 || noHours != 0 || noAmount != 0 {
		t.Errorf("expected zero results for invalid month, got %d rows, %.2f hours, %.2f amount", len(noSummary), noHours, noAmount)
	}
}

func createInvoiceForMonth(t *testing.T, store *Store, month string, _ float64) {
	t.Helper()
	if err := store.CreateEntry(month+"-15", "Consulting", 4.5, ""); err != nil {
		t.Fatalf("CreateEntry for %s: %v", month, err)
	}
	settings := Settings{
		BusinessName:    "Test Business Ltd",
		BusinessAddress: "123 Test St\nLondon\nSW1A 1AA",
		BankName:        "Test Bank",
		AccountName:     "Test Account",
		AccountNumber:   "12345678",
		SortCode:        "12-34-56",
		PaymentTerms:    "Payment due within 30 days.",
	}
	_, err := store.GenerateInvoice(month, "All", "2024-04-01", 30, settings)
	if err != nil {
		t.Fatalf("GenerateInvoice for %s: %v", month, err)
	}
}
