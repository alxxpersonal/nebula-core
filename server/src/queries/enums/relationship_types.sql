-- List relationship types
SELECT id, name
FROM relationship_types
WHERE is_active = TRUE
ORDER BY name;
