package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// SlackNotifier sends notifications via Slack webhook
type SlackNotifier struct {
	WebhookURL string
}

// NewSlackNotifier creates a new Slack notifier
func NewSlackNotifier() *SlackNotifier {
	return &SlackNotifier{
		WebhookURL: os.Getenv("SLACK_WEBHOOK_URL"),
	}
}

func (s *SlackNotifier) Channel() string {
	return "slack"
}

func (s *SlackNotifier) Send(ctx context.Context, n *Notification) error {
	if s.WebhookURL == "" {
		return fmt.Errorf("SLACK_WEBHOOK_URL not set")
	}

	payload := map[string]interface{}{
		"blocks": []map[string]interface{}{
			{
				"type": "header",
				"text": map[string]string{
					"type":  "plain_text",
					"text":  "🏈 AkiGura 空き枠通知",
					"emoji": "true",
				},
			},
			{
				"type": "section",
				"fields": []map[string]string{
					{"type": "mrkdwn", "text": fmt.Sprintf("*チーム:*\n%s", n.TeamName)},
					{"type": "mrkdwn", "text": fmt.Sprintf("*施設:*\n%s", n.FacilityName)},
					{"type": "mrkdwn", "text": fmt.Sprintf("*日時:*\n%s %s", n.SlotDate, n.SlotTime)},
					{"type": "mrkdwn", "text": fmt.Sprintf("*場所:*\n%s", n.CourtName)},
				},
			},
			{
				"type": "context",
				"elements": []map[string]string{
					{"type": "mrkdwn", "text": "お早めにご予約ください。"},
				},
			},
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", s.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("slack webhook error: %d", resp.StatusCode)
	}
	return nil
}
