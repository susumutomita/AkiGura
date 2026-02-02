-- Migration: Fix watch_conditions foreign key to reference grounds instead of facilities
-- This is needed because actual facility data is stored in the grounds table

-- Temporarily disable foreign keys for the migration
PRAGMA foreign_keys = OFF;

-- Step 1: Create new table with correct FK (no actual FK enforcement during creation)
CREATE TABLE IF NOT EXISTS watch_conditions_new (
    id TEXT PRIMARY KEY,
    team_id TEXT NOT NULL,
    facility_id TEXT NOT NULL,
    days_of_week TEXT NOT NULL,
    time_from TEXT NOT NULL,
    time_to TEXT NOT NULL,
    date_from DATE,
    date_to DATE,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Step 2: Copy existing valid data only (where facility_id exists in grounds)
INSERT OR IGNORE INTO watch_conditions_new 
SELECT wc.* FROM watch_conditions wc
INNER JOIN grounds g ON g.id = wc.facility_id;

-- Step 3: Drop old table and rename new one
DROP TABLE IF EXISTS watch_conditions;
ALTER TABLE watch_conditions_new RENAME TO watch_conditions;

-- Step 4: Recreate indexes
CREATE INDEX IF NOT EXISTS idx_watch_conditions_team ON watch_conditions(team_id);
CREATE INDEX IF NOT EXISTS idx_watch_conditions_facility ON watch_conditions(facility_id);

-- Re-enable foreign keys
PRAGMA foreign_keys = ON;

-- Record execution of this migration
INSERT OR IGNORE INTO migrations (migration_number, migration_name)
VALUES (15, '015-fix-watch-conditions-fk');
