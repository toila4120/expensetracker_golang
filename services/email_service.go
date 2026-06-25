package services

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strconv"
)

type EmailService struct {
	host     string
	port     int
	username string
	password string
	from     string
	enabled  bool
}

func NewEmailService() *EmailService {
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	host := os.Getenv("SMTP_HOST")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")
	from := os.Getenv("SMTP_FROM")

	enabled := host != "" && user != "" && pass != ""
	if enabled {
		log.Println("✅ Email Service initialized")
	} else {
		log.Println("⚠️ Email Service disabled (SMTP not configured)")
	}

	return &EmailService{
		host:     host,
		port:     port,
		username: user,
		password: pass,
		from:     from,
		enabled:  enabled,
	}
}

func (s *EmailService) Send(to, subject, body string) error {
	if !s.enabled {
		return nil
	}

	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		s.from, to, subject, body)

	auth := smtp.PlainAuth("", s.username, s.password, s.host)

	err := smtp.SendMail(addr, auth, s.from, []string{to}, []byte(msg))
	if err != nil {
		log.Println("Email send error:", err)
		return err
	}

	log.Printf("Email sent to %s: %s", to, subject)
	return nil
}
