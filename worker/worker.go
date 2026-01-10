package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"time"

	"github.com/google/uuid"
)

// ScraperResult represents the JSON output from scraper_wrapper.py
type ScraperResult struct {
	Success      bool   `json:"success"`
	Error        string `json:"error"`
	FacilityType string `json:"facility_type"`
	Slots        []Slot `json:"slots"`
	ScrapedAt    string `json:"scraped_at"`
}

// Slot represents a single available time slot
type Slot struct {
	Date         *string `json:"date"`
	TimeFrom     *string `json:"time_from"`
	TimeTo       *string `json:"time_to"`
	CourtName    *string `json:"court_name"`
	RawText      string  `json:"raw_text"`
	FacilityType string  `json:"facility_type"`
}

// Worker handles scraping jobs
type Worker struct {
	DB            *sql.DB
	ScraperPath   string
	PythonPath    string
}

// NewWorker creates a new Worker instance
func NewWorker(db *sql.DB, scraperPath, pythonPath string) *Worker {
	if pythonPath == "" {
		pythonPath = "python3"
	}
	return &Worker{
		DB:          db,
		ScraperPath: scraperPath,
		PythonPath:  pythonPath,
	}
}

// RunScraper executes the Python scraper for a facility
func (w *Worker) RunScraper(ctx context.Context, facilityType string) (*ScraperResult, error) {
	slog.Info("running scraper", "facility_type", facilityType)

	cmd := exec.CommandContext(ctx, w.PythonPath, w.ScraperPath, facilityType)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("scraper execution failed: %w", err)
	}

	var result ScraperResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse scraper output: %w", err)
	}

	return &result, nil
}

// SaveSlots saves scraped slots to the database
func (w *Worker) SaveSlots(ctx context.Context, facilityID string, slots []Slot) (int, error) {
	saved := 0
	for _, slot := range slots {
		if slot.Date == nil {
			continue // Skip slots without parseable dates
		}

		id := uuid.New().String()
		timeFrom := ""
		timeTo := ""
		courtName := ""
		if slot.TimeFrom != nil {
			timeFrom = *slot.TimeFrom
		}
		if slot.TimeTo != nil {
			timeTo = *slot.TimeTo
		}
		if slot.CourtName != nil {
			courtName = *slot.CourtName
		}

		_, err := w.DB.ExecContext(ctx, `
			INSERT INTO slots (id, facility_id, slot_date, time_from, time_to, court_name, raw_text, scraped_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT DO NOTHING
		`, id, facilityID, *slot.Date, timeFrom, timeTo, courtName, slot.RawText)
		if err != nil {
			slog.Warn("failed to save slot", "error", err)
			continue
		}
		saved++
	}
	return saved, nil
}

// CreateJob creates a scrape job record
func (w *Worker) CreateJob(ctx context.Context, facilityID string) (string, error) {
	id := uuid.New().String()
	_, err := w.DB.ExecContext(ctx, `
		INSERT INTO scrape_jobs (id, facility_id, status, created_at)
		VALUES (?, ?, 'pending', CURRENT_TIMESTAMP)
	`, id, facilityID)
	if err != nil {
		return "", err
	}
	return id, nil
}

// UpdateJob updates a scrape job status
func (w *Worker) UpdateJob(ctx context.Context, jobID, status string, slotsFound int, errorMsg string) error {
	_, err := w.DB.ExecContext(ctx, `
		UPDATE scrape_jobs SET 
			status = ?,
			slots_found = ?,
			error_message = ?,
			started_at = CASE WHEN ? = 'running' THEN CURRENT_TIMESTAMP ELSE started_at END,
			completed_at = CASE WHEN ? IN ('completed', 'failed') THEN CURRENT_TIMESTAMP ELSE completed_at END
		WHERE id = ?
	`, status, slotsFound, errorMsg, status, status, jobID)
	return err
}

// ProcessFacility runs the full scrape process for a facility
func (w *Worker) ProcessFacility(ctx context.Context, facilityID, scraperType string) error {
	// Create job
	jobID, err := w.CreateJob(ctx, facilityID)
	if err != nil {
		return fmt.Errorf("create job: %w", err)
	}

	// Mark as running
	if err := w.UpdateJob(ctx, jobID, "running", 0, ""); err != nil {
		return fmt.Errorf("update job running: %w", err)
	}

	// Run scraper
	result, err := w.RunScraper(ctx, scraperType)
	if err != nil {
		w.UpdateJob(ctx, jobID, "failed", 0, err.Error())
		return fmt.Errorf("run scraper: %w", err)
	}

	if !result.Success {
		w.UpdateJob(ctx, jobID, "failed", 0, result.Error)
		return fmt.Errorf("scraper error: %s", result.Error)
	}

	// Save slots
	saved, err := w.SaveSlots(ctx, facilityID, result.Slots)
	if err != nil {
		w.UpdateJob(ctx, jobID, "failed", saved, err.Error())
		return fmt.Errorf("save slots: %w", err)
	}

	// Mark as completed
	if err := w.UpdateJob(ctx, jobID, "completed", saved, ""); err != nil {
		return fmt.Errorf("update job completed: %w", err)
	}

	slog.Info("scrape completed", "facility_id", facilityID, "slots_found", len(result.Slots), "slots_saved", saved)
	return nil
}

// ProcessAllFacilities processes all enabled facilities
func (w *Worker) ProcessAllFacilities(ctx context.Context) error {
	rows, err := w.DB.QueryContext(ctx, `
		SELECT id, scraper_type FROM facilities WHERE enabled = 1
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var facilities []struct {
		ID          string
		ScraperType string
	}
	for rows.Next() {
		var f struct {
			ID          string
			ScraperType string
		}
		if err := rows.Scan(&f.ID, &f.ScraperType); err != nil {
			continue
		}
		facilities = append(facilities, f)
	}

	for _, f := range facilities {
		if err := w.ProcessFacility(ctx, f.ID, f.ScraperType); err != nil {
			slog.Error("failed to process facility", "facility_id", f.ID, "error", err)
		}
	}

	return nil
}

// StartScheduler starts a periodic scraping loop
func (w *Worker) StartScheduler(ctx context.Context, interval time.Duration) {
	slog.Info("starting scraper scheduler", "interval", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run immediately on start
	w.ProcessAllFacilities(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("scheduler stopped")
			return
		case <-ticker.C:
			w.ProcessAllFacilities(ctx)
		}
	}
}
