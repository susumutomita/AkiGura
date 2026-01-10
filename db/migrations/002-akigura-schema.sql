-- AkiGura SaaS Schema
-- Teams, Facilities, WatchConditions, Notifications, Support Tickets

-- Teams (チーム)
CREATE TABLE IF NOT EXISTS teams (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    plan TEXT NOT NULL DEFAULT 'free', -- free, personal, pro, org
    status TEXT NOT NULL DEFAULT 'active', -- active, paused, cancelled
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Facilities (施設)
CREATE TABLE IF NOT EXISTS facilities (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    municipality TEXT NOT NULL, -- 自治体名
    scraper_type TEXT NOT NULL, -- スクレイパーの種類
    url TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Watch Conditions (監視条件)
CREATE TABLE IF NOT EXISTS watch_conditions (
    id TEXT PRIMARY KEY,
    team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    facility_id TEXT NOT NULL REFERENCES facilities(id) ON DELETE CASCADE,
    days_of_week TEXT NOT NULL, -- JSON array e.g. [0,6] for Sun, Sat
    time_from TEXT NOT NULL, -- HH:MM format
    time_to TEXT NOT NULL, -- HH:MM format
    date_from DATE,
    date_to DATE,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Slots (空き枠 - スクレイピング結果)
CREATE TABLE IF NOT EXISTS slots (
    id TEXT PRIMARY KEY,
    facility_id TEXT NOT NULL REFERENCES facilities(id) ON DELETE CASCADE,
    slot_date DATE NOT NULL,
    time_from TEXT NOT NULL,
    time_to TEXT NOT NULL,
    court_name TEXT,
    raw_text TEXT,
    scraped_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Notifications (通知履歴)
CREATE TABLE IF NOT EXISTS notifications (
    id TEXT PRIMARY KEY,
    team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    watch_condition_id TEXT NOT NULL REFERENCES watch_conditions(id) ON DELETE CASCADE,
    slot_id TEXT NOT NULL REFERENCES slots(id) ON DELETE CASCADE,
    channel TEXT NOT NULL, -- email, line, slack
    status TEXT NOT NULL DEFAULT 'pending', -- pending, sent, failed
    sent_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Support Tickets (AIサポート)
CREATE TABLE IF NOT EXISTS support_tickets (
    id TEXT PRIMARY KEY,
    team_id TEXT REFERENCES teams(id) ON DELETE SET NULL,
    email TEXT NOT NULL,
    subject TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'open', -- open, ai_handled, escalated, resolved
    priority TEXT NOT NULL DEFAULT 'normal', -- low, normal, high, urgent
    ai_response TEXT,
    human_response TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Support Messages (チケット内のメッセージ)
CREATE TABLE IF NOT EXISTS support_messages (
    id TEXT PRIMARY KEY,
    ticket_id TEXT NOT NULL REFERENCES support_tickets(id) ON DELETE CASCADE,
    role TEXT NOT NULL, -- user, ai, human_agent
    content TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Scrape Jobs (スクレイピングジョブ)
CREATE TABLE IF NOT EXISTS scrape_jobs (
    id TEXT PRIMARY KEY,
    facility_id TEXT NOT NULL REFERENCES facilities(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending', -- pending, running, completed, failed
    slots_found INTEGER DEFAULT 0,
    error_message TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- System Metrics (システムメトリクス)
CREATE TABLE IF NOT EXISTS system_metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    metric_name TEXT NOT NULL,
    metric_value REAL NOT NULL,
    recorded_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_watch_conditions_team ON watch_conditions(team_id);
CREATE INDEX IF NOT EXISTS idx_watch_conditions_facility ON watch_conditions(facility_id);
CREATE INDEX IF NOT EXISTS idx_slots_facility ON slots(facility_id);
CREATE INDEX IF NOT EXISTS idx_slots_date ON slots(slot_date);
CREATE INDEX IF NOT EXISTS idx_notifications_team ON notifications(team_id);
CREATE INDEX IF NOT EXISTS idx_support_tickets_team ON support_tickets(team_id);
CREATE INDEX IF NOT EXISTS idx_support_tickets_status ON support_tickets(status);

-- Record execution of this migration
INSERT OR IGNORE INTO migrations (migration_number, migration_name)
VALUES (002, '002-akigura-schema');
