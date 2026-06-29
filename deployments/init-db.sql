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
    id BIGSERIAL PRIMARY KEY,
    action_text TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_events_created_at
    ON audit_events (created_at DESC);
COMMIT;
