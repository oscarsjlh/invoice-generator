package handler

import (
	"net/http/httptest"
	"testing"
)

func TestParsePositiveFloat(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{"valid positive float", "123.45", 123.45, false},
		{"valid integer as float", "42", 42.0, false},
		{"zero is rejected", "0", 0, true},
		{"negative rejected", "-5.0", 0, true},
		{"non-numeric rejected", "abc", 0, true},
		{"whitespace trimmed", " 10.5 ", 10.5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePositiveFloat(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePositiveFloat(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parsePositiveFloat(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParsePositiveInt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{"valid positive int", "7", 7, false},
		{"zero rejected", "0", 0, true},
		{"negative rejected", "-3", 0, true},
		{"non-numeric rejected", "xyz", 0, true},
		{"whitespace trimmed", " 10 ", 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePositiveInt(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePositiveInt(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parsePositiveInt(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateDate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"valid date", "2024-03-15", "2024-03-15", false},
		{"invalid format rejected", "15/03/2024", "", true},
		{"whitespace trimmed", " 2024-03-15 ", "2024-03-15", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateDate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDate(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("validateDate(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateMonth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"valid month", "2024-03", "2024-03", false},
		{"full date rejected", "2024-03-15", "", true},
		{"whitespace trimmed", " 2024-06 ", "2024-06", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateMonth(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMonth(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("validateMonth(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeCategory(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"non-empty trimmed", " Consulting ", "Consulting"},
		{"empty returns All", "", "All"},
		{"single word", "Design", "Design"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeCategory(tt.input)
			if got != tt.want {
				t.Errorf("normalizeCategory(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatWithCommas(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input int64
		want  string
	}{
		{"small number", 42, "42"},
		{"thousands", 1000, "1,000"},
		{"millions", 1234567, "1,234,567"},
		{"negative", -9999, "-9,999"},
		{"zero", 0, "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatWithCommas(tt.input)
			if got != tt.want {
				t.Errorf("formatWithCommas(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMoney(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input float64
		want  string
	}{
		{"whole number", 100.0, "100.00"},
		{"decimal", 45.67, "45.67"},
		{"zero", 0.0, "0.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := money(tt.input)
			if got != tt.want {
				t.Errorf("money(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNumfmt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input float64
		want  string
	}{
		{"simple decimal", 45.50, "45.50"},
		{"large with commas", 1234.50, "1,234.50"},
		{"negative decimals", -45.50, "-45.50"},
		{"zero", 0.0, "0.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := numfmt(tt.input)
			if got != tt.want {
				t.Errorf("numfmt(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDateLabel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"YYYY-MM-DD", "2024-03-15", "15 Mar 2024"},
		{"RFC3339", "2024-03-15T10:30:00Z", "15 Mar 2024"},
		{"unparseable passes through", "not-a-date", "not-a-date"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dateLabel(tt.input)
			if got != tt.want {
				t.Errorf("dateLabel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSelected(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		current   string
		candidate string
		want      string
	}{
		{"matching", "2024-03", "2024-03", "selected"},
		{"non-matching", "2024-03", "2024-04", ""},
		{"empty current", "", "2024-03", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selected(tt.current, tt.candidate)
			if got != tt.want {
				t.Errorf("selected(%q, %q) = %q, want %q", tt.current, tt.candidate, got, tt.want)
			}
		})
	}
}

func TestMonthName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"01 is January", "01", "January"},
		{"06 is June", "06", "June"},
		{"12 is December", "12", "December"},
		{"invalid passes through", "13", "13"},
		{"empty passes through", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := monthName(tt.input)
			if got != tt.want {
				t.Errorf("monthName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsHTMX(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		header   string
		wantHTMX bool
	}{
		{"HX-Request true", "true", true},
		{"missing header", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.header != "" {
				req.Header.Set("HX-Request", tt.header)
			}
			got := isHTMX(req)
			if got != tt.wantHTMX {
				t.Errorf("isHTMX() = %v, want %v", got, tt.wantHTMX)
			}
		})
	}
}
