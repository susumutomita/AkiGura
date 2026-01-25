package srv

import (
	"net/http"
	"time"
)

const (
	// HTTP Status Codes (for clarity and consistency)
	StatusOK                  = http.StatusOK
	StatusCreated             = http.StatusCreated
	StatusBadRequest          = http.StatusBadRequest
	StatusUnauthorized        = http.StatusUnauthorized
	StatusForbidden           = http.StatusForbidden
	StatusNotFound            = http.StatusNotFound
	StatusInternalServerError = http.StatusInternalServerError

	// Auth token expiry
	DefaultTokenExpiry = 15 * time.Minute

	// Database query limits
	DefaultAPILimit = 100
	MaxAPILimit     = 10000

	// Pagination defaults
	DefaultPageSize = 50
	MaxPageSize     = 500

	// Timeouts
	DefaultRequestTimeout = 30 * time.Second
	LongRequestTimeout    = 60 * time.Second

	// Subscription plans
	PlanFree     = "free"
	PlanPersonal = "personal"
	PlanPro      = "pro"
	PlanOrg      = "org"

	// Billing intervals
	BillingIntervalMonthly = "month"
	BillingIntervalYearly  = "year"

	// Scraper types
	ScraperTypeYokohama  = "yokohama"
	ScraperTypeKanagawa  = "kanagawa"
	ScraperTypeHiratsuka = "hiratsuka"
	ScraperTypeAyase     = "ayase"
	ScraperTypeKamakura  = "kamakura"
	ScraperTypeFujisawa  = "fujisawa"

	// Job status
	JobStatusPending   = "pending"
	JobStatusRunning   = "running"
	JobStatusCompleted = "completed"
	JobStatusFailed    = "failed"
)

// Supported scraper types
var SupportedScraperTypes = []string{
	ScraperTypeYokohama,
	ScraperTypeKanagawa,
	ScraperTypeHiratsuka,
	ScraperTypeAyase,
	ScraperTypeKamakura,
	ScraperTypeFujisawa,
}

// ValidatePlan checks if a plan is valid
func ValidatePlan(plan string) bool {
	switch plan {
	case PlanFree, PlanPersonal, PlanPro, PlanOrg:
		return true
	default:
		return false
	}
}

// ValidateScraperType checks if a scraper type is supported
func ValidateScraperType(scraperType string) bool {
	for _, st := range SupportedScraperTypes {
		if st == scraperType {
			return true
		}
	}
	return false
}
