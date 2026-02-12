-- List privacy scopes
SELECT id, name
FROM privacy_scopes
WHERE is_active = TRUE
ORDER BY name;
