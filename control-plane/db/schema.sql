-- AkiGura Database Schema
-- This file is used by sqlc to generate type-safe Go code

-- Migrations tracking
CREATE TABLE migrations (
    migration_number INTEGER PRIMARY KEY,
    migration_name TEXT NOT NULL,
    executed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Visitors (legacy)
CREATE TABLE visitors (
    id TEXT PRIMARY KEY,
    view_count INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    last_seen TIMESTAMP NOT NULL
);

-- Teams
CREATE TABLE teams (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    plan TEXT NOT NULL DEFAULT 'free', -- free, personal, pro, org
    status TEXT NOT NULL DEFAULT 'active', -- active, paused, cancelled
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Municipalities (one per scraper)
CREATE TABLE municipalities (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,           -- e.g., Yokohama, Ayase
    scraper_type TEXT NOT NULL UNIQUE,  -- e.g., yokohama, ayase
    url TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Grounds (watchable units within a municipality)
CREATE TABLE grounds (
    id TEXT PRIMARY KEY,
    municipality_id TEXT NOT NULL REFERENCES municipalities(id) ON DELETE CASCADE,
    name TEXT NOT NULL,           -- e.g., Shin-Yokohama Park Baseball Stadium
    court_pattern TEXT,           -- pattern to match court_name
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_grounds_municipality ON grounds(municipality_id);

-- Facilities (legacy, kept for compatibility)
CREATE TABLE facilities (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    municipality TEXT NOT NULL,
    scraper_type TEXT NOT NULL,
    url TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Plan Limits
CREATE TABLE plan_limits (
    plan TEXT PRIMARY KEY,        -- free, personal, pro, org
    max_grounds INTEGER NOT NULL,
    weekend_only INTEGER NOT NULL DEFAULT 0,
    max_conditions_per_ground INTEGER NOT NULL,
    notification_priority INTEGER NOT NULL DEFAULT 0
);

-- Watch Conditions
CREATE TABLE watch_conditions (
    id TEXT PRIMARY KEY,
    team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    facility_id TEXT NOT NULL REFERENCES facilities(id) ON DELETE CASCADE,
    days_of_week TEXT NOT NULL,
    time_from TEXT NOT NULL,
    time_to TEXT NOT NULL,
    date_from DATE,
    date_to DATE,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_watch_conditions_team ON watch_conditions(team_id);
CREATE INDEX idx_watch_conditions_facility ON watch_conditions(facility_id);

-- Slots (available time slots from scraping)
CREATE TABLE slots (
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
CREATE INDEX idx_slots_municipality ON slots(municipality_id);
CREATE INDEX idx_slots_ground ON slots(ground_id);
CREATE INDEX idx_slots_date ON slots(slot_date);
CREATE INDEX idx_slots_facility ON slots(facility_id);

-- Notifications
CREATE TABLE notifications (
    id TEXT PRIMARY KEY,
    team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    watch_condition_id TEXT NOT NULL REFERENCES watch_conditions(id) ON DELETE CASCADE,
    slot_id TEXT NOT NULL REFERENCES slots(id) ON DELETE CASCADE,
    channel TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    sent_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_notifications_team ON notifications(team_id);

-- Scrape Jobs
CREATE TABLE scrape_jobs (
    id TEXT PRIMARY KEY,
    municipality_id TEXT NOT NULL REFERENCES municipalities(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending',
    slots_found INTEGER DEFAULT 0,
    error_message TEXT,
    scrape_status TEXT,
    diagnostics TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_scrape_jobs_municipality ON scrape_jobs(municipality_id);

-- Support Tickets
CREATE TABLE support_tickets (
    id TEXT PRIMARY KEY,
    team_id TEXT REFERENCES teams(id) ON DELETE SET NULL,
    email TEXT NOT NULL,
    subject TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'open',
    priority TEXT NOT NULL DEFAULT 'normal',
    ai_response TEXT,
    human_response TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_support_tickets_team ON support_tickets(team_id);
CREATE INDEX idx_support_tickets_status ON support_tickets(status);

-- Support Messages
CREATE TABLE support_messages (
    id TEXT PRIMARY KEY,
    ticket_id TEXT NOT NULL REFERENCES support_tickets(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- System Metrics
CREATE TABLE system_metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    metric_name TEXT NOT NULL,
    metric_value REAL NOT NULL,
    recorded_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
