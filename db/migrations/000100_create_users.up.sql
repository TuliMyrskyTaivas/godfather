CREATE USER moexmon WITH PASSWORD 'moexmon';
GRANT CONNECT ON DATABASE godfather TO moexmon;
GRANT USAGE ON SCHEMA public TO moexmon;
GRANT SELECT ON moex_assets, moex_watchlist, notifications TO moexmon;
GRANT UPDATE ON moex_watchlist TO moexmon;

CREATE USER squealer WITH PASSWORD 'squealer';
GRANT CONNECT ON DATABASE godfather TO squealer;
GRANT USAGE ON SCHEMA public TO squealer;
GRANT SELECT ON notifications TO squealer;
