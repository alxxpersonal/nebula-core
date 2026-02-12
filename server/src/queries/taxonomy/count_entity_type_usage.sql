SELECT COUNT(*)::INT AS usage_count
FROM entities
WHERE type_id = $1::UUID;
