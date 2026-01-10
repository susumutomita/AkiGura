package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// WatchCondition represents a user's watch condition
type WatchCondition struct {
	ID         string
	TeamID     string
	TeamEmail  string
	TeamName   string
	FacilityID string
	DaysOfWeek []int // 0=Sun, 1=Mon, ..., 6=Sat
	TimeFrom   string
	TimeTo     string
	DateFrom   *string
	DateTo     *string
}

// MatchedSlot represents a slot that matches a condition
type MatchedSlot struct {
	SlotID      string
	FacilityID  string
	Date        string
	TimeFrom    string
	TimeTo      string
	CourtName   string
	ConditionID string
	TeamID      string
	TeamEmail   string
	TeamName    string
}

// Matcher handles matching slots to watch conditions
type Matcher struct {
	DB *sql.DB
}

// NewMatcher creates a new Matcher
func NewMatcher(db *sql.DB) *Matcher {
	return &Matcher{DB: db}
}

// GetActiveConditions retrieves all active watch conditions for a facility
func (m *Matcher) GetActiveConditions(ctx context.Context, facilityID string) ([]WatchCondition, error) {
	rows, err := m.DB.QueryContext(ctx, `
		SELECT wc.id, wc.team_id, t.email, t.name, wc.facility_id, 
		       wc.days_of_week, wc.time_from, wc.time_to, wc.date_from, wc.date_to
		FROM watch_conditions wc
		JOIN teams t ON wc.team_id = t.id
		WHERE wc.facility_id = ? AND wc.enabled = 1 AND t.status = 'active'
	`, facilityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conditions []WatchCondition
	for rows.Next() {
		var c WatchCondition
		var daysJSON string
		if err := rows.Scan(&c.ID, &c.TeamID, &c.TeamEmail, &c.TeamName, &c.FacilityID,
			&daysJSON, &c.TimeFrom, &c.TimeTo, &c.DateFrom, &c.DateTo); err != nil {
			continue
		}
		// Parse days_of_week JSON array
		json.Unmarshal([]byte(daysJSON), &c.DaysOfWeek)
		conditions = append(conditions, c)
	}
	return conditions, nil
}

// GetNewSlots retrieves slots scraped recently that haven't been notified
func (m *Matcher) GetNewSlots(ctx context.Context, facilityID string, since time.Time) ([]MatchedSlot, error) {
	rows, err := m.DB.QueryContext(ctx, `
		SELECT id, facility_id, slot_date, time_from, time_to, COALESCE(court_name, '')
		FROM slots
		WHERE facility_id = ? AND scraped_at > ? AND slot_date >= date('now')
	`, facilityID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slots []MatchedSlot
	for rows.Next() {
		var s MatchedSlot
		if err := rows.Scan(&s.SlotID, &s.FacilityID, &s.Date, &s.TimeFrom, &s.TimeTo, &s.CourtName); err != nil {
			continue
		}
		slots = append(slots, s)
	}
	return slots, nil
}

// MatchSlot checks if a slot matches a condition
func (m *Matcher) MatchSlot(slot MatchedSlot, cond WatchCondition) bool {
	// Parse slot date
	slotDate, err := time.Parse("2006-01-02", slot.Date)
	if err != nil {
		return false
	}

	// Check day of week
	slotDOW := int(slotDate.Weekday())
	dayMatch := false
	for _, d := range cond.DaysOfWeek {
		if d == slotDOW {
			dayMatch = true
			break
		}
	}
	if !dayMatch && len(cond.DaysOfWeek) > 0 {
		return false
	}

	// Check time range
	slotFromMins := parseTimeToMinutes(slot.TimeFrom)
	slotToMins := parseTimeToMinutes(slot.TimeTo)
	condFromMins := parseTimeToMinutes(cond.TimeFrom)
	condToMins := parseTimeToMinutes(cond.TimeTo)

	// Slot must overlap with condition time range
	if slotToMins <= condFromMins || slotFromMins >= condToMins {
		return false
	}

	// Check date range
	if cond.DateFrom != nil {
		dateFrom, _ := time.Parse("2006-01-02", *cond.DateFrom)
		if slotDate.Before(dateFrom) {
			return false
		}
	}
	if cond.DateTo != nil {
		dateTo, _ := time.Parse("2006-01-02", *cond.DateTo)
		if slotDate.After(dateTo) {
			return false
		}
	}

	return true
}

func parseTimeToMinutes(t string) int {
	if len(t) < 4 {
		return 0
	}
	// Handle HH:MM or HHMM format
	var h, m int
	if len(t) == 5 && t[2] == ':' {
		h, _ = strconv.Atoi(t[0:2])
		m, _ = strconv.Atoi(t[3:5])
	} else if len(t) == 4 {
		h, _ = strconv.Atoi(t[0:2])
		m, _ = strconv.Atoi(t[2:4])
	}
	return h*60 + m
}

// CreateNotification creates a notification record
func (m *Matcher) CreateNotification(ctx context.Context, teamID, conditionID, slotID, channel string) error {
	// Check if already notified
	var exists int
	err := m.DB.QueryRowContext(ctx, `
		SELECT 1 FROM notifications 
		WHERE team_id = ? AND slot_id = ? AND watch_condition_id = ?
	`, teamID, slotID, conditionID).Scan(&exists)
	if err == nil {
		return nil // Already notified
	}

	id := uuid.New().String()
	_, err = m.DB.ExecContext(ctx, `
		INSERT INTO notifications (id, team_id, watch_condition_id, slot_id, channel, status, created_at)
		VALUES (?, ?, ?, ?, ?, 'pending', CURRENT_TIMESTAMP)
	`, id, teamID, conditionID, slotID, channel)
	return err
}

// ProcessMatches finds matches and creates notifications
func (m *Matcher) ProcessMatches(ctx context.Context, facilityID string, since time.Time) (int, error) {
	conditions, err := m.GetActiveConditions(ctx, facilityID)
	if err != nil {
		return 0, err
	}

	slots, err := m.GetNewSlots(ctx, facilityID, since)
	if err != nil {
		return 0, err
	}

	matches := 0
	for _, slot := range slots {
		for _, cond := range conditions {
			if m.MatchSlot(slot, cond) {
				if err := m.CreateNotification(ctx, cond.TeamID, cond.ID, slot.SlotID, "email"); err != nil {
					slog.Warn("failed to create notification", "error", err)
					continue
				}
				matches++
				slog.Info("match found", 
					"team", cond.TeamName, 
					"slot_date", slot.Date,
					"time", slot.TimeFrom+"-"+slot.TimeTo,
					"court", slot.CourtName)
			}
		}
	}

	return matches, nil
}
