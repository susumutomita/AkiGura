package notifier

import (
	"context"
	"database/sql"
	"log/slog"
	"time"
)

// Sender processes pending notifications and sends them
type Sender struct {
	DB      *sql.DB
	Manager *Manager
}

// NewSender creates a new notification sender
func NewSender(db *sql.DB) *Sender {
	mgr := NewManager()
	mgr.Register(NewEmailNotifier())
	// Optionally register LINE and Slack if configured
	if ln := NewLINENotifier(); ln.AccessToken != "" {
		mgr.Register(ln)
	}
	if sl := NewSlackNotifier(); sl.WebhookURL != "" {
		mgr.Register(sl)
	}

	return &Sender{
		DB:      db,
		Manager: mgr,
	}
}

// ProcessPending sends all pending notifications
func (s *Sender) ProcessPending(ctx context.Context) (sent, failed int, err error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT n.id, n.team_id, n.channel, n.slot_id,
		       t.name as team_name, t.email as team_email,
		       sl.slot_date, sl.time_from, sl.time_to, sl.court_name,
		       f.name as facility_name
		FROM notifications n
		JOIN teams t ON n.team_id = t.id
		JOIN slots sl ON n.slot_id = sl.id
		JOIN facilities f ON sl.facility_id = f.id
		WHERE n.status = 'pending'
		ORDER BY n.created_at ASC
		LIMIT 100
	`)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var n Notification
		var slotID string
		if err := rows.Scan(&n.ID, &n.TeamID, &n.Channel, &slotID,
			&n.TeamName, &n.TeamEmail,
			&n.SlotDate, &n.SlotTime, &n.CourtName, &n.CourtName,
			&n.FacilityName); err != nil {
			slog.Warn("scan notification", "error", err)
			continue
		}
		notifications = append(notifications, n)
	}

	for _, n := range notifications {
		err := s.Manager.Send(ctx, &n)
		if err != nil {
			slog.Warn("send notification failed", "id", n.ID, "error", err)
			s.updateStatus(ctx, n.ID, "failed")
			failed++
		} else {
			slog.Info("notification sent", "id", n.ID, "channel", n.Channel, "team", n.TeamName)
			s.updateStatus(ctx, n.ID, "sent")
			sent++
		}
	}

	return sent, failed, nil
}

func (s *Sender) updateStatus(ctx context.Context, id, status string) {
	_, err := s.DB.ExecContext(ctx, `
		UPDATE notifications 
		SET status = ?, sent_at = CASE WHEN ? = 'sent' THEN CURRENT_TIMESTAMP ELSE sent_at END
		WHERE id = ?
	`, status, status, id)
	if err != nil {
		slog.Warn("update notification status", "error", err)
	}
}

// StartSender starts a periodic notification sender
func (s *Sender) StartSender(ctx context.Context, interval time.Duration) {
	slog.Info("starting notification sender", "interval", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("notification sender stopped")
			return
		case <-ticker.C:
			sent, failed, err := s.ProcessPending(ctx)
			if err != nil {
				slog.Error("process pending notifications", "error", err)
			} else if sent > 0 || failed > 0 {
				slog.Info("notifications processed", "sent", sent, "failed", failed)
			}
		}
	}
}
