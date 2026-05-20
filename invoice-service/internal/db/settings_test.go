package db_test

import (
	"testing"

	"invoice-app/internal/db"
	"invoice-app/internal/testutil"
)

func TestSaveAndLoadSettings(t *testing.T) {
	t.Parallel()
	store := testutil.NewTestDB(t)

	settings := db.Settings{
		BusinessName:    "Test Business Ltd",
		BusinessAddress: "123 Test St\nLondon\nSW1A 1AA",
		BankName:        "Test Bank",
		AccountName:     "Test Account",
		AccountNumber:   "12345678",
		SortCode:        "12-34-56",
		PaymentTerms:    "Net 30.",
		DefaultDueDays:  45,
		CustomerName:    "John Doe",
		CustomerEmail:   "john@example.com",
	}

	err := store.SaveSettings(settings)
	if err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	loaded, err := store.LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}

	checks := []struct {
		name string
		got  string
		want string
	}{
		{"BusinessName", loaded.BusinessName, "Test Business Ltd"},
		{"BusinessAddress", loaded.BusinessAddress, "123 Test St\nLondon\nSW1A 1AA"},
		{"BankName", loaded.BankName, "Test Bank"},
		{"AccountName", loaded.AccountName, "Test Account"},
		{"AccountNumber", loaded.AccountNumber, "12345678"},
		{"SortCode", loaded.SortCode, "12-34-56"},
		{"PaymentTerms", loaded.PaymentTerms, "Net 30."},
		{"CustomerName", loaded.CustomerName, "John Doe"},
		{"CustomerEmail", loaded.CustomerEmail, "john@example.com"},
	}

	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}

	if loaded.DefaultDueDays != 45 {
		t.Errorf("DefaultDueDays = %d, want 45", loaded.DefaultDueDays)
	}
}

func TestLoadSettingsReturnsDefaultsWhenNoneSaved(t *testing.T) {
	t.Parallel()
	store := testutil.NewTestDB(t)

	// The migration inserts default settings with empty values.
	// LoadSettings returns a zero-value struct except for DefaultDueDays (30).
	settings, err := store.LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}

	if settings.DefaultDueDays != 30 {
		t.Errorf("DefaultDueDays = %d, want 30", settings.DefaultDueDays)
	}
	// Business fields should be empty strings (set by migration)
	if settings.BusinessName != "" {
		t.Errorf("BusinessName = %q, want empty", settings.BusinessName)
	}
}
