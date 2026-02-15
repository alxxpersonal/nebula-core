-- List active log types for schema contract
SELECT
    id,
    name,
    description,
    value_schema,
    is_builtin,
    is_active
FROM log_types
WHERE is_active = TRUE
ORDER BY name;

