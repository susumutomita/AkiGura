-- Billing: Add Stripe fields to teams and promo codes table

-- Add Stripe fields to teams (use ALTER TABLE only if column doesn't exist)
-- SQLite doesn't support IF NOT EXISTS for ALTER TABLE, so we check via pragma
-- Since this might already be partially applied, we wrap in separate statements
-- that can fail individually

-- Create promo_codes table first (this is safe to run multiple times with CREATE IF NOT EXISTS)
CREATE TABLE IF NOT EXISTS promo_codes (
    id TEXT PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    discount_type TEXT NOT NULL, -- percent, fixed
    discount_value INTEGER NOT NULL, -- percent (e.g., 20) or fixed amount in JPY
    applies_to TEXT, -- NULL = all plans, or specific plan ID
    valid_from TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    valid_until TIMESTAMP,
    max_uses INTEGER, -- NULL = unlimited
    uses_count INTEGER NOT NULL DEFAULT 0,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_promo_codes_code ON promo_codes(code);

-- Promo Code Usage (track which teams used which codes)
CREATE TABLE IF NOT EXISTS promo_code_usages (
    id TEXT PRIMARY KEY,
    promo_code_id TEXT NOT NULL REFERENCES promo_codes(id) ON DELETE CASCADE,
    team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(promo_code_id, team_id)
);
CREATE INDEX IF NOT EXISTS idx_promo_code_usages_team ON promo_code_usages(team_id);
