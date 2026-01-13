-- Add municipality_id and ground_id to slots table
-- This allows slots to be linked to the new municipalities/grounds structure

-- Create new slots table with updated schema
CREATE TABLE IF NOT EXISTS slots_new (
    id TEXT PRIMARY KEY,
    facility_id TEXT REFERENCES facilities(id) ON DELETE SET NULL,
    municipality_id TEXT REFERENCES municipalities(id) ON DELETE CASCADE,
    ground_id TEXT REFERENCES grounds(id) ON DELETE SET NULL,
    slot_date DATE NOT NULL,
    time_from TEXT NOT NULL,
    time_to TEXT NOT NULL,
    court_name TEXT,
    raw_text TEXT,
    scraped_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(municipality_id, slot_date, time_from, time_to, court_name)
);

-- Migrate existing data
INSERT INTO slots_new (id, facility_id, municipality_id, slot_date, time_from, time_to, court_name, raw_text, scraped_at)
SELECT id, facility_id, facility_id, slot_date, time_from, time_to, court_name, raw_text, scraped_at
FROM slots;

-- Drop old table and rename new one
DROP TABLE IF EXISTS slots;
ALTER TABLE slots_new RENAME TO slots;

-- Recreate indexes
CREATE INDEX IF NOT EXISTS idx_slots_municipality ON slots(municipality_id);
CREATE INDEX IF NOT EXISTS idx_slots_ground ON slots(ground_id);
CREATE INDEX IF NOT EXISTS idx_slots_date ON slots(slot_date);

-- Record execution of this migration
INSERT OR IGNORE INTO migrations (migration_number, migration_name)
VALUES (007, '007-slots-municipality');
