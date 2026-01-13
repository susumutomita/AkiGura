-- Add Fujisawa grounds

-- Fujisawa grounds
INSERT OR IGNORE INTO grounds (id, municipality_id, name, court_pattern, enabled, created_at) VALUES
('842ab6ef-e69e-4c61-a584-9b582285a1fa', 'af168e03-6f16-455b-8319-000f6d2e32bf', '八部球場', '八部', 1, CURRENT_TIMESTAMP),
('cc39fe86-c23a-4fa2-9a86-697127ac8da3', 'af168e03-6f16-455b-8319-000f6d2e32bf', '秋葉台球場', '秋葉台', 1, CURRENT_TIMESTAMP),
('14f9cc81-ae8f-4cc4-b199-98f28d272c18', 'af168e03-6f16-455b-8319-000f6d2e32bf', '引地台球場', '引地台', 1, CURRENT_TIMESTAMP),
('b20b1635-37a2-464e-934c-c06eb948ed7b', 'af168e03-6f16-455b-8319-000f6d2e32bf', '辻堂南部公園野球場', '辻堂', 1, CURRENT_TIMESTAMP),
('6cfb534f-363a-4c6a-b5ae-ab181b9375ee', 'af168e03-6f16-455b-8319-000f6d2e32bf', '長久保公園野球場', '長久保', 1, CURRENT_TIMESTAMP);

-- Record execution of this migration
INSERT OR IGNORE INTO migrations (migration_number, migration_name)
VALUES (005, '005-fujisawa-grounds');
