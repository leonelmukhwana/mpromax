package notifications

import (
	"fmt"
	"net/smtp"
	"os"
)

type EmailProvider struct {
	Host   string
	Port   string
	Sender string
}

func NewEmailProvider() *EmailProvider {
	return &EmailProvider{
		Host:   "smtp.resend.com",
		Port:   "587",
		Sender: "onboarding@resend.dev",
	}
}

func (p *EmailProvider) Send(to, subject, body string) error {
	// 1. Get credentials from .env
	// We use "resend" as the username and your API Key (re_...) as the password
	user := "resend"
	pass := os.Getenv("SMTP_PASS")

	if pass == "" {
		fmt.Println("❌ ERROR: SMTP_PASS is missing in .env file")
		return fmt.Errorf("SMTP credentials missing")
	}

	fmt.Printf("DEBUG: Connecting to %s:%s to send email to %s...\n", p.Host, p.Port, to)

	// 2. Set up Authentication
	auth := smtp.PlainAuth("", user, pass, p.Host)

	// 3. Format the email message (HTML support included)
	msg := fmt.Sprintf("From: MaidProMax <%s>\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"\r\n"+
		"<html><body>%s</body></html>\r\n", p.Sender, to, subject, body)

	// 4. Send the mail
	addr := fmt.Sprintf("%s:%s", p.Host, p.Port)
	err := smtp.SendMail(addr, auth, p.Sender, []string{to}, []byte(msg))

	if err != nil {
		// This will print the exact error from Resend (e.g., Auth failure or Sandbox limit)
		fmt.Printf("❌ Resend SMTP Error: %v\n", err)
		return err
	}

	fmt.Printf("✅ SUCCESS: Email sent to %s via Resend\n", to)
	return nil
}
