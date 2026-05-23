package email

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"strconv"

	gomail "github.com/go-mail/mail/v2"
)

type Config struct {
	SMTPHost string
	SMTPPort string
	SMTPUser string
	SMTPPass string
	SMTPFrom string
}

type Dialer interface {
	DialAndSend(m ...*gomail.Message) error
}

// gomailDialer wraps gomail.Dialer to implement the Dialer interface.
type gomailDialer struct {
	d *gomail.Dialer
}

func (g *gomailDialer) DialAndSend(m ...*gomail.Message) error {
	return g.d.DialAndSend(m...)
}

// NewDialer creates a Dialer from SMTP config.
func NewDialer(cfg Config) (Dialer, error) {
	port, err := strconv.Atoi(cfg.SMTPPort)
	if err != nil {
		return nil, fmt.Errorf("invalid SMTP port %s: %w", cfg.SMTPPort, err)
	}
	d := gomail.NewDialer(cfg.SMTPHost, port, cfg.SMTPUser, cfg.SMTPPass)
	d.TLSConfig = &tls.Config{ServerName: cfg.SMTPHost}
	return &gomailDialer{d: d}, nil
}

// sendEmail is the internal implementation that accepts a Dialer.
func sendEmail(d Dialer, to, toName, from, invoiceNumber string, pdfData []byte) error {
	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", fmt.Sprintf("Invoice %s", invoiceNumber))

	body := fmt.Sprintf(`Dear %s,

Please find attached invoice %s.

Thank you for your business.

Best regards`, toName, invoiceNumber)
	m.SetBody("text/plain", body)
	m.AttachReader(fmt.Sprintf("%s.pdf", invoiceNumber), bytes.NewReader(pdfData))

	return d.DialAndSend(m)
}

// SendInvoice sends an invoice email using the given Dialer.
// If dialer is nil, it creates a new dialer from the config.
func SendInvoice(d Dialer, to, toName, from, invoiceNumber string, pdfData []byte, cfg Config) error {
	if d != nil {
		return sendEmail(d, to, toName, from, invoiceNumber, pdfData)
	}
	dialer, err := NewDialer(cfg)
	if err != nil {
		return err
	}
	return sendEmail(dialer, to, toName, from, invoiceNumber, pdfData)
}
