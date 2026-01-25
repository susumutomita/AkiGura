-- Add scrape_status and diagnostics columns to scrape_jobs
-- scrape_status: success, success_no_slots, parse_error, network_error, etc.
-- diagnostics: JSON with detailed debug info

ALTER TABLE scrape_jobs ADD COLUMN scrape_status TEXT;
ALTER TABLE scrape_jobs ADD COLUMN diagnostics TEXT;

-- Record execution of this migration
INSERT OR IGNORE INTO migrations (migration_number, migration_name)
VALUES (10, '010-scrape-diagnostics');
