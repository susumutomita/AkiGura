-- Add Kamakura, Kanagawa, and Ayase municipalities and grounds

-- Kamakura municipality
INSERT OR IGNORE INTO municipalities (id, name, scraper_type, url, enabled, created_at)
VALUES ('c8e4f9a1-2345-4678-9abc-def012345678', '鎌倉市', 'kamakura', 'https://yoyaku.e-kanagawa.lg.jp', 1, CURRENT_TIMESTAMP);

-- Kamakura grounds
INSERT OR IGNORE INTO grounds (id, municipality_id, name, court_pattern, enabled, created_at) VALUES
('a1b2c3d4-1111-4222-8333-444455556666', 'c8e4f9a1-2345-4678-9abc-def012345678', '笛田公園野球場', '笛田', 1, CURRENT_TIMESTAMP);

-- Kanagawa prefecture municipality
INSERT OR IGNORE INTO municipalities (id, name, scraper_type, url, enabled, created_at)
VALUES ('d9e5f0b2-3456-5789-0bcd-ef0123456789', '神奈川県', 'kanagawa', 'https://yoyaku.e-kanagawa.lg.jp', 1, CURRENT_TIMESTAMP);

-- Kanagawa prefecture grounds
INSERT OR IGNORE INTO grounds (id, municipality_id, name, court_pattern, enabled, created_at) VALUES
('b2c3d4e5-2222-5333-9444-555566667777', 'd9e5f0b2-3456-5789-0bcd-ef0123456789', '保土ヶ谷公園硬式野球場', '保土ヶ谷', 1, CURRENT_TIMESTAMP),
('c3d4e5f6-3333-6444-0555-666677778888', 'd9e5f0b2-3456-5789-0bcd-ef0123456789', 'ラバーボール球場A面', 'ラバー.*A', 1, CURRENT_TIMESTAMP),
('d4e5f6a7-4444-7555-1666-777788889999', 'd9e5f0b2-3456-5789-0bcd-ef0123456789', 'ラバーボール球場B面', 'ラバー.*B', 1, CURRENT_TIMESTAMP);

-- Ayase municipality
INSERT OR IGNORE INTO municipalities (id, name, scraper_type, url, enabled, created_at)
VALUES ('e0f1a2b3-4567-6890-1cde-f01234567890', '綾瀬市', 'ayase', 'https://www.ayaseins.jp', 1, CURRENT_TIMESTAMP);

-- Ayase grounds
INSERT OR IGNORE INTO grounds (id, municipality_id, name, court_pattern, enabled, created_at) VALUES
('e5f6a7b8-5555-8666-2777-888899990000', 'e0f1a2b3-4567-6890-1cde-f01234567890', '綾瀬ノーブルスタジアム', 'ノーブル', 1, CURRENT_TIMESTAMP);

-- Record execution of this migration
INSERT OR IGNORE INTO migrations (migration_number, migration_name)
VALUES (009, '009-kamakura-kanagawa-ayase');
