-- List active privacy scopes for schema contract
SELECT
    id,
    name,
    description,
    is_builtin,
    is_active
FROM privacy_scopes
WHERE is_active = TRUE
ORDER BY name;

