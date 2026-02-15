-- List active relationship types for schema contract
SELECT
    id,
    name,
    description,
    is_symmetric,
    is_builtin,
    is_active
FROM relationship_types
WHERE is_active = TRUE
ORDER BY name;

