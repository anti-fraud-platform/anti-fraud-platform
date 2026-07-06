BEGIN;

CREATE TABLE IF NOT EXISTS click_logs (
    id BIGSERIAL PRIMARY KEY,
    ip VARCHAR(45) NOT NULL,
    campaign_id VARCHAR(128) NOT NULL DEFAULT 'unknown',
    user_agent TEXT,
    is_bot BOOLEAN NOT NULL DEFAULT FALSE,
    reason VARCHAR(100),
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE click_logs
    ADD COLUMN IF NOT EXISTS campaign_id VARCHAR(128) NOT NULL DEFAULT 'unknown',
    ALTER COLUMN is_bot SET DEFAULT FALSE,
    ALTER COLUMN processed_at SET DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_click_logs_ip
    ON click_logs (ip);

CREATE INDEX IF NOT EXISTS idx_click_logs_campaign_id
    ON click_logs (campaign_id);

CREATE INDEX IF NOT EXISTS idx_click_logs_processed_at
    ON click_logs (processed_at DESC);

CREATE INDEX IF NOT EXISTS idx_click_logs_campaign_processed_at
    ON click_logs (campaign_id, processed_at DESC);


CREATE TABLE IF NOT EXISTS audit_events (
    id SERIAL PRIMARY KEY,
    action_text TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- campaign cost table, dynamic blacklist, extra log columns

CREATE TABLE IF NOT EXISTS campaigns (
    campaign_id VARCHAR(128) PRIMARY KEY,
    cost_per_click DECIMAL(10,2) NOT NULL DEFAULT 5.00
);

ALTER TABLE click_logs
    ADD COLUMN IF NOT EXISTS country CHAR(2),
    ADD COLUMN IF NOT EXISTS risk_score INT,
    ADD COLUMN IF NOT EXISTS risk_reasons TEXT;

CREATE TABLE IF NOT EXISTS dynamic_blacklist (
    ip VARCHAR(45) PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    reason TEXT,
    expires_at TIMESTAMPTZ
);

COMMIT;
