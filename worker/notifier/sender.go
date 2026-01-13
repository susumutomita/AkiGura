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

// pendingRow represents a row from the pending notifications query
type pendingRow struct {
	NotificationID string
	TeamID         string
	TeamName       string
	TeamEmail      string
	Channel        string
	SlotID         string
	SlotDate       string
	TimeFrom       string
	TimeTo         string
	CourtName      string
	FacilityName   string
	ReservationURL string
}

// ProcessPending sends all pending notifications, grouped by team
func (s *Sender) ProcessPending(ctx context.Context) (sent, failed int, err error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT n.id, n.team_id, n.channel, n.slot_id,
		       t.name as team_name, t.email as team_email,
		       sl.slot_date, sl.time_from, sl.time_to, COALESCE(sl.court_name, '') as court_name,
		       COALESCE(g.name, '') as facility_name,
		       COALESCE(m.url, '') as reservation_url
		FROM notifications n
		JOIN teams t ON n.team_id = t.id
		JOIN slots sl ON n.slot_id = sl.id
		LEFT JOIN grounds g ON sl.ground_id = g.id
		LEFT JOIN municipalities m ON sl.municipality_id = m.id
		WHERE n.status = 'pending'
		ORDER BY n.team_id, n.channel, sl.slot_date, sl.time_from
		LIMIT 500
	`)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	var pendingRows []pendingRow
	for rows.Next() {
		var r pendingRow
		if err := rows.Scan(&r.NotificationID, &r.TeamID, &r.Channel, &r.SlotID,
			&r.TeamName, &r.TeamEmail,
			&r.SlotDate, &r.TimeFrom, &r.TimeTo, &r.CourtName,
			&r.FacilityName, &r.ReservationURL); err != nil {
			slog.Warn("scan notification", "error", err)
			continue
		}
		pendingRows = append(pendingRows, r)
	}

	// Group by team + channel
	grouped := make(map[string][]pendingRow) // key: teamID:channel
	for _, r := range pendingRows {
		key := r.TeamID + ":" + r.Channel
		grouped[key] = append(grouped[key], r)
	}

	// Send one notification per team+channel with all slots
	for _, rows := range grouped {
		if len(rows) == 0 {
			continue
		}

		first := rows[0]
		n := &Notification{
			ID:        first.NotificationID, // Use first notification ID for logging
			TeamID:    first.TeamID,
			TeamName:  first.TeamName,
			TeamEmail: first.TeamEmail,
			Channel:   first.Channel,
			Slots:     make([]SlotInfo, 0, len(rows)),
		}

		// Collect all slots and notification IDs for this team
		var notificationIDs []string
		for _, r := range rows {
			slotTime := r.TimeFrom
			if r.TimeTo != "" {
				slotTime = r.TimeFrom + "-" + r.TimeTo
			}
			n.Slots = append(n.Slots, SlotInfo{
				SlotID:         r.SlotID,
				SlotDate:       r.SlotDate,
				SlotTime:       slotTime,
				CourtName:      r.CourtName,
				FacilityName:   r.FacilityName,
				ReservationURL: r.ReservationURL,
			})
			notificationIDs = append(notificationIDs, r.NotificationID)
		}

		// Send combined notification
		err := s.Manager.Send(ctx, n)
		if err != nil {
			slog.Warn("send notification failed", "team", n.TeamName, "slots", len(n.Slots), "error", err)
			for _, id := range notificationIDs {
				s.updateStatus(ctx, id, "failed")
			}
			failed += len(notificationIDs)
		} else {
			slog.Info("notification sent", "channel", n.Channel, "team", n.TeamName, "slots", len(n.Slots))
			for _, id := range notificationIDs {
				s.updateStatus(ctx, id, "sent")
			}
			sent += len(notificationIDs)
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
