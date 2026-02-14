-- Add UNIQUE constraints to prevent duplicate records
-- This migration adds constraints to:
-- 1. grounds: prevent duplicate ground names per municipality
-- 2. watch_conditions: prevent duplicate conditions (same team + facility)

-- Create unique index on grounds (municipality_id, name)
-- This prevents auto-created grounds from creating duplicates
CREATE UNIQUE INDEX IF NOT EXISTS idx_grounds_municipality_name 
  ON grounds(municipality_id, name);

-- Create unique index on watch_conditions (team_id, facility_id)
-- This prevents duplicate watch conditions for the same team+facility
CREATE UNIQUE INDEX IF NOT EXISTS idx_watch_conditions_team_facility 
  ON watch_conditions(team_id, facility_id) WHERE enabled = 1;

-- Record execution of this migration
INSERT OR IGNORE INTO migrations (migration_number, migration_name)
VALUES (018, '018-unique-constraints');
