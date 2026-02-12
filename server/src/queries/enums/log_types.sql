-- List log types
SELECT id, name
FROM log_types
WHERE is_active = TRUE
ORDER BY name;
