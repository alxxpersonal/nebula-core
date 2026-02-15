-- List active entity types for schema contract
SELECT
    id,
    name,
    description,
    is_builtin,
    is_active
FROM entity_types
WHERE is_active = TRUE
ORDER BY name;

