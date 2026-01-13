-- Add Yokohama municipality and grounds

-- Yokohama municipality
INSERT OR IGNORE INTO municipalities (id, name, scraper_type, url, enabled, created_at)
VALUES ('e7f6a658-4549-4761-8b77-b316576b22d6', '横浜市', 'yokohama', 'https://www.shisetsu.city.yokohama.lg.jp', 1, CURRENT_TIMESTAMP);

-- Yokohama grounds (baseball fields in various parks)
INSERT OR IGNORE INTO grounds (id, municipality_id, name, court_pattern, enabled, created_at) VALUES
('7b18fcbf-944e-45bf-9ba9-3e8cf8593a0c', 'e7f6a658-4549-4761-8b77-b316576b22d6', 'こども自然公園', 'こども自然公園', 1, CURRENT_TIMESTAMP),
('7057dad4-e53a-46b8-9e4c-74697a53480d', 'e7f6a658-4549-4761-8b77-b316576b22d6', '今川公園', '今川公園', 1, CURRENT_TIMESTAMP),
('462461c0-7c15-43c4-8783-83b953619510', 'e7f6a658-4549-4761-8b77-b316576b22d6', '岡村公園', '岡村公園', 1, CURRENT_TIMESTAMP),
('5a6f0ebf-0f90-4e6a-b067-b7a4fd13bebc', 'e7f6a658-4549-4761-8b77-b316576b22d6', '俣野公園', '俣野公園', 1, CURRENT_TIMESTAMP),
('9299d1c0-8ffd-4660-85e8-1dc0467e7ce9', 'e7f6a658-4549-4761-8b77-b316576b22d6', '金井公園', '金井公園', 1, CURRENT_TIMESTAMP);

-- Record execution of this migration
INSERT OR IGNORE INTO migrations (migration_number, migration_name)
VALUES (008, '008-yokohama');
