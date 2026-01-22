-- Seed municipality data that was missing from facilities migration
-- This ensures municipalities table has proper data for all scrapers

INSERT OR IGNORE INTO municipalities (id, name, scraper_type, url, enabled)
VALUES
    ('e7f6a658-4549-4761-8b77-b316576b22d6', '横浜市', 'yokohama', 'https://yoyaku.city.yokohama.lg.jp/', 1),
    ('c8e4f9a1-2345-4678-9abc-def012345678', '鎌倉市', 'kamakura', 'https://shisetsu.city.kamakura.kanagawa.jp/', 1),
    ('d9e5f0b2-3456-5789-0bcd-ef0123456789', '神奈川県', 'kanagawa', 'https://www.e-kanagawa.lg.jp/', 1),
    ('e0f1a2b3-4567-6890-1cde-f01234567890', '綾瀬市', 'ayase', 'https://www.lics-saas.nexs-service.jp/ayase/', 1),
    ('1438eb89-e0d1-4b49-b9a2-135718c207e2', '平塚市', 'hiratsuka', 'https://www.hiratsuka-shimin.jp/', 1),
    ('af168e03-6f16-455b-8319-000f6d2e32bf', '藤沢市', 'fujisawa', 'https://www.fujisawa-shisetsu.jp/', 1);

-- Record execution of this migration
INSERT OR IGNORE INTO migrations (migration_number, migration_name)
VALUES (013, '013-municipality-seed-data');
