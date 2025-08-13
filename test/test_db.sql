BEGIN;

INSERT INTO notifications (id, tg_bot_token, tg_chat_id, smtp_host, smtp_port, smtp_user, smtp_pass, smtp_mail_from, smtp_encryption_type, created_at, updated_at) VALUES
(1, 'api token', 12345678, NULL, 1, NULL, NULL, NULL, 'none', NOW(), NOW());

INSERT INTO moex_assets (ticker, class_id, name) VALUES
('SBER', 'stock', 'Sberbank of Russia'),
('GAZP', 'stock', 'Gazprom'),
('USD000TSTTOM', 'currency', 'US Dollar TOM'),
('EUR_RUB_TOM', 'currency', 'Euro RUB TOM');

INSERT INTO moex_watchlist (id, ticker_id, notification_id, target_price, condition, is_active) VALUES
(1, 'SBER', 1, 300.00, 'above', TRUE),
(2, 'GAZP', 1, 200.00, 'below', TRUE),
(3, 'USD000TSTTOM', 1, 75.00, 'above', TRUE),
(4, 'EUR_RUB_TOM', 1, 90.00, 'below', TRUE);

COMMIT;

