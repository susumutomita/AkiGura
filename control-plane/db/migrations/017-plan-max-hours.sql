-- Add max_hours column to plan_limits (for future use)
-- Currently all plans allow 24 hours (no restriction)

ALTER TABLE plan_limits ADD COLUMN max_hours INTEGER NOT NULL DEFAULT 24;

-- Record execution of this migration
INSERT OR IGNORE INTO migrations (migration_number, migration_name)
VALUES (017, '017-plan-max-hours');
