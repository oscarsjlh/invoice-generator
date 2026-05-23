package pdf

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func GenerateInvoicePDF(workDir string, templateBytes []byte, typContent string) ([]byte, error) {
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("create work dir: %w", err)
	}
	typFile := filepath.Join(workDir, "invoice.typ")

	if err := os.WriteFile(filepath.Join(workDir, "invoice-maker.typ"), templateBytes, 0644); err != nil {
		return nil, fmt.Errorf("write typst template: %w", err)
	}

	if err := os.WriteFile(typFile, []byte(typContent), 0644); err != nil {
		return nil, fmt.Errorf("write typst file: %w", err)
	}

	// Try typst first, then typst-cli, then check common paths
	typstBin := findTypst()
	if typstBin == "" {
		return nil, fmt.Errorf("typst not found — install with: curl -L https://github.com/typst/typst/releases/latest/download/typst-x86_64-unknown-linux-musl.tar.xz")
	}

	cmd := exec.Command(typstBin, "compile", "--root", workDir, typFile, filepath.Join(workDir, "invoice.pdf"))
	cmd.Dir = workDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("typst compile: %w\n%s", err, string(output))
	}

	pdfBytes, err := os.ReadFile(filepath.Join(workDir, "invoice.pdf"))
	if err != nil {
		return nil, fmt.Errorf("read generated pdf: %w", err)
	}

	return pdfBytes, nil
}

func findTypst() string {
	if override := os.Getenv("TYPST_BIN"); override != "" {
		return override
	}
	for _, name := range []string{"typst", "typst-cli"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	home, _ := os.UserHomeDir()
	for _, p := range []string{
		"/usr/local/bin/typst",
		home + "/.cargo/bin/typst",
		"/app/typst",
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func FormatInvoiceTyp(invoice InvoiceData) string {
	items := ""
	for _, item := range invoice.Items {
		hours := float64(item.DurMin) / 60
		items += fmt.Sprintf(`    (
	      description: "%s",
	      dur-min: %d,
	      hourly-rate: %.2f,
	    ),
`, escape(fmt.Sprintf("%s (%.2fh @ £%.2f/hr)", item.Description, hours, item.HourlyRate)), item.DurMin, item.HourlyRate)
	}

	return fmt.Sprintf(`#import "invoice-maker.typ": *

#show: invoice.with(
  language: "en",
  currency: "£",
  invoice-id: "%s",
  issuing-date: "%s",
  delivery-date: "%s",
  due-date: "%s",
  vat: 0,
  biller: (
    name: "%s",
%s
    bank: "%s",
    account-name: "%s",
    sort-code: "%s",
    account-number: "%s",
    payment-terms: "%s",
    address: (
      country: "United Kingdom",
      city: "%s",
      postal-code: "%s",
      street: "%s",
    ),
  ),
  recipient: (
    name: "%s",
%s
    address: (
      country: "United Kingdom",
      city: "%s",
      postal-code: "%s",
      street: "%s",
    ),
  ),
  items: (
%s  ),
  styling: (font: none),
)
`,
		invoice.InvoiceNumber,
		invoice.InvoiceDate,
		invoice.Month+"-01",
		invoice.DueDate,
		escape(invoice.BusinessName),
		billerTitle(invoice),
		escape(invoice.BankName),
		escape(invoice.AccountName),
		escape(invoice.SortCode),
		escape(invoice.AccountNumber),
		escape(invoice.PaymentTerms),
		escape(invoice.City),
		escape(invoice.PostalCode),
		escape(invoice.Street),
		escape(invoice.CustomerName),
		customerTitle(invoice),
		escape(invoice.CustomerCity),
		escape(invoice.CustomerPostalCode),
		escape(invoice.CustomerStreet),
		items,
	)
}

func escape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, `#`, `\#`)
	s = strings.ReplaceAll(s, `@`, `\@`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}

func billerTitle(inv InvoiceData) string {
	if inv.Title != "" {
		return fmt.Sprintf(`    title: "%s",
`, escape(inv.Title))
	}
	return ""
}

func customerTitle(inv InvoiceData) string {
	if inv.CustomerTitle != "" {
		return fmt.Sprintf(`    title: "%s",
`, escape(inv.CustomerTitle))
	}
	return ""
}

type InvoiceData struct {
	InvoiceNumber      string
	InvoiceDate        string
	DueDate            string
	Month              string
	BusinessName       string
	Title              string
	Street             string
	City               string
	PostalCode         string
	BankName           string
	AccountName        string
	SortCode           string
	AccountNumber      string
	PaymentTerms       string
	CustomerName       string
	CustomerTitle      string
	CustomerStreet     string
	CustomerCity       string
	CustomerPostalCode string
	Items              []ItemData
}

// ParseAddress splits an address string into street, city, and postal code.
// Convention: up to 3 non-empty lines — last line = postal code, second-to-last = city, rest = street.
func ParseAddress(address string) (street, city, postalCode string) {
	parts := strings.Split(address, "\n")
	nonEmpty := []string{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	switch len(nonEmpty) {
	case 0:
		return address, "", ""
	case 1:
		return nonEmpty[0], "", ""
	case 2:
		return nonEmpty[0], nonEmpty[1], ""
	default:
		last := nonEmpty[len(nonEmpty)-1]
		city := nonEmpty[len(nonEmpty)-2]
		street = strings.Join(nonEmpty[:len(nonEmpty)-2], ", ")
		return street, city, last
	}
}

func parseAddress(address string) (street, city, postalCode string) {
	return ParseAddress(address)
}

type ItemData struct {
	Description string
	DurMin      int
	HourlyRate  float64
}
