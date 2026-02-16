-- List active API keys for agent
SELECT id, key_prefix, name, last_used_at, expires_at, created_at
FROM api_keys
WHERE agent_id = $1::uuid AND revoked_at IS NULL
ORDER BY created_at DESC;
