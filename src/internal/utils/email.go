package utils

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"os"
	"strings"
)

// loginAuth implements smtp.Auth using the AUTH LOGIN mechanism,
// required by GoDaddy/Secureserver SMTP which rejects AUTH PLAIN.
type loginAuth struct {
	username, password string
}

func (a *loginAuth) Start(_ *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", nil, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	challenge := strings.ToLower(strings.TrimSpace(string(fromServer)))
	switch {
	case strings.HasPrefix(challenge, "username"):
		return []byte(a.username), nil
	case strings.HasPrefix(challenge, "password"):
		return []byte(a.password), nil
	default:
		return nil, fmt.Errorf("unexpected server challenge: %q", fromServer)
	}
}

func buildMessage(senderName, senderEmail, to, subject, body string) []byte {
	return []byte(fmt.Sprintf(
		"From: %s <%s>\r\nTo: %s\r\nSubject: %s\r\nMIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n%s\r\n",
		senderName, senderEmail, to, subject, body,
	))
}

func smtpConfig() (host, port, username, password, senderEmail, senderName string, err error) {
	host = os.Getenv("SMTP_HOST")
	port = os.Getenv("SMTP_PORT")
	username = os.Getenv("SMTP_USERNAME")
	password = os.Getenv("SMTP_PASSWORD")
	senderEmail = os.Getenv("SMTP_SENDER_EMAIL")
	senderName = os.Getenv("SMTP_SENDER_NAME")
	if host == "" || port == "" || senderEmail == "" {
		err = fmt.Errorf("SMTP settings not fully configured (host=%q port=%q sender=%q)", host, port, senderEmail)
		return
	}
	if senderName == "" {
		senderName = "Zef Platform"
	}
	return
}

// sendSMTP sends email, using implicit TLS for port 465 and STARTTLS for all other ports.
func sendSMTP(to, subject, body string) error {
	host, port, username, password, senderEmail, senderName, err := smtpConfig()
	if err != nil {
		return err
	}

	msg := buildMessage(senderName, senderEmail, to, subject, body)
	addr := host + ":" + port
	auth := &loginAuth{username, password}

	if port == "465" {
		// Implicit TLS (SMTPS)
		tlsCfg := &tls.Config{ServerName: host}
		conn, err := tls.Dial("tcp", addr, tlsCfg)
		if err != nil {
			return fmt.Errorf("TLS dial: %w", err)
		}
		c, err := smtp.NewClient(conn, host)
		if err != nil {
			return fmt.Errorf("SMTP client: %w", err)
		}
		defer c.Close()
		if err = c.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth: %w", err)
		}
		if err = c.Mail(senderEmail); err != nil {
			return fmt.Errorf("SMTP MAIL FROM: %w", err)
		}
		if err = c.Rcpt(to); err != nil {
			return fmt.Errorf("SMTP RCPT TO: %w", err)
		}
		wc, err := c.Data()
		if err != nil {
			return fmt.Errorf("SMTP DATA: %w", err)
		}
		if _, err = wc.Write(msg); err != nil {
			return fmt.Errorf("SMTP write: %w", err)
		}
		return wc.Close()
	}

	// STARTTLS (port 587 / 25)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("SMTP client: %w", err)
	}
	defer c.Close()
	tlsCfg := &tls.Config{ServerName: host}
	if err = c.StartTLS(tlsCfg); err != nil {
		return fmt.Errorf("STARTTLS: %w", err)
	}
	if err = c.Auth(auth); err != nil {
		return fmt.Errorf("SMTP auth: %w", err)
	}
	if err = c.Mail(senderEmail); err != nil {
		return fmt.Errorf("SMTP MAIL FROM: %w", err)
	}
	if err = c.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP RCPT TO: %w", err)
	}
	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA: %w", err)
	}
	if _, err = wc.Write(msg); err != nil {
		return fmt.Errorf("SMTP write: %w", err)
	}
	return wc.Close()
}

// SendEmail sends an email asynchronously.
func SendEmail(to, subject, body string) error {
	if _, _, _, _, se, _, err := smtpConfig(); err != nil || se == "" {
		log.Printf("SMTP not configured, skipping email to %s", to)
		return err
	}
	go func() {
		if err := sendSMTP(to, subject, body); err != nil {
			log.Printf("[SMTP ERROR] Failed to send email to %s: %v", to, err)
		} else {
			log.Printf("[SMTP SUCCESS] Sent email to %s with subject: %q", to, subject)
		}
	}()
	return nil
}

// SendEmailSync sends an email synchronously and returns any SMTP error directly.
func SendEmailSync(to, subject, body string) error {
	if err := sendSMTP(to, subject, body); err != nil {
		log.Printf("[SMTP ERROR] Failed to send email to %s: %v", to, err)
		return err
	}
	log.Printf("[SMTP SUCCESS] Sent email to %s with subject: %q", to, subject)
	return nil
}
