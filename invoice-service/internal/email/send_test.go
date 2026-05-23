package email

import (
	"errors"
	"testing"

	gomail "github.com/go-mail/mail/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockDialer struct {
	lastMsg *gomail.Message
	err     error
}

func (m *mockDialer) DialAndSend(msgs ...*gomail.Message) error {
	if m.err != nil {
		return m.err
	}
	if len(msgs) > 0 {
		m.lastMsg = msgs[0]
	}
	return nil
}

func TestSendInvoiceWithMockDialer(t *testing.T) {
	t.Parallel()
	md := &mockDialer{}

	pdfData := []byte("fake-pdf-content")
	err := sendEmail(md, "customer@example.com", "John Doe", "biz@example.com", "INV-2024-03-ALL", pdfData)
	require.NoError(t, err)
	require.NotNil(t, md.lastMsg)

	msg := md.lastMsg
	fromList := msg.GetHeader("From")
	assert.Contains(t, fromList[0], "biz@example.com")

	toList := msg.GetHeader("To")
	assert.Contains(t, toList[0], "customer@example.com")

	subjectList := msg.GetHeader("Subject")
	assert.Contains(t, subjectList[0], "INV-2024-03-ALL")
}

func TestSendInvoiceDialerError(t *testing.T) {
	t.Parallel()
	md := &mockDialer{err: errors.New("smtp connection refused")}

	err := sendEmail(md, "customer@example.com", "John Doe", "biz@example.com", "INV-2024-03-ALL", []byte("pdf"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "smtp connection refused")
}

func TestSendInvoiceBodyContainsRecipientName(t *testing.T) {
	t.Parallel()
	md := &mockDialer{}

	err := sendEmail(md, "customer@example.com", "Alice", "biz@example.com", "INV-2024-01-DESIGN", []byte("pdf"))
	require.NoError(t, err)

	msg := md.lastMsg
	fromList := msg.GetHeader("Subject")
	assert.Contains(t, fromList[0], "INV-2024-01-DESIGN")
}

func TestNewDialerInvalidPort(t *testing.T) {
	t.Parallel()
	cfg := Config{
		SMTPHost: "smtp.example.com",
		SMTPPort: "not-a-port",
		SMTPFrom: "test@example.com",
	}
	_, err := NewDialer(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid SMTP port")
}
