package notifier

import (
	"context"
	"fmt"
)

// SlotInfo represents a single slot in a notification
type SlotInfo struct {
	SlotID         string
	SlotDate       string
	SlotTime       string
	CourtName      string
	FacilityName   string // ground name
	ReservationURL string // URL to the reservation system
}

// Notification represents a notification to be sent
// Contains multiple slots for batch notification
type Notification struct {
	ID        string
	TeamID    string
	TeamName  string
	TeamEmail string
	Channel   string // email, line, slack
	Slots     []SlotInfo
}

// Notifier interface for sending notifications
type Notifier interface {
	Send(ctx context.Context, n *Notification) error
	Channel() string
}

// Result represents the result of a notification attempt
type Result struct {
	NotificationID string
	Success        bool
	Error          error
}

// Manager manages multiple notifiers
type Manager struct {
	notifiers map[string]Notifier
}

// NewManager creates a new notification manager
func NewManager() *Manager {
	return &Manager{
		notifiers: make(map[string]Notifier),
	}
}

// Register adds a notifier for a channel
func (m *Manager) Register(n Notifier) {
	m.notifiers[n.Channel()] = n
}

// Send sends a notification using the appropriate channel
func (m *Manager) Send(ctx context.Context, n *Notification) error {
	notifier, ok := m.notifiers[n.Channel]
	if !ok {
		return fmt.Errorf("no notifier registered for channel: %s", n.Channel)
	}
	return notifier.Send(ctx, n)
}

// SendAll sends notifications using all registered channels
func (m *Manager) SendAll(ctx context.Context, n *Notification) []Result {
	var results []Result
	for _, notifier := range m.notifiers {
		err := notifier.Send(ctx, n)
		results = append(results, Result{
			NotificationID: n.ID,
			Success:        err == nil,
			Error:          err,
		})
	}
	return results
}
