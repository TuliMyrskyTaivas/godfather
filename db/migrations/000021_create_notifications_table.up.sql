CREATE TABLE IF NOT EXISTS notifications (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR NOT NULL,
    tg_bot_token VARCHAR NOT NULL,
    tg_chat_id BIGINT
);