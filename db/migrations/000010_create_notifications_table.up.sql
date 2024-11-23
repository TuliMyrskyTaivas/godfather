CREATE TABLE IF NOT EXISTS notifications (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR,
    telegram VARCHAR
);