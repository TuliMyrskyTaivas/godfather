CREATE TABLE IF NOT EXISTS moex_assets (
    ticker VARCHAR PRIMARY KEY,
    class_id VARCHAR REFERENCES moex_api_url,
    name VARCHAR NOT NULL
);