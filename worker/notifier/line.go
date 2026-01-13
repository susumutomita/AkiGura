package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// LINENotifier sends notifications via LINE Notify
type LINENotifier struct {
	AccessToken string
}

// NewLINENotifier creates a new LINE notifier
func NewLINENotifier() *LINENotifier {
	return &LINENotifier{
		AccessToken: os.Getenv("LINE_NOTIFY_TOKEN"),
	}
}

func (l *LINENotifier) Channel() string {
	return "line"
}

func (l *LINENotifier) Send(ctx context.Context, n *Notification) error {
	if l.AccessToken == "" {
		return fmt.Errorf("LINE_NOTIFY_TOKEN not set")
	}

	if len(n.Slots) == 0 {
		return nil
	}

	// Build message with all slots
	var lines []string
	lines = append(lines, fmt.Sprintf("🏈 AkiGura 空き枠通知\n"))
	lines = append(lines, fmt.Sprintf("%s様\n", n.TeamName))
	lines = append(lines, fmt.Sprintf("空き枠が%d件見つかりました。\n", len(n.Slots)))

	for i, slot := range n.Slots {
		lines = append(lines, fmt.Sprintf("\n【%d】%s", i+1, slot.FacilityName))
		lines = append(lines, fmt.Sprintf("日時: %s %s", slot.SlotDate, slot.SlotTime))
		lines = append(lines, fmt.Sprintf("場所: %s", slot.CourtName))
	}
	lines = append(lines, "\nお早めにご予約ください。")

	message := strings.Join(lines, "\n")

	data := fmt.Sprintf("message=%s", message)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://notify-api.line.me/api/notify", bytes.NewBufferString(data))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+l.AccessToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("LINE notify error: %d", resp.StatusCode)
	}
	return nil
}

// LINEMessagingNotifier uses LINE Messaging API for individual users
type LINEMessagingNotifier struct {
	ChannelAccessToken string
}

// NewLINEMessagingNotifier creates a LINE Messaging API notifier
func NewLINEMessagingNotifier() *LINEMessagingNotifier {
	return &LINEMessagingNotifier{
		ChannelAccessToken: os.Getenv("LINE_CHANNEL_ACCESS_TOKEN"),
	}
}

func (l *LINEMessagingNotifier) Channel() string {
	return "line_messaging"
}

func (l *LINEMessagingNotifier) Send(ctx context.Context, n *Notification) error {
	if l.ChannelAccessToken == "" {
		return fmt.Errorf("LINE_CHANNEL_ACCESS_TOKEN not set")
	}

	if len(n.Slots) == 0 {
		return nil
	}

	// Note: This requires storing LINE user IDs in the database
	// For now, we'll use a placeholder
	lineUserID := "" // Would come from team's LINE registration
	if lineUserID == "" {
		return fmt.Errorf("LINE user ID not registered for team")
	}

	// Build body contents for all slots
	var bodyContents []map[string]interface{}
	bodyContents = append(bodyContents, map[string]interface{}{
		"type":   "text",
		"text":   fmt.Sprintf("空き枠が%d件見つかりました", len(n.Slots)),
		"weight": "bold",
		"size":   "lg",
	})

	for _, slot := range n.Slots {
		bodyContents = append(bodyContents, map[string]interface{}{
			"type": "separator", "margin": "md",
		})
		bodyContents = append(bodyContents, map[string]interface{}{
			"type": "text", "text": slot.FacilityName, "weight": "bold", "margin": "md",
		})
		bodyContents = append(bodyContents, map[string]interface{}{
			"type": "text", "text": fmt.Sprintf("日時: %s %s", slot.SlotDate, slot.SlotTime),
		})
		bodyContents = append(bodyContents, map[string]interface{}{
			"type": "text", "text": fmt.Sprintf("場所: %s", slot.CourtName),
		})
	}

	message := map[string]interface{}{
		"to": lineUserID,
		"messages": []map[string]interface{}{
			{
				"type":    "flex",
				"altText": fmt.Sprintf("空き枠通知（%d件）", len(n.Slots)),
				"contents": map[string]interface{}{
					"type": "bubble",
					"header": map[string]interface{}{
						"type":            "box",
						"layout":          "vertical",
						"backgroundColor": "#4F46E5",
						"contents": []map[string]interface{}{
							{"type": "text", "text": "🏈 AkiGura", "color": "#ffffff", "weight": "bold"},
						},
					},
					"body": map[string]interface{}{
						"type":     "box",
						"layout":   "vertical",
						"contents": bodyContents,
					},
				},
			},
		},
	}

	body, _ := json.Marshal(message)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.line.me/v2/bot/message/push", bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+l.ChannelAccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("LINE messaging error: %d", resp.StatusCode)
	}
	return nil
}
