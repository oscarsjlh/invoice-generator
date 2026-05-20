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

func SendInvoice(to, toName, from, invoiceNumber string, pdfData []byte, cfg Config) error {
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

	port, err := strconv.Atoi(cfg.SMTPPort)
	if err != nil {
		return fmt.Errorf("invalid SMTP port %s: %w", cfg.SMTPPort, err)
	}

	d := gomail.NewDialer(cfg.SMTPHost, port, cfg.SMTPUser, cfg.SMTPPass)
	d.TLSConfig = &tls.Config{ServerName: cfg.SMTPHost}
	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	return nil
}
