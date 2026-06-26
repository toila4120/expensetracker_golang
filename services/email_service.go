package services

import (
	"crypto/tls"
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

	// Port 465 dùng SSL, port 587 dùng STARTTLS
	if s.port == 465 {
		return s.sendWithSSL(addr, auth, to, msg)
	}

	err := smtp.SendMail(addr, auth, s.from, []string{to}, []byte(msg))
	if err != nil {
		log.Println("Email send error:", err)
		return err
	}

	log.Printf("Email sent to %s: %s", to, subject)
	return nil
}

func (s *EmailService) sendWithSSL(addr string, auth smtp.Auth, to, msg string) error {
	tlsConfig := &tls.Config{
		ServerName: s.host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		log.Println("SSL dial error:", err)
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		log.Println("SMTP client error:", err)
		return err
	}
	defer client.Close()

	if err = client.Auth(auth); err != nil {
		log.Println("SMTP auth error:", err)
		return err
	}

	if err = client.Mail(s.from); err != nil {
		log.Println("SMTP mail error:", err)
		return err
	}

	if err = client.Rcpt(to); err != nil {
		log.Println("SMTP rcpt error:", err)
		return err
	}

	w, err := client.Data()
	if err != nil {
		log.Println("SMTP data error:", err)
		return err
	}

	if _, err = w.Write([]byte(msg)); err != nil {
		log.Println("SMTP write error:", err)
		return err
	}

	if err = w.Close(); err != nil {
		log.Println("SMTP close error:", err)
		return err
	}

	log.Printf("Email sent to %s (SSL)", to)
	return client.Quit()
}
