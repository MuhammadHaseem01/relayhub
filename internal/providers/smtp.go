package providers

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

type SMTPProvider struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func NewSMTPProvider(host, port, username, password, from string) *SMTPProvider {
	return &SMTPProvider{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

func (p *SMTPProvider) Name() string {
	return "smtp"
}

func (p *SMTPProvider) Send(recipient string, message string) error {
	if p.host == "" {
		return fmt.Errorf("smtp: SMTP_HOST is not configured")
	}
	if p.from == "" {
		return fmt.Errorf("smtp: SMTP_FROM is not configured")
	}
	if recipient == "" {
		return fmt.Errorf("smtp: recipient email address is required")
	}
	if !strings.Contains(recipient, "@") {
		return fmt.Errorf("smtp: recipient %q does not look like a valid email address", recipient)
	}

	addr := net.JoinHostPort(p.host, p.port)
	body := buildMessage(p.from, recipient, "Notification from RelayHub", message)

	var sendErr error
	if p.port == "465" {
		sendErr = p.sendTLS(addr, body, recipient)
	} else {
		sendErr = p.sendSTARTTLS(addr, body, recipient)
	}

	return sendErr
}

func (p *SMTPProvider) sendSTARTTLS(addr string, body []byte, recipient string) error {
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("smtp: connection refused or timeout connecting to %s: %w", addr, err)
	}

	c, err := smtp.NewClient(conn, p.host)
	if err != nil {
		return fmt.Errorf("smtp: failed to create SMTP client: %w", err)
	}
	defer c.Close()

	if ok, _ := c.Extension("STARTTLS"); ok {
		tlsCfg := &tls.Config{ServerName: p.host}
		if err = c.StartTLS(tlsCfg); err != nil {
			return fmt.Errorf("smtp: STARTTLS handshake failed: %w", err)
		}
	}

	if err = p.authenticate(c); err != nil {
		return err
	}

	return p.transmit(c, recipient, body)
}

func (p *SMTPProvider) sendTLS(addr string, body []byte, recipient string) error {
	tlsCfg := &tls.Config{ServerName: p.host}
	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 10 * time.Second},
		"tcp", addr, tlsCfg,
	)
	if err != nil {
		return fmt.Errorf("smtp: TLS connection failed to %s: %w", addr, err)
	}

	c, err := smtp.NewClient(conn, p.host)
	if err != nil {
		return fmt.Errorf("smtp: failed to create SMTP client: %w", err)
	}
	defer c.Close()

	if err = p.authenticate(c); err != nil {
		return err
	}

	return p.transmit(c, recipient, body)
}

func (p *SMTPProvider) authenticate(c *smtp.Client) error {
	if p.username == "" && p.password == "" {
		return nil
	}
	auth := smtp.PlainAuth("", p.username, p.password, p.host)
	if err := c.Auth(auth); err != nil {
		return fmt.Errorf("smtp: authentication failed (check SMTP_USERNAME / SMTP_PASSWORD): %w", err)
	}
	return nil
}

func (p *SMTPProvider) transmit(c *smtp.Client, recipient string, body []byte) error {
	if err := c.Mail(p.from); err != nil {
		return fmt.Errorf("smtp: MAIL FROM rejected for %q: %w", p.from, err)
	}
	if err := c.Rcpt(recipient); err != nil {
		return fmt.Errorf("smtp: RCPT TO rejected for %q — invalid or non-existent address: %w", recipient, err)
	}
	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp: failed to open DATA stream: %w", err)
	}
	if _, err = wc.Write(body); err != nil {
		return fmt.Errorf("smtp: failed to write message body: %w", err)
	}
	if err = wc.Close(); err != nil {
		return fmt.Errorf("smtp: server rejected the message: %w", err)
	}
	return c.Quit()
}

func buildMessage(from, to, subject, body string) []byte {
	var sb strings.Builder
	sb.WriteString("From: " + from + "\r\n")
	sb.WriteString("To: " + to + "\r\n")
	sb.WriteString("Subject: " + subject + "\r\n")
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(body)
	return []byte(sb.String())
}
