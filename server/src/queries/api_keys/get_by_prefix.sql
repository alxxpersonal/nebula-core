-- Get active API key by prefix for authentication
SELECT id, entity_id, agent_id, key_hash, scopes, revoked_at, expires_at
FROM api_keys
WHERE key_prefix = $1
  AND revoked_at IS NULL
  AND (expires_at IS NULL OR expires_at > NOW())
