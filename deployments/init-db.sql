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

-- ==========================================
-- USERS TABLE (Authentication & Authorization)
-- ==========================================
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    username VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Индексы для быстрого поиска
CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);
CREATE INDEX IF NOT EXISTS idx_users_username ON users (username);
CREATE INDEX IF NOT EXISTS idx_users_role ON users (role);

-- Триггер для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- "password123"
INSERT INTO users (email, username, password_hash, role) 
VALUES (
    'admin@antifraud.local',
    'admin',
    '$2a$10$X7V8Z9Y2W3U4T5S6R7Q8P9O0N1M2L3K4J5I6H7G8F9E0D1C2B3A4',
    'admin'
) ON CONFLICT (email) DO NOTHING;

COMMIT;
