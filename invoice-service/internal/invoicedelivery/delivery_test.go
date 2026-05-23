package invoicedelivery

import (
	"context"
	"errors"
	"testing"

	"invoice-app/internal/config"
	"invoice-app/internal/db"
)

type fakeStore struct {
	invoice  db.Invoice
	settings db.Settings
	err      error
}

func (f fakeStore) GetInvoice(id int64) (db.Invoice, error) {
	if f.err != nil {
		return db.Invoice{}, f.err
	}
	return f.invoice, nil
}

func (f fakeStore) LoadSettings() (db.Settings, error) {
	if f.err != nil {
		return db.Settings{}, f.err
	}
	return f.settings, nil
}

type fakeRenderer struct {
	data []byte
	err  error
}

func (f fakeRenderer) RenderInvoice(ctx context.Context, invoice db.Invoice, settings db.Settings) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.data, nil
}

type fakeMailer struct {
	input SendEmailInput
	err   error
}

func (f *fakeMailer) SendInvoice(ctx context.Context, input SendEmailInput) error {
	f.input = input
	return f.err
}

func TestRenderPDF(t *testing.T) {
	t.Parallel()

	d := Delivery{
		Store: fakeStore{invoice: db.Invoice{InvoiceNumber: "INV-1"}},
		Renderer: fakeRenderer{
			data: []byte("pdf"),
		},
	}

	got, err := d.RenderPDF(context.Background(), 1)
	if err != nil {
		t.Fatalf("RenderPDF returned error: %v", err)
	}
	if string(got.Data) != "pdf" {
		t.Fatalf("Data = %q, want pdf", got.Data)
	}
	if got.FileName != "INV-1.pdf" {
		t.Fatalf("FileName = %q, want INV-1.pdf", got.FileName)
	}
}

func TestSendRequiresCustomerEmail(t *testing.T) {
	t.Parallel()

	d := Delivery{
		Store:    fakeStore{invoice: db.Invoice{InvoiceNumber: "INV-1"}},
		Renderer: fakeRenderer{data: []byte("pdf")},
		Config:   config.Config{SMTPHost: "smtp.example.com", SMTPFrom: "billing@example.com"},
	}

	_, err := d.Send(context.Background(), 1)
	if !errors.Is(err, ErrCustomerEmailMissing) {
		t.Fatalf("err = %v, want ErrCustomerEmailMissing", err)
	}
}

func TestSendRequiresSMTPConfig(t *testing.T) {
	t.Parallel()

	d := Delivery{
		Store: fakeStore{
			invoice:  db.Invoice{InvoiceNumber: "INV-1"},
			settings: db.Settings{CustomerEmail: "customer@example.com"},
		},
		Renderer: fakeRenderer{data: []byte("pdf")},
	}

	_, err := d.Send(context.Background(), 1)
	if !errors.Is(err, ErrSMTPNotConfigured) {
		t.Fatalf("err = %v, want ErrSMTPNotConfigured", err)
	}
}

func TestSendUsesRendererAndMailer(t *testing.T) {
	t.Parallel()

	mailer := &fakeMailer{}
	d := Delivery{
		Store: fakeStore{
			invoice:  db.Invoice{InvoiceNumber: "INV-1"},
			settings: db.Settings{CustomerEmail: "customer@example.com"},
		},
		Renderer: fakeRenderer{data: []byte("pdf")},
		Mailer:   mailer,
		Config:   config.Config{SMTPHost: "smtp.example.com", SMTPPort: "587", SMTPFrom: "billing@example.com"},
	}

	got, err := d.Send(context.Background(), 1)
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if got.Recipient != "customer@example.com" {
		t.Fatalf("Recipient = %q, want customer@example.com", got.Recipient)
	}
	if mailer.input.ToName != "Customer" {
		t.Fatalf("ToName = %q, want Customer", mailer.input.ToName)
	}
	if string(mailer.input.PDFData) != "pdf" {
		t.Fatalf("PDFData = %q, want pdf", mailer.input.PDFData)
	}
}
