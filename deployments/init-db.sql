BEGIN;

CREATE TABLE IF NOT EXISTS click_logs (
    id BIGSERIAL PRIMARY KEY,
    ip VARCHAR(45) NOT NULL,
    campaign_id VARCHAR(128) NOT NULL DEFAULT 'unknown',
    user_agent TEXT,
    is_bot BOOLEAN NOT NULL DEFAULT FALSE,
    reason VARCHAR(100),
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    country VARCHAR(64),
    risk_score DOUBLE PRECISION DEFAULT 0,
    risk_reasons TEXT
);

ALTER TABLE click_logs
    ADD COLUMN IF NOT EXISTS campaign_id VARCHAR(128) NOT NULL DEFAULT 'unknown',
    ADD COLUMN IF NOT EXISTS country VARCHAR(64),
    ADD COLUMN IF NOT EXISTS risk_score DOUBLE PRECISION DEFAULT 0,
    ADD COLUMN IF NOT EXISTS risk_reasons TEXT,
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

CREATE INDEX IF NOT EXISTS idx_click_logs_risk_score
    ON click_logs (risk_score);

CREATE INDEX IF NOT EXISTS idx_click_logs_country
    ON click_logs (country);


CREATE TABLE IF NOT EXISTS audit_events (
    id SERIAL PRIMARY KEY,
    action_text TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);


CREATE TABLE IF NOT EXISTS campaigns (
    campaign_id VARCHAR(128) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    budget NUMERIC DEFAULT 0,
    cost_per_click BIGINT NOT NULL DEFAULT 5,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO campaigns (campaign_id, name)

VALUES

    ('unknown', 'Unknown Campaign'),

    ('demo', 'Demo Campaign')

ON CONFLICT (campaign_id) DO NOTHING;


CREATE TABLE IF NOT EXISTS dynamic_blacklist (
    ip VARCHAR(45) PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    reason TEXT,
    expires_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_dynamic_blacklist_ip
    ON dynamic_blacklist(ip);

COMMIT;
