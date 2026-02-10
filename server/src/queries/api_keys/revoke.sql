-- Revoke API key for entity
UPDATE api_keys SET revoked_at = NOW()
WHERE id = $1::uuid AND entity_id = $2 AND revoked_at IS NULL
