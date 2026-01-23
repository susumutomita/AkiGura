-- Seed grounds data for all municipalities
-- This ensures grounds table has data for all scrapers

-- Hiratsuka grounds
INSERT OR IGNORE INTO grounds (id, municipality_id, name, court_pattern, enabled, created_at) VALUES
('b0ddfeca-20c7-45d4-84c5-a70843ca8070', '1438eb89-e0d1-4b49-b9a2-135718c207e2', '平塚球場', '平塚球場', 1, CURRENT_TIMESTAMP),
('bb06ac2a-9a63-4e08-95a4-43c887a54bd0', '1438eb89-e0d1-4b49-b9a2-135718c207e2', '馬入ふれあい公園サッカー場', '馬入', 1, CURRENT_TIMESTAMP),
('56849576-908b-4bfb-bc30-ac537fd1cb5c', '1438eb89-e0d1-4b49-b9a2-135718c207e2', '総合公園野球場', '総合公園', 1, CURRENT_TIMESTAMP),
('d4d5ba7a-5d47-472f-b3d7-c62c94c961dd', '1438eb89-e0d1-4b49-b9a2-135718c207e2', '大神スポーツ広場野球場', '大神スポーツ広場', 1, CURRENT_TIMESTAMP),
('ef58b68c-8992-402f-9092-bf469d9b891f', '1438eb89-e0d1-4b49-b9a2-135718c207e2', '大神グラウンド野球場', '大神グラウンド野球場', 1, CURRENT_TIMESTAMP);

-- Fujisawa grounds
INSERT OR IGNORE INTO grounds (id, municipality_id, name, court_pattern, enabled, created_at) VALUES
('842ab6ef-e69e-4c61-a584-9b582285a1fa', 'af168e03-6f16-455b-8319-000f6d2e32bf', '八部球場', '八部', 1, CURRENT_TIMESTAMP),
('cc39fe86-c23a-4fa2-9a86-697127ac8da3', 'af168e03-6f16-455b-8319-000f6d2e32bf', '秋葉台球場', '秋葉台', 1, CURRENT_TIMESTAMP),
('14f9cc81-ae8f-4cc4-b199-98f28d272c18', 'af168e03-6f16-455b-8319-000f6d2e32bf', '引地台球場', '引地台', 1, CURRENT_TIMESTAMP),
('b20b1635-37a2-464e-934c-c06eb948ed7b', 'af168e03-6f16-455b-8319-000f6d2e32bf', '辻堂南部公園野球場', '辻堂', 1, CURRENT_TIMESTAMP),
('6cfb534f-363a-4c6a-b5ae-ab181b9375ee', 'af168e03-6f16-455b-8319-000f6d2e32bf', '長久保公園野球場', '長久保', 1, CURRENT_TIMESTAMP);

-- Yokohama grounds
INSERT OR IGNORE INTO grounds (id, municipality_id, name, court_pattern, enabled, created_at) VALUES
('7b18fcbf-944e-45bf-9ba9-3e8cf8593a0c', 'e7f6a658-4549-4761-8b77-b316576b22d6', 'こども自然公園', 'こども自然公園', 1, CURRENT_TIMESTAMP),
('7057dad4-e53a-46b8-9e4c-74697a53480d', 'e7f6a658-4549-4761-8b77-b316576b22d6', '今川公園', '今川公園', 1, CURRENT_TIMESTAMP),
('462461c0-7c15-43c4-8783-83b953619510', 'e7f6a658-4549-4761-8b77-b316576b22d6', '岡村公園', '岡村公園', 1, CURRENT_TIMESTAMP),
('5a6f0ebf-0f90-4e6a-b067-b7a4fd13bebc', 'e7f6a658-4549-4761-8b77-b316576b22d6', '俣野公園', '俣野公園', 1, CURRENT_TIMESTAMP),
('9299d1c0-8ffd-4660-85e8-1dc0467e7ce9', 'e7f6a658-4549-4761-8b77-b316576b22d6', '金井公園', '金井公園', 1, CURRENT_TIMESTAMP);

-- Kamakura grounds
INSERT OR IGNORE INTO grounds (id, municipality_id, name, court_pattern, enabled, created_at) VALUES
('a1b2c3d4-1111-4222-8333-444455556666', 'c8e4f9a1-2345-4678-9abc-def012345678', '笛田公園野球場', '笛田', 1, CURRENT_TIMESTAMP);

-- Kanagawa prefecture grounds
INSERT OR IGNORE INTO grounds (id, municipality_id, name, court_pattern, enabled, created_at) VALUES
('b2c3d4e5-2222-5333-9444-555566667777', 'd9e5f0b2-3456-5789-0bcd-ef0123456789', '保土ヶ谷公園硬式野球場', '保土ヶ谷', 1, CURRENT_TIMESTAMP),
('c3d4e5f6-3333-6444-0555-666677778888', 'd9e5f0b2-3456-5789-0bcd-ef0123456789', 'ラバーボール球場A面', 'ラバー.*A', 1, CURRENT_TIMESTAMP),
('d4e5f6a7-4444-7555-1666-777788889999', 'd9e5f0b2-3456-5789-0bcd-ef0123456789', 'ラバーボール球場B面', 'ラバー.*B', 1, CURRENT_TIMESTAMP);

-- Ayase grounds
INSERT OR IGNORE INTO grounds (id, municipality_id, name, court_pattern, enabled, created_at) VALUES
('e5f6a7b8-5555-8666-2777-888899990000', 'e0f1a2b3-4567-6890-1cde-f01234567890', '綾瀬ノーブルスタジアム', 'ノーブル', 1, CURRENT_TIMESTAMP);

-- Record execution of this migration
INSERT OR IGNORE INTO migrations (migration_number, migration_name)
VALUES (014, '014-grounds-seed-data');
