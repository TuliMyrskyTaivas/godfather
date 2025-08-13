CREATE TABLE IF NOT EXISTS notifications (
    id BIGSERIAL PRIMARY KEY,
    tg_bot_token VARCHAR,
    tg_chat_id BIGINT,
    smtp_host VARCHAR,
    smtp_port INT CHECK (smtp_port > 0 AND smtp_port <= 65535),
    smtp_user VARCHAR,
    smtp_pass VARCHAR,
    smtp_mail_from VARCHAR,
    smtp_encryption_type TEXT NOT NULL CHECK (smtp_encryption_type IN ('none', 'ssl', 'tls', 'starttls')),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT valid_email CHECK (smtp_mail_from IS NULL OR smtp_mail_from = '' OR smtp_mail_from LIKE '%_@__%.__%')
);