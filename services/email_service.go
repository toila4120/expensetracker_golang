package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type EmailService struct {
	sendgridKey string
	from        string
	fromName    string
	enabled     bool
}

func NewEmailService() *EmailService {
	apiKey := os.Getenv("SENDGRID_API_KEY")
	from := os.Getenv("SMTP_FROM")
	fromName := os.Getenv("SMTP_FROM_NAME")

	if fromName == "" {
		fromName = "ExpenseTracker"
	}

	enabled := apiKey != ""
	if enabled {
		log.Println("✅ Email Service initialized (SendGrid)")
	} else {
		log.Println("⚠️ Email Service disabled (SENDGRID_API_KEY not set)")
	}

	return &EmailService{
		sendgridKey: apiKey,
		from:        from,
		fromName:    fromName,
		enabled:     enabled,
	}
}

func (s *EmailService) Send(to, subject, body string) error {
	if !s.enabled {
		return nil
	}

	payload := map[string]interface{}{
		"personalizations": []map[string]interface{}{
			{
				"to": []map[string]string{
					{"email": to},
				},
				"subject": subject,
			},
		},
		"from": map[string]string{
			"email": s.from,
			"name":  s.fromName,
		},
		"content": []map[string]string{
			{
				"type":  "text/plain",
				"value": body,
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Println("Email marshal error:", err)
		return err
	}

	req, err := http.NewRequest("POST", "https://api.sendgrid.com/v3/mail/send", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println("Email request error:", err)
		return err
	}

	req.Header.Set("Authorization", "Bearer "+s.sendgridKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Email send error:", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Email send error (status %d): %s", resp.StatusCode, string(body))
		return fmt.Errorf("sendgrid error: %d", resp.StatusCode)
	}

	log.Printf("Email sent to %s: %s", to, subject)
	return nil
}
