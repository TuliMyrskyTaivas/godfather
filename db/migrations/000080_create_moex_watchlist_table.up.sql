CREATE TABLE IF NOT EXISTS moex_watchlist (
    id BIGSERIAL PRIMARY KEY,
    ticker_id VARCHAR REFERENCES moex_assets,
    notification_id INTEGER REFERENCES notifications,
    target_price MONEY,
    condition VARCHAR NOT NULL,
    is_active BOOLEAN DEFAULT TRUE
);