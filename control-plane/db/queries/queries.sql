-- =============================================================================
-- Teams
-- =============================================================================

-- name: CreateTeam :one
INSERT INTO teams (id, name, email, plan, status, created_at, updated_at)
VALUES (?1, ?2, ?3, ?4, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
RETURNING *;

-- name: GetTeam :one
SELECT * FROM teams WHERE id = ?;

-- name: GetTeamByEmail :one
SELECT * FROM teams WHERE email = ?;

-- name: ListTeams :many
SELECT * FROM teams ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: CountTeams :one
SELECT COUNT(*) as count FROM teams;

-- name: CountTeamsByPlan :many
SELECT plan, COUNT(*) as count FROM teams GROUP BY plan;

-- name: UpdateTeam :exec
UPDATE teams SET name = ?2, email = ?3, plan = ?4, status = ?5, updated_at = CURRENT_TIMESTAMP WHERE id = ?1;

-- name: DeleteTeam :exec
DELETE FROM teams WHERE id = ?;

-- =============================================================================
-- Municipalities (municipalities)
-- =============================================================================

-- name: ListMunicipalities :many
SELECT * FROM municipalities WHERE enabled = 1 ORDER BY name;

-- name: ListAllMunicipalities :many
SELECT * FROM municipalities ORDER BY name;

-- name: GetMunicipality :one
SELECT * FROM municipalities WHERE id = ?;

-- name: GetMunicipalityByScraperType :one
SELECT * FROM municipalities WHERE scraper_type = ?;

-- name: CreateMunicipality :one
INSERT INTO municipalities (id, name, scraper_type, url, enabled, created_at)
VALUES (?1, ?2, ?3, ?4, ?5, CURRENT_TIMESTAMP)
RETURNING *;

-- =============================================================================
-- Grounds (grounds)
-- =============================================================================

-- name: ListGrounds :many
SELECT g.id, g.municipality_id, g.name, g.court_pattern, g.enabled, g.created_at,
       m.name as municipality_name, m.scraper_type
FROM grounds g
JOIN municipalities m ON g.municipality_id = m.id
WHERE g.enabled = 1
ORDER BY m.name, g.name;

-- name: ListGroundsByMunicipality :many
SELECT * FROM grounds WHERE municipality_id = ? AND enabled = 1 ORDER BY name;

-- name: GetGround :one
SELECT g.id, g.municipality_id, g.name, g.court_pattern, g.enabled, g.created_at,
       m.name as municipality_name, m.scraper_type
FROM grounds g
JOIN municipalities m ON g.municipality_id = m.id
WHERE g.id = ?;

-- name: CreateGround :one
INSERT INTO grounds (id, municipality_id, name, court_pattern, enabled, created_at)
VALUES (?1, ?2, ?3, ?4, 1, CURRENT_TIMESTAMP)
RETURNING *;

-- name: MatchGroundByCourtName :one
SELECT g.id FROM grounds g
WHERE g.municipality_id = ?1 AND instr(?2, g.court_pattern) > 0
LIMIT 1;

-- =============================================================================
-- Facilities (legacy)
-- =============================================================================

-- name: CreateFacility :one
INSERT INTO facilities (id, name, municipality, scraper_type, url, enabled, created_at)
VALUES (?1, ?2, ?3, ?4, ?5, 1, CURRENT_TIMESTAMP)
RETURNING *;

-- name: GetFacility :one
SELECT * FROM facilities WHERE id = ?;

-- name: ListFacilities :many
SELECT * FROM facilities ORDER BY municipality, name LIMIT ? OFFSET ?;

-- name: CountFacilities :one
SELECT COUNT(*) as count FROM facilities;

-- name: ListEnabledFacilities :many
SELECT * FROM facilities WHERE enabled = 1 ORDER BY municipality, name;

-- name: UpdateFacility :exec
UPDATE facilities SET name = ?2, municipality = ?3, scraper_type = ?4, url = ?5, enabled = ?6 WHERE id = ?1;

-- name: DeleteFacility :exec
DELETE FROM facilities WHERE id = ?;

-- =============================================================================
-- Plan Limits
-- =============================================================================

-- name: GetPlanLimits :one
SELECT * FROM plan_limits WHERE plan = ?;

-- =============================================================================
-- Watch Conditions
-- =============================================================================

-- name: CreateWatchCondition :one
INSERT INTO watch_conditions (id, team_id, facility_id, days_of_week, time_from, time_to, date_from, date_to, enabled, created_at, updated_at)
VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
RETURNING *;

-- name: GetWatchCondition :one
SELECT * FROM watch_conditions WHERE id = ?;

-- name: ListWatchConditionsByTeam :many
SELECT * FROM watch_conditions WHERE team_id = ? ORDER BY created_at DESC;

-- name: ListWatchConditionsByFacility :many
SELECT wc.*, t.email as team_email, t.name as team_name
FROM watch_conditions wc
JOIN teams t ON wc.team_id = t.id
WHERE wc.facility_id = ? AND wc.enabled = 1 AND t.status = 'active';

-- name: CountWatchConditions :one
SELECT COUNT(*) as count FROM watch_conditions;

-- name: UpdateWatchCondition :exec
UPDATE watch_conditions SET facility_id = ?2, days_of_week = ?3, time_from = ?4, time_to = ?5, date_from = ?6, date_to = ?7, enabled = ?8, updated_at = CURRENT_TIMESTAMP WHERE id = ?1;

-- name: DeleteWatchCondition :exec
DELETE FROM watch_conditions WHERE id = ?;

-- =============================================================================
-- Slots
-- =============================================================================

-- name: UpsertSlot :one
INSERT INTO slots (id, facility_id, municipality_id, ground_id, slot_date, time_from, time_to, court_name, raw_text, scraped_at)
VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, CURRENT_TIMESTAMP)
ON CONFLICT(municipality_id, slot_date, time_from, time_to, court_name) DO UPDATE SET
    ground_id = COALESCE(excluded.ground_id, slots.ground_id),
    raw_text = excluded.raw_text,
    scraped_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetSlot :one
