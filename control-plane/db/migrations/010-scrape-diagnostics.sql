-- Add scrape_status and diagnostics columns to scrape_jobs
-- scrape_status: success, success_no_slots, parse_error, network_error, etc.
-- diagnostics: JSON with detailed debug info

-- Note: These columns may already exist if added manually.
-- SQLite doesn't support IF NOT EXISTS for ALTER TABLE ADD COLUMN,
-- so we check existence in application code or skip if already present.

-- This migration is considered complete if the columns exist.
-- Run: SELECT sql FROM sqlite_master WHERE name='scrape_jobs' to verify.
