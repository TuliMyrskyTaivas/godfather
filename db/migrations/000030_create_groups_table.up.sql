CREATE TABLE IF NOT EXISTS groups (
    id BIGSERIAL PRIMARY KEY,
    notification_id INTEGER REFERENCES notifications,
    name VARCHAR
);