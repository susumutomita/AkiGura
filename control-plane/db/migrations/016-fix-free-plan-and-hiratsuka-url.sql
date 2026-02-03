-- Fix free plan limits and Hiratsuka municipality URL

-- Free plan: keep max_grounds = 1 (1 rule), weekend_only = 1
-- (No change needed for max_grounds, it's already 1)

-- Fix Hiratsuka municipality URL
UPDATE municipalities 
SET url = 'https://shisetsu.city.hiratsuka.kanagawa.jp/cultos/reserve/gin_menu'
WHERE scraper_type = 'hiratsuka';

-- Record execution of this migration
INSERT OR IGNORE INTO migrations (migration_number, migration_name)
VALUES (016, '016-fix-free-plan-and-hiratsuka-url');
