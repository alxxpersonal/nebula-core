-- List active API keys for entity
SELECT id, key_prefix, name, last_used_at, expires_at, created_at
FROM api_keys
WHERE entity_id = $1 AND revoked_at IS NULL
ORDER BY created_at DESC
