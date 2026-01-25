package worker

import "time"

const (
	// HTTP Timeouts
	DefaultHTTPTimeout = 30 * time.Second
	LongHTTPTimeout    = 60 * time.Second

	// Database query limits
	DefaultQueryLimit = 100
	MaxQueryLimit     = 10000
	DefaultSlotLimit  = 500

	// Notification limits
	MaxNotificationSlots = 50

	// Worker intervals
	DefaultScrapeInterval = 15 * time.Minute
	MinScrapeInterval     = 1 * time.Minute

	// Status constants
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"

	// Notification status
	NotificationStatusPending = "pending"
	NotificationStatusSent    = "sent"
	NotificationStatusFailed  = "failed"

	// Facility types
	FacilityTypeBaseball  = "baseball"
	FacilityTypeMultiPose = "multi_purpose"
	FacilityTypeTennis    = "tennis"
	FacilityTypeGymnasium = "gymnasium"
	FacilityTypeSwimPool  = "swimming_pool"
	FacilityTypeMeetingRm = "meeting_room"
	FacilityTypeOther     = "other"
)
