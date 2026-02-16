-- Revoke API key for agent
UPDATE api_keys
SET revoked_at = NOW()
WHERE id = $1::uuid
  AND agent_id = $2::uuid
  AND revoked_at IS NULL;
