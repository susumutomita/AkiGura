// Package scraper provides facility availability scrapers for various municipalities.
package scraper

import (
	"context"
	"time"
)

// Slot represents an available time slot at a facility.
type Slot struct {
	Date      string // YYYY-MM-DD format
	TimeFrom  string // HH:MM format
	TimeTo    string // HH:MM format
	CourtName string // e.g., "大神グラウンド野球場Ａ面"
	RawText   string // Original text from the website
}

// Result represents the result of a scrape operation.
type Result struct {
	Success     bool
	Status      string
	Error       string
	Slots       []Slot
	ScrapedAt   time.Time
	Diagnostics map[string]interface{}
}

// Status codes for scrape results.
const (
	StatusSuccess       = "success"
	StatusSuccessEmpty  = "success_no_slots"
	StatusNetworkError  = "network_error"
	StatusParseError    = "parse_error"
	StatusUnknownError  = "unknown_error"
)

// Scraper defines the interface for facility scrapers.
type Scraper interface {
	// Scrape fetches available slots from the facility.
	Scrape(ctx context.Context) (*Result, error)
	// Name returns the scraper identifier.
	Name() string
}

// ExcludedPatterns contains patterns for facilities to exclude.
// These are not adult baseball/softball facilities.
var ExcludedPatterns = []string{
	"少年",      // Youth/junior fields
	"サッカー",    // Soccer
	"テニス",     // Tennis
	"ラグビー",    // Rugby
	"フットサル",  // Futsal
	"体育館",     // Gymnasium
	"プール",     // Pool
	"投球練習",   // Pitching practice
	"会議室",     // Meeting room
}

// ShouldExclude checks if a facility should be excluded based on its name.
func ShouldExclude(courtName string) bool {
	for _, pattern := range ExcludedPatterns {
		if contains(courtName, pattern) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsRunes([]rune(s), []rune(substr)))
}

func containsRunes(s, substr []rune) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if matchRunes(s[i:], substr) {
			return true
		}
	}
	return false
}

func matchRunes(s, substr []rune) bool {
	for i := 0; i < len(substr); i++ {
		if s[i] != substr[i] {
			return false
		}
	}
	return true
}
