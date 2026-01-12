-- Separate municipalities from facilities
-- municipalities = 自治体（スクレイパー単位）
-- facilities = グラウンド/施設（監視単位）

-- Municipalities (自治体)
CREATE TABLE IF NOT EXISTS municipalities (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,           -- 横浜市、綾瀬市など
    scraper_type TEXT NOT NULL UNIQUE,  -- yokohama, ayase など
    url TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Grounds (グラウンド/施設 - 監視単位)
-- 既存のfacilitiesを置き換えるために新テーブルを作成
CREATE TABLE IF NOT EXISTS grounds (
    id TEXT PRIMARY KEY,
    municipality_id TEXT NOT NULL REFERENCES municipalities(id) ON DELETE CASCADE,
    name TEXT NOT NULL,           -- 新横浜公園野球場、綾瀬ノーブルスタジアムなど
    court_pattern TEXT,           -- スクレイプ結果のcourt_nameとマッチさせるパターン
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Plan limits table
CREATE TABLE IF NOT EXISTS plan_limits (
    plan TEXT PRIMARY KEY,        -- free, personal, pro, org
    max_grounds INTEGER NOT NULL,
    weekend_only INTEGER NOT NULL DEFAULT 0,  -- 1 = 週末のみ監視可
    max_conditions_per_ground INTEGER NOT NULL,
    notification_priority INTEGER NOT NULL DEFAULT 0  -- 高いほど優先
);

-- Insert default plan limits
INSERT OR IGNORE INTO plan_limits (plan, max_grounds, weekend_only, max_conditions_per_ground, notification_priority)
VALUES 
    ('free', 1, 1, 1, 0),      -- 1施設、週末のみ、条件1つ
    ('personal', 3, 0, 3, 1),  -- 3施設、全曜日、条件3つ
    ('pro', 10, 0, 10, 2),     -- 10施設、全曜日、条件10、優先通知
    ('org', 999, 0, 999, 3);   -- 無制限

CREATE INDEX IF NOT EXISTS idx_grounds_municipality ON grounds(municipality_id);

-- Migrate existing data from facilities to municipalities
-- (Existing facilities table had municipality-level data)
INSERT OR IGNORE INTO municipalities (id, name, scraper_type, url, enabled, created_at)
SELECT id, name, scraper_type, url, enabled, created_at FROM facilities;

-- Record execution of this migration
INSERT OR IGNORE INTO migrations (migration_number, migration_name)
VALUES (003, '003-municipalities');
