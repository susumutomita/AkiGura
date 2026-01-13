-- Update scrape_jobs to reference municipalities instead of facilities
-- The worker now uses municipality_id for scrape jobs

-- SQLite doesn't support ALTER TABLE to change foreign keys
-- So we need to recreate the table

-- Step 1: Create new table with correct foreign key
CREATE TABLE IF NOT EXISTS scrape_jobs_new (
    id TEXT PRIMARY KEY,
    municipality_id TEXT NOT NULL REFERENCES municipalities(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending', -- pending, running, completed, failed
    slots_found INTEGER DEFAULT 0,
    error_message TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Step 2: Copy existing data (if any valid records exist)
INSERT OR IGNORE INTO scrape_jobs_new (id, municipality_id, status, slots_found, error_message, started_at, completed_at, created_at)
SELECT sj.id, sj.facility_id, sj.status, sj.slots_found, sj.error_message, sj.started_at, sj.completed_at, sj.created_at
FROM scrape_jobs sj
WHERE EXISTS (SELECT 1 FROM municipalities m WHERE m.id = sj.facility_id);

-- Step 3: Drop old table
DROP TABLE IF EXISTS scrape_jobs;

-- Step 4: Rename new table
ALTER TABLE scrape_jobs_new RENAME TO scrape_jobs;

-- Step 5: Create index for performance
CREATE INDEX IF NOT EXISTS idx_scrape_jobs_municipality ON scrape_jobs(municipality_id);

-- Record execution of this migration
INSERT OR IGNORE INTO migrations (migration_number, migration_name)
VALUES (006, '006-scrape-jobs-municipality');
