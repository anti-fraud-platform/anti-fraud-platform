CREATE TABLE IF NOT EXISTS click_logs (
    id BIGSERIAL PRIMARY KEY,
    ip VARCHAR(45) NOT NULL,
    user_agent TEXT,
    is_bot BOOLEAN DEFAULT FALSE,
    reason VARCHAR(100),
    processed_at TIMESTAMPTZ DEFAULT NOW()
);