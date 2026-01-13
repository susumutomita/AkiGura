package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/smtp"
	"os"
	"strings"
)

// EmailNotifier sends notifications via email
type EmailNotifier struct {
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	FromAddress  string
	FromName     string
}

// NewEmailNotifier creates a new email notifier from environment variables
func NewEmailNotifier() *EmailNotifier {
	return &EmailNotifier{
		SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:     getEnv("SMTP_PORT", "587"),
		SMTPUser:     os.Getenv("SMTP_USER"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		FromAddress:  getEnv("SMTP_FROM", "noreply@akigura.jp"),
		FromName:     getEnv("SMTP_FROM_NAME", "AkiGura"),
	}
}

func (e *EmailNotifier) Channel() string {
	return "email"
}

func (e *EmailNotifier) Send(ctx context.Context, n *Notification) error {
	if len(n.Slots) == 0 {
		return nil
	}

	if e.SMTPUser == "" || e.SMTPPassword == "" {
		// Fall back to logging if SMTP not configured
		fmt.Printf("[EMAIL] To: %s, Subject: 空き枠通知 (%d件)\n", n.TeamEmail, len(n.Slots))
		for _, slot := range n.Slots {
			fmt.Printf("  - %s: %s %s (%s)\n", slot.FacilityName, slot.SlotDate, slot.SlotTime, slot.CourtName)
		}
		return nil
	}

	subject := fmt.Sprintf("【AkiGura】空き枠が見つかりました（%d件）", len(n.Slots))
	body, err := e.renderTemplate(n)
	if err != nil {
		return fmt.Errorf("render template: %w", err)
	}

	msg := fmt.Sprintf("From: %s <%s>\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"\r\n%s",
		e.FromName, e.FromAddress, n.TeamEmail, subject, body)

	auth := smtp.PlainAuth("", e.SMTPUser, e.SMTPPassword, e.SMTPHost)
	addr := fmt.Sprintf("%s:%s", e.SMTPHost, e.SMTPPort)

	return smtp.SendMail(addr, auth, e.FromAddress, []string{n.TeamEmail}, []byte(msg))
}

func (e *EmailNotifier) renderTemplate(n *Notification) (string, error) {
	tmpl := `<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
  <div style="background: #4F46E5; color: white; padding: 20px; border-radius: 8px 8px 0 0;">
    <h1 style="margin: 0;">🏈 AkiGura</h1>
  </div>
  <div style="border: 1px solid #e5e7eb; border-top: none; padding: 20px; border-radius: 0 0 8px 8px;">
    <p>{{.TeamName}} 様</p>
    <p>ご登録いただいた条件にマッチする空き枠が <strong>{{len .Slots}}件</strong> 見つかりました。</p>
    {{range .Slots}}
    <div style="background: #f3f4f6; padding: 15px; border-radius: 8px; margin: 15px 0;">
      <p style="margin: 5px 0;"><strong>施設:</strong> {{.FacilityName}}</p>
      <p style="margin: 5px 0;"><strong>日時:</strong> {{.SlotDate}} {{.SlotTime}}</p>
      <p style="margin: 5px 0;"><strong>場所:</strong> {{.CourtName}}</p>
      {{if .ReservationURL}}
      <p style="margin: 10px 0 5px 0;">
        <a href="{{.ReservationURL}}" style="display: inline-block; background: #4F46E5; color: white; padding: 8px 16px; border-radius: 4px; text-decoration: none; font-size: 14px;">予約サイトを開く →</a>
      </p>
      {{end}}
    </div>
    {{end}}
    <p>お早めにご予約ください。</p>
    <hr style="border: none; border-top: 1px solid #e5e7eb; margin: 20px 0;">
    <p style="color: #6b7280; font-size: 12px;">このメールは AkiGura から自動送信されています。</p>
  </div>
</body>
</html>`

	t, err := template.New("email").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, n); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// SendGridNotifier uses SendGrid API for email
type SendGridNotifier struct {
	APIKey      string
	FromAddress string
	FromName    string
}

// NewSendGridNotifier creates a SendGrid notifier
func NewSendGridNotifier() *SendGridNotifier {
	return &SendGridNotifier{
		APIKey:      os.Getenv("SENDGRID_API_KEY"),
		FromAddress: getEnv("SENDGRID_FROM", "noreply@akigura.jp"),
		FromName:    getEnv("SENDGRID_FROM_NAME", "AkiGura"),
	}
}

func (s *SendGridNotifier) Channel() string {
	return "email"
}

func (s *SendGridNotifier) Send(ctx context.Context, n *Notification) error {
	if s.APIKey == "" {
		return fmt.Errorf("SENDGRID_API_KEY not set")
	}

	if len(n.Slots) == 0 {
		return nil
	}

	// Build plain text body with all slots
	var bodyLines []string
	bodyLines = append(bodyLines, fmt.Sprintf("%s様\n", n.TeamName))
	bodyLines = append(bodyLines, fmt.Sprintf("空き枠が%d件見つかりました。\n", len(n.Slots)))
	for i, slot := range n.Slots {
		bodyLines = append(bodyLines, fmt.Sprintf("\n【%d】%s", i+1, slot.FacilityName))
		bodyLines = append(bodyLines, fmt.Sprintf("日時: %s %s", slot.SlotDate, slot.SlotTime))
		bodyLines = append(bodyLines, fmt.Sprintf("場所: %s", slot.CourtName))
		if slot.ReservationURL != "" {
			bodyLines = append(bodyLines, fmt.Sprintf("予約: %s", slot.ReservationURL))
		}
	}
	bodyLines = append(bodyLines, "\nお早めにご予約ください。")

	payload := map[string]interface{}{
		"personalizations": []map[string]interface{}{
			{
				"to": []map[string]string{
					{"email": n.TeamEmail, "name": n.TeamName},
				},
				"subject": fmt.Sprintf("【AkiGura】空き枠が見つかりました（%d件）", len(n.Slots)),
			},
		},
		"from": map[string]string{
			"email": s.FromAddress,
			"name":  s.FromName,
		},
		"content": []map[string]string{
			{
				"type":  "text/plain",
				"value": strings.Join(bodyLines, "\n"),
			},
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.sendgrid.com/v3/mail/send", bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+s.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("sendgrid error: %d", resp.StatusCode)
	}
	return nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
