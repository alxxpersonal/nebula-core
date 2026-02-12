SELECT COUNT(*)::INT AS usage_count
FROM relationships
WHERE type_id = $1::UUID;
