BEGIN;

INSERT INTO notifications (id, email, telegram) VALUES
(1, 'test@gmail.com', NULL),
(2, NULL, 'my_telegram_id');

INSERT INTO moex_assets (ticker, class_id, name) VALUES
('SBER', 'shares', 'Sberbank of Russia'),
('GAZP', 'shares', 'Gazprom'),
('USD000TSTTOM', 'currency', 'US Dollar TOM'),
('EUR_RUB_TOM', 'currency', 'Euro RUB TOM');

INSERT INTO moex_watchlist (id, ticker_id, notification_id, target_price, condition, is_active) VALUES
(1, 'SBER', 1, 300.00, '>', TRUE),
(2, 'GAZP', 1, 200.00, '<', TRUE),
(3, 'USD000TSTTOM', 1, 75.00, '>', TRUE),
(4, 'EUR_RUB_TOM', 1, 90.00, '<', TRUE);

COMMIT;

