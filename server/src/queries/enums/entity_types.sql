-- List entity types
SELECT id, name
FROM entity_types
WHERE is_active = TRUE
ORDER BY name;