SELECT * FROM slots WHERE id = ?;

-- name: ListSlotsByFacility :many
SELECT * FROM slots WHERE facility_id = ? AND slot_date >= date('now') ORDER BY slot_date, time_from;

-- name: ListSlotsByMunicipality :many
SELECT * FROM slots WHERE municipality_id = ? AND slot_date >= date('now') ORDER BY slot_date, time_from;

-- name: ListSlotsByGround :many
SELECT * FROM slots WHERE ground_id = ? AND slot_date >= date('now') ORDER BY slot_date, time_from;

-- name: ListSlotsByDateRange :many
SELECT * FROM slots WHERE municipality_id = ? AND slot_date BETWEEN ? AND ? ORDER BY slot_date, time_from;

-- name: CountSlots :one
SELECT COUNT(*) as count FROM slots;

-- name: CountSlotsByMunicipality :many
SELECT municipality_id, COUNT(*) as count FROM slots WHERE slot_date >= date('now') GROUP BY municipality_id;

-- name: DeleteOldSlots :exec
DELETE FROM slots WHERE slot_date < date('now', '-7 days');

-- =============================================================================
-- Notifications
-- =============================================================================

-- name: CreateNotification :one
INSERT INTO notifications (id, team_id, watch_condition_id, slot_id, channel, status, created_at)
VALUES (?1, ?2, ?3, ?4, ?5, 'pending', CURRENT_TIMESTAMP)
RETURNING *;

-- name: GetNotification :one
SELECT * FROM notifications WHERE id = ?;

-- name: ListNotificationsByTeam :many
SELECT * FROM notifications WHERE team_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: UpdateNotificationStatus :exec
UPDATE notifications SET status = ?2, sent_at = CASE WHEN ?2 = 'sent' THEN CURRENT_TIMESTAMP ELSE sent_at END WHERE id = ?1;

-- name: CountNotifications :one
SELECT COUNT(*) as count FROM notifications;

-- name: CountNotificationsByStatus :many
SELECT status, COUNT(*) as count FROM notifications GROUP BY status;

-- name: CountNotificationsToday :one
SELECT COUNT(*) as count FROM notifications WHERE date(created_at) = date('now');

-- =============================================================================
-- Scrape Jobs
-- =============================================================================

-- name: CreateScrapeJob :one
INSERT INTO scrape_jobs (id, municipality_id, status, created_at)
VALUES (?1, ?2, 'pending', CURRENT_TIMESTAMP)
RETURNING *;

