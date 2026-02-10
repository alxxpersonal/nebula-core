-- Get entity type_id by entity id
SELECT type_id FROM entities WHERE id = $1::uuid
