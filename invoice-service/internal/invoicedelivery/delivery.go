package invoicedelivery

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"

	"invoice-app/internal/config"
	"invoice-app/internal/db"
	"invoice-app/internal/email"
	"invoice-app/internal/pdf"
	"invoice-app/templates"
)

var (
	ErrCustomerEmailMissing = errors.New("customer email not configured")
	ErrSMTPNotConfigured    = errors.New("smtp not configured")
	ErrRenderPDF            = errors.New("render pdf")
)

type Store interface {
	GetInvoice(id int64) (db.Invoice, error)
	LoadSettings() (db.Settings, error)
}

type PDFRenderer interface {
	RenderInvoice(ctx context.Context, invoice db.Invoice, settings db.Settings) ([]byte, error)
}

type Mailer interface {
	SendInvoice(ctx context.Context, input SendEmailInput) error
}

type SendEmailInput struct {
	To            string
	ToName        string
	From          string
	InvoiceNumber string
	PDFData       []byte
	Config        email.Config
}

type Delivery struct {
	Store    Store
	Renderer PDFRenderer
	Mailer   Mailer
	Config   config.Config
}

type PDFResult struct {
	Data     []byte
	FileName string
}

type SendResult struct {
	Recipient string
}

func New(store Store, cfg config.Config) *Delivery {
	return &Delivery{
		Store:    store,
		Renderer: TypstRenderer{},
		Mailer:   EmailMailer{},
		Config:   cfg,
	}
}

func (d *Delivery) RenderPDF(ctx context.Context, invoiceID int64) (PDFResult, error) {
	invoice, err := d.Store.GetInvoice(invoiceID)
	if err != nil {
		return PDFResult{}, err
	}
	settings, err := d.Store.LoadSettings()
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return PDFResult{}, fmt.Errorf("load settings: %w", err)
	}
	data, err := d.Renderer.RenderInvoice(ctx, invoice, settings)
	if err != nil {
		return PDFResult{}, err
	}
	return PDFResult{Data: data, FileName: invoice.InvoiceNumber + ".pdf"}, nil
}

func (d *Delivery) Send(ctx context.Context, invoiceID int64) (SendResult, error) {
	invoice, err := d.Store.GetInvoice(invoiceID)
	if err != nil {
		return SendResult{}, err
	}
	settings, err := d.Store.LoadSettings()
	if err != nil {
		return SendResult{}, fmt.Errorf("load settings: %w", err)
	}
	if settings.CustomerEmail == "" {
		return SendResult{}, ErrCustomerEmailMissing
	}

	cfg := email.Config{
		SMTPHost: d.Config.SMTPHost,
		SMTPPort: d.Config.SMTPPort,
		SMTPUser: d.Config.SMTPUser,
		SMTPPass: d.Config.SMTPPass,
		SMTPFrom: d.Config.SMTPFrom,
	}
	if cfg.SMTPHost == "" || cfg.SMTPFrom == "" {
		return SendResult{}, ErrSMTPNotConfigured
	}

	pdfData, err := d.Renderer.RenderInvoice(ctx, invoice, settings)
	if err != nil {
		return SendResult{}, fmt.Errorf("%w: %w", ErrRenderPDF, err)
	}

	customerName := settings.CustomerName
	if customerName == "" {
		customerName = "Customer"
	}

	if err := d.Mailer.SendInvoice(ctx, SendEmailInput{
		To:            settings.CustomerEmail,
		ToName:        customerName,
		From:          cfg.SMTPFrom,
		InvoiceNumber: invoice.InvoiceNumber,
		PDFData:       pdfData,
		Config:        cfg,
	}); err != nil {
		return SendResult{}, err
	}
	return SendResult{Recipient: settings.CustomerEmail}, nil
}

type EmailMailer struct{}

func (EmailMailer) SendInvoice(ctx context.Context, input SendEmailInput) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return email.SendInvoice(nil, input.To, input.ToName, input.From, input.InvoiceNumber, input.PDFData, input.Config)
}

type TypstRenderer struct{}

func (TypstRenderer) RenderInvoice(ctx context.Context, invoice db.Invoice, settings db.Settings) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	tmpDir, err := os.MkdirTemp("", "typst-invoice-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	templateBytes, err := templates.FS.ReadFile("invoice-maker.typ")
	if err != nil {
		return nil, fmt.Errorf("read typst template: %w", err)
	}

	typContent := pdf.FormatInvoiceTyp(BuildPDFData(invoice, settings))
	return pdf.GenerateInvoicePDF(tmpDir, templateBytes, typContent)
}

func BuildPDFData(invoice db.Invoice, settings db.Settings) pdf.InvoiceData {
	street, city, postalCode := pdf.ParseAddress(invoice.BusinessAddress)
	parsedCustomerStreet, parsedCustomerCity, parsedCustomerPostalCode := pdf.ParseAddress(settings.CustomerAddress)

	customerStreet := parsedCustomerStreet
	if customerStreet == "" {
		customerStreet = street
	}

	customerCity := settings.CustomerCity
	if customerCity == "" {
		customerCity = parsedCustomerCity
	}
	if customerCity == "" {
		customerCity = city
	}

	customerPostalCode := settings.CustomerPostalCode
	if customerPostalCode == "" {
		customerPostalCode = parsedCustomerPostalCode
	}
	if customerPostalCode == "" {
		customerPostalCode = postalCode
	}

	data := pdf.InvoiceData{
		InvoiceNumber:      invoice.InvoiceNumber,
		InvoiceDate:        invoice.InvoiceDate,
		DueDate:            invoice.DueDate,
		Month:              invoice.Month,
		BusinessName:       invoice.BusinessName,
		Street:             street,
		City:               city,
		PostalCode:         postalCode,
		BankName:           invoice.BankName,
		AccountName:        invoice.AccountName,
		SortCode:           invoice.SortCode,
		AccountNumber:      invoice.AccountNumber,
		PaymentTerms:       invoice.PaymentTerms,
		CustomerName:       settings.CustomerName,
		CustomerTitle:      settings.CustomerTitle,
		CustomerStreet:     customerStreet,
		CustomerCity:       customerCity,
		CustomerPostalCode: customerPostalCode,
	}

	for _, line := range invoice.Lines {
		hoursMinutes := int(math.Round(line.Hours * 60))
		data.Items = append(data.Items, pdf.ItemData{
			Description: line.Category,
			DurMin:      hoursMinutes,
			HourlyRate:  line.Rate,
		})
	}

	return data
}