-- name: GetScrapeJob :one
SELECT * FROM scrape_jobs WHERE id = ?;

-- name: ListRecentScrapeJobs :many
SELECT sj.*, m.name as municipality_name, m.scraper_type
FROM scrape_jobs sj
JOIN municipalities m ON sj.municipality_id = m.id
ORDER BY sj.created_at DESC LIMIT ?;

-- name: UpdateScrapeJobStatus :exec
UPDATE scrape_jobs SET
    status = ?2,
    slots_found = ?3,
    error_message = ?4,
    scrape_status = ?5,
    diagnostics = ?6,
    started_at = CASE WHEN ?2 = 'running' THEN CURRENT_TIMESTAMP ELSE started_at END,
    completed_at = CASE WHEN ?2 IN ('completed', 'failed') THEN CURRENT_TIMESTAMP ELSE completed_at END
WHERE id = ?1;

-- name: CountScrapeJobsByStatus :many
SELECT status, COUNT(*) as count FROM scrape_jobs WHERE created_at > datetime('now', '-24 hours') GROUP BY status;

-- =============================================================================
-- Support Tickets
-- =============================================================================

-- name: CreateSupportTicket :one
INSERT INTO support_tickets (id, team_id, email, subject, status, priority, created_at, updated_at)
VALUES (?1, ?2, ?3, ?4, 'open', 'normal', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
RETURNING *;

-- name: GetSupportTicket :one
SELECT * FROM support_tickets WHERE id = ?;

-- name: ListSupportTickets :many
SELECT * FROM support_tickets ORDER BY
    CASE priority WHEN 'urgent' THEN 1 WHEN 'high' THEN 2 WHEN 'normal' THEN 3 ELSE 4 END,
    created_at DESC
LIMIT ? OFFSET ?;

-- name: ListOpenSupportTickets :many
SELECT * FROM support_tickets WHERE status IN ('open', 'escalated') ORDER BY
    CASE priority WHEN 'urgent' THEN 1 WHEN 'high' THEN 2 WHEN 'normal' THEN 3 ELSE 4 END,
    created_at DESC;

-- name: CountSupportTickets :one
SELECT COUNT(*) as count FROM support_tickets;

-- name: CountSupportTicketsByStatus :many
SELECT status, COUNT(*) as count FROM support_tickets GROUP BY status;

-- name: UpdateSupportTicket :exec
UPDATE support_tickets SET status = ?2, priority = ?3, ai_response = ?4, human_response = ?5, updated_at = CURRENT_TIMESTAMP WHERE id = ?1;

-- name: UpdateSupportTicketAIResponse :exec
UPDATE support_tickets SET ai_response = ?2, status = 'ai_handled', updated_at = CURRENT_TIMESTAMP WHERE id = ?1;

-- =============================================================================
-- Support Messages
-- =============================================================================

-- name: CreateSupportMessage :one
INSERT INTO support_messages (id, ticket_id, role, content, created_at)
VALUES (?1, ?2, ?3, ?4, CURRENT_TIMESTAMP)
RETURNING *;

-- name: ListSupportMessagesByTicket :many
SELECT * FROM support_messages WHERE ticket_id = ? ORDER BY created_at ASC;

-- =============================================================================
-- System Metrics
-- =============================================================================

-- name: RecordMetric :exec
INSERT INTO system_metrics (metric_name, metric_value, recorded_at)
VALUES (?1, ?2, CURRENT_TIMESTAMP);

-- name: GetLatestMetric :one
SELECT * FROM system_metrics WHERE metric_name = ? ORDER BY recorded_at DESC LIMIT 1;

-- name: GetMetricsHistory :many
SELECT * FROM system_metrics WHERE metric_name = ? AND recorded_at > datetime('now', ?) ORDER BY recorded_at ASC;

-- =============================================================================
-- Visitors (legacy)
-- =============================================================================

-- name: UpsertVisitor :one
INSERT INTO visitors (id, view_count, created_at, last_seen)
VALUES (?1, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT(id) DO UPDATE SET
    view_count = visitors.view_count + 1,
    last_seen = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetVisitor :one
SELECT * FROM visitors WHERE id = ?;

-- name: CountVisitors :one
SELECT COUNT(*) as count FROM visitors;
