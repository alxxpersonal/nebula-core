SELECT COUNT(*)::INT AS usage_count
FROM logs
WHERE log_type_id = $1::UUID;
