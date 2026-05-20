package pdf

import (
	"strings"
	"testing"
)

func TestFormatInvoiceTyp(t *testing.T) {
	t.Parallel()
	data := InvoiceData{
		InvoiceNumber:      "INV-2024-03-TEST",
		InvoiceDate:        "2024-04-01",
		DueDate:            "2024-05-01",
		Month:              "2024-03",
		BusinessName:       "Test Business Ltd",
		Title:              "Director",
		Street:             "123 Test Street",
		City:               "London",
		PostalCode:         "SW1A 1AA",
		BankName:           "Test Bank",
		AccountName:        "Test Account",
		SortCode:           "12-34-56",
		AccountNumber:      "12345678",
		PaymentTerms:       "Payment due within 30 days.",
		CustomerName:       "John Doe",
		CustomerTitle:      "Manager",
		CustomerStreet:     "456 Client Road",
		CustomerCity:       "Manchester",
		CustomerPostalCode: "M1 1AA",
		Items: []ItemData{
			{Description: "Consulting", DurMin: 270, HourlyRate: 150.00},
		},
	}

	output := FormatInvoiceTyp(data)

	// Check that all key fields are present in the output
	checks := []string{
		"invoice-id:", "INV-2024-03-TEST",
		"issuing-date:", "2024-04-01",
		"delivery-date:", "2024-03-01",
		"due-date:", "2024-05-01",
		"name:", "Test Business Ltd",
		"title:", "Director",
		"bank:", "Test Bank",
		"account-name:", "Test Account",
		"sort-code:", "12-34-56",
		"account-number:", "12345678",
		"payment-terms:", "Payment due within 30 days.",
		"city:", "London",
		"postal-code:", "SW1A 1AA",
		"street:", "123 Test Street",
		"recipient:", "", // just check the structure exists
		"items:", "",
	}

	for i := 0; i < len(checks); i += 2 {
		if checks[i+1] == "" {
			// Just check the key exists
			if !strings.Contains(output, checks[i]) {
				t.Errorf("FormatInvoiceTyp output missing %q", checks[i])
			}
		} else {
			if !strings.Contains(output, checks[i+1]) {
				t.Errorf("FormatInvoiceTyp output missing value %q", checks[i+1])
			}
		}
	}

	// Check item rendering
	if !strings.Contains(output, "Consulting") {
		t.Error("FormatInvoiceTyp should include item description")
	}
	if !strings.Contains(output, "270") {
		t.Error("FormatInvoiceTyp should include duration in minutes")
	}
	if !strings.Contains(output, "150.00") {
		t.Error("FormatInvoiceTyp should include hourly rate")
	}
}

func TestFormatInvoiceTypNoTitle(t *testing.T) {
	t.Parallel()
	data := InvoiceData{
		InvoiceNumber:      "INV-2024-03-TEST",
		InvoiceDate:        "2024-04-01",
		DueDate:            "2024-05-01",
		Month:              "2024-03",
		BusinessName:       "Test Business Ltd",
		Street:             "123 Test Street",
		City:               "London",
		PostalCode:         "SW1A 1AA",
		BankName:           "Test Bank",
		AccountName:        "Test Account",
		SortCode:           "12-34-56",
		AccountNumber:      "12345678",
		PaymentTerms:       "Payment due within 30 days.",
		CustomerName:       "John Doe",
		CustomerStreet:     "456 Client Road",
		CustomerCity:       "Manchester",
		CustomerPostalCode: "M1 1AA",
		Items: []ItemData{
			{Description: "Consulting", DurMin: 270, HourlyRate: 150.00},
		},
	}

	output := FormatInvoiceTyp(data)

	// Title fields should not appear when title is empty
	if strings.Contains(output, "title:") {
		t.Error("FormatInvoiceTyp should not include title fields when title is empty")
	}
}

func TestFormatInvoiceTypEscapesSpecialChars(t *testing.T) {
	t.Parallel()
	data := InvoiceData{
		InvoiceNumber:      "INV-2024-03-TEST",
		InvoiceDate:        "2024-04-01",
		DueDate:            "2024-05-01",
		Month:              "2024-03",
		BusinessName:       `Test #1 "Business" @Home`,
		Street:             "123 Main St",
		City:               "London",
		PostalCode:         "SW1A 1AA",
		BankName:           "Test Bank",
		AccountName:        "Test Account",
		SortCode:           "12-34-56",
		AccountNumber:      "12345678",
		PaymentTerms:       "Payment due within 30 days.",
		CustomerName:       `Client #2 "Corp"`,
		CustomerStreet:     "456 Client Road",
		CustomerCity:       "Manchester",
		CustomerPostalCode: "M1 1AA",
		Items: []ItemData{
			{Description: "Consulting", DurMin: 270, HourlyRate: 150.00},
		},
	}

	output := FormatInvoiceTyp(data)

	// Verify special characters are escaped in the output
	if !strings.Contains(output, `\#1`) {
		t.Error("FormatInvoiceTyp should contain escaped # in business name")
	}
	if !strings.Contains(output, `\@Home`) {
		t.Error("FormatInvoiceTyp should contain escaped @ in business name")
	}
	if !strings.Contains(output, `\"Business\"`) {
		t.Error("FormatInvoiceTyp should contain escaped quotes in business name")
	}
	if !strings.Contains(output, `\#2`) {
		t.Error("FormatInvoiceTyp should contain escaped # in customer name")
	}
}

func TestEscape(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no escaping needed", "hello world", "hello world"},
		{"backslash escaped", `hello\world`, `hello\\world`},
		{"double quote escaped", `hello"world`, `hello\"world`},
		{"both escapes", `hello\"world`, `hello\\\"world`},
		{"hash escaped", "invoice #123", `invoice \#123`},
		{"at sign escaped", "@mention", `\@mention`},
		{"newline escaped", "line1\nline2", `line1\nline2`},
		{"all special chars", `#test@"path"\here`, `\#test\@\"path\"\\here`},
		{"typst injection attempt", `#show: link("https://evil.com")`, `\#show: link(\"https://evil.com\")`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escape(tt.input)
			if got != tt.want {
				t.Errorf("escape(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseAddress(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		input      string
		wantStreet string
		wantCity   string
		wantPostal string
	}{
		{"single line", "123 Main St", "123 Main St", "", ""},
		{"two lines", "123 Main St\nLondon", "123 Main St", "London", ""},
		{"three lines", "123 Main St\nLondon\nSW1A 1AA", "123 Main St", "London", "SW1A 1AA"},
		{"four lines", "Flat 1\n123 Main St\nLondon\nSW1A 1AA", "Flat 1, 123 Main St", "London", "SW1A 1AA"},
		{"empty string", "", "", "", ""},
		{"whitespace only", "   \n  \n  ", "   \n  \n  ", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			street, city, postalCode := parseAddress(tt.input)
			if street != tt.wantStreet {
				t.Errorf("street = %q, want %q", street, tt.wantStreet)
			}
			if city != tt.wantCity {
				t.Errorf("city = %q, want %q", city, tt.wantCity)
			}
			if postalCode != tt.wantPostal {
				t.Errorf("postalCode = %q, want %q", postalCode, tt.wantPostal)
			}
		})
	}
}
