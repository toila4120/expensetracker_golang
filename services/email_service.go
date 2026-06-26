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

type EmailData struct {
	Title      string
	Heading    string
	Message    string
	Highlight  string
	ButtonText string
	ButtonURL  string
	FooterText string
}

func (s *EmailService) Send(to, subject, body string) error {
	return s.SendHTML(to, subject, s.buildHTML(body, ""))
}

func (s *EmailService) SendHTML(to, subject, htmlBody string) error {
	if !s.enabled {
		return nil
	}

	payload := map[string]interface{}{
		"personalizations": []map[string]interface{}{
			{
				"to":      []map[string]string{{"email": to}},
				"subject": subject,
			},
		},
		"from": map[string]string{
			"email": s.from,
			"name":  s.fromName,
		},
		"content": []map[string]string{
			{"type": "text/html", "value": htmlBody},
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

	resp, err := (&http.Client{}).Do(req)
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

func (s *EmailService) GetEmailTemplate(notifType, title, message string) (subject, htmlBody string) {
	switch notifType {
	case "budget_warning":
		subject = fmt.Sprintf("⚠️ %s", title)
		htmlBody = s.buildTemplateHTML(EmailData{
			Heading:    "Cảnh báo ngân sách",
			Message:    message,
			Highlight:  "Bạn đã chi tiêu vượt 80% ngân sách được phân bổ.",
			ButtonText: "Xem ngân sách",
			ButtonURL:  "https://expensetracker.app/budget",
			FooterText: "Hãy theo dõi chi tiêu để không vượt ngân sách nhé!",
		})
	case "budget_exceeded":
		subject = fmt.Sprintf("🚨 %s", title)
		htmlBody = s.buildTemplateHTML(EmailData{
			Heading:    "Vượt ngân sách!",
			Message:    message,
			Highlight:  "Chi tiêu của bạn đã vượt quá ngân sách. Hãy xem lại và điều chỉnh.",
			ButtonText: "Xem chi tiết",
			ButtonURL:  "https://expensetracker.app/budget",
			FooterText: "Cần hỗ trợ? Liên hệ tại support@expensetracker.app",
		})
	case "goal_completed":
		subject = fmt.Sprintf("🎉 %s", title)
		htmlBody = s.buildTemplateHTML(EmailData{
			Heading:    "Chúc mừng!",
			Message:    message,
			Highlight:  "Bạn đã hoàn thành mục tiêu tiết kiệm!",
			ButtonText: "Xem mục tiêu",
			ButtonURL:  "https://expensetracker.app/goals",
		})
	case "goal_deadline":
		subject = fmt.Sprintf("⏰ %s", title)
		htmlBody = s.buildTemplateHTML(EmailData{
			Heading:    "Mục tiêu sắp hết hạn",
			Message:    message,
			Highlight:  "Hãy hoàn thành mục tiêu trước khi hết hạn.",
			ButtonText: "Xem mục tiêu",
			ButtonURL:  "https://expensetracker.app/goals",
		})
	default:
		subject = title
		htmlBody = s.buildHTML(message, "")
	}
	return
}

func (s *EmailService) buildHTML(body, highlight string) string {
	highlightHTML := ""
	if highlight != "" {
		highlightHTML = fmt.Sprintf(`<div style="background:#f0f9ff;border-left:4px solid #3b82f6;padding:16px;margin:24px 0;border-radius:4px;"><p style="margin:0;color:#1e40af;font-size:16px;">%s</p></div>`, highlight)
	}

	return fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="UTF-8"></head>
<body style="margin:0;padding:0;background:#f3f4f6;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;">
<table width="100%%" cellpadding="0" cellspacing="0" style="background:#f3f4f6;padding:40px 20px;"><tr><td align="center">
<table width="600" cellpadding="0" cellspacing="0" style="background:#fff;border-radius:12px;overflow:hidden;box-shadow:0 4px 6px rgba(0,0,0,0.1);">
<tr><td style="background:linear-gradient(135deg,#667eea 0%%,#764ba2 100%%);padding:32px;text-align:center;">
<h1 style="margin:0;color:#fff;font-size:24px;font-weight:700;">ExpenseTracker</h1></td></tr>
<tr><td style="padding:40px 32px;">
<p style="margin:0 0 16px;color:#374151;font-size:16px;line-height:1.6;">%s</p>%s</td></tr>
<tr><td style="background:#f9fafb;padding:24px 32px;text-align:center;border-top:1px solid #e5e7eb;">
<p style="margin:0;color:#9ca3af;font-size:13px;">© 2026 ExpenseTracker</p></td></tr>
</table></td></tr></table></body></html>`, body, highlightHTML)
}

func (s *EmailService) buildTemplateHTML(data EmailData) string {
	heading := data.Heading
	if heading == "" {
		heading = data.Title
	}
	footer := data.FooterText
	if footer == "" {
		footer = "© 2026 ExpenseTracker"
	}

	buttonHTML := ""
	if data.ButtonURL != "" && data.ButtonText != "" {
		buttonHTML = fmt.Sprintf(`<div style="text-align:center;margin:32px 0;"><a href="%s" style="display:inline-block;background:linear-gradient(135deg,#667eea 0%%,#764ba2 100%%);color:#fff;text-decoration:none;padding:14px 32px;border-radius:8px;font-weight:600;font-size:16px;">%s</a></div>`, data.ButtonURL, data.ButtonText)
	}

	highlightHTML := ""
	if data.Highlight != "" {
		highlightHTML = fmt.Sprintf(`<div style="background:#f0f9ff;border-left:4px solid #3b82f6;padding:16px;margin:24px 0;border-radius:4px;"><p style="margin:0;color:#1e40af;font-size:15px;">%s</p></div>`, data.Highlight)
	}

	return fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="UTF-8"></head>
<body style="margin:0;padding:0;background:#f3f4f6;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;">
<table width="100%%" cellpadding="0" cellspacing="0" style="background:#f3f4f6;padding:40px 20px;"><tr><td align="center">
<table width="600" cellpadding="0" cellspacing="0" style="background:#fff;border-radius:12px;overflow:hidden;box-shadow:0 4px 6px rgba(0,0,0,0.1);">
<tr><td style="background:linear-gradient(135deg,#667eea 0%%,#764ba2 100%%);padding:32px;text-align:center;">
<h1 style="margin:0;color:#fff;font-size:24px;font-weight:700;">ExpenseTracker</h1></td></tr>
<tr><td style="padding:40px 32px;">
<h2 style="margin:0 0 16px;color:#1f2937;font-size:22px;font-weight:600;">%s</h2>
<p style="margin:0 0 16px;color:#374151;font-size:16px;line-height:1.6;">%s</p>%s%s</td></tr>
<tr><td style="background:#f9fafb;padding:24px 32px;text-align:center;border-top:1px solid #e5e7eb;">
<p style="margin:0;color:#9ca3af;font-size:13px;">%s</p></td></tr>
</table></td></tr></table></body></html>`, heading, data.Message, highlightHTML, buttonHTML, footer)
}
