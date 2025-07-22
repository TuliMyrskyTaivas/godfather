CREATE TABLE IF NOT EXISTS moex_alerts (
    id BIGSERIAL PRIMARY KEY,
    watchlist_id INTEGER REFERENCES moex_watchlist,
    timestamp TIMESTAMP NOT NULL
);