-- Add Hiratsuka municipality and grounds

-- Hiratsuka municipality
INSERT OR IGNORE INTO municipalities (id, name, scraper_type, url, enabled, created_at)
VALUES ('1438eb89-e0d1-4b49-b9a2-135718c207e2', '平塚市', 'hiratsuka', 'https://www.city.hiratsuka.kanagawa.jp', 1, CURRENT_TIMESTAMP);

-- Hiratsuka grounds
INSERT OR IGNORE INTO grounds (id, municipality_id, name, court_pattern, enabled, created_at) VALUES
('b0ddfeca-20c7-45d4-84c5-a70843ca8070', '1438eb89-e0d1-4b49-b9a2-135718c207e2', '平塚球場', '平塚球場', 1, CURRENT_TIMESTAMP),
('bb06ac2a-9a63-4e08-95a4-43c887a54bd0', '1438eb89-e0d1-4b49-b9a2-135718c207e2', '馬入ふれあい公園サッカー場', '馬入', 1, CURRENT_TIMESTAMP),
('56849576-908b-4bfb-bc30-ac537fd1cb5c', '1438eb89-e0d1-4b49-b9a2-135718c207e2', '総合公園野球場', '総合公園', 1, CURRENT_TIMESTAMP),
('d4d5ba7a-5d47-472f-b3d7-c62c94c961dd', '1438eb89-e0d1-4b49-b9a2-135718c207e2', '大神スポーツ広場野球場', '大神', 1, CURRENT_TIMESTAMP);

-- Record execution of this migration
INSERT OR IGNORE INTO migrations (migration_number, migration_name)
VALUES (004, '004-hiratsuka');
