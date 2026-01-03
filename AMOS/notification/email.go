package notification

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"
	"strings"

	"github.com/HarshD0011/AMOS/AMOS/config"
)

// EmailService handles sending emails
type EmailService struct {
	config config.EmailConfig
}

// NewEmailService creates a new email service
func NewEmailService(cfg config.EmailConfig) *EmailService {
	return &EmailService{config: cfg}
}

// SendEmail sends an email
func (es *EmailService) SendEmail(to []string, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", es.config.SMTPHost, es.config.SMTPPort)
	from := es.config.FromAddress

	// Setup headers
	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = strings.Join(to, ",")
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=\"UTF-8\""

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	auth := smtp.PlainAuth("", es.config.Username, es.config.Password, es.config.SMTPHost)

	// Handle TLS if enabled
	if es.config.UseTLS {
		tlsconfig := &tls.Config{
			InsecureSkipVerify: true, // For dev/testing, maybe flag for prod
			ServerName:         es.config.SMTPHost,
		}

		conn, err := tls.Dial("tcp", addr, tlsconfig)
		if err != nil {
			return fmt.Errorf("TLS dial failed: %w", err)
		}

		c, err := smtp.NewClient(conn, es.config.SMTPHost)
		if err != nil {
			return err
		}

		if es.config.Username != "" {
			if err = c.Auth(auth); err != nil {
				return err
			}
		}

		if err = c.Mail(from); err != nil {
			return err
		}

		for _, recipient := range to {
			if err = c.Rcpt(recipient); err != nil {
				return err
			}
		}

		w, err := c.Data()
		if err != nil {
			return err
		}

		_, err = w.Write([]byte(message))
		if err != nil {
			return err
		}

		err = w.Close()
		if err != nil {
			return err
		}

		c.Quit()
		return nil
	}

	// Non-TLS / StartTLS standard
	return smtp.SendMail(addr, auth, from, to, []byte(message))
}
