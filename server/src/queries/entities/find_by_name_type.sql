-- Find person entity by name and type
SELECT id FROM entities WHERE name = $1 AND type_id = $2
