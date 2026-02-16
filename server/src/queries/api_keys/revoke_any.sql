-- Revoke API key by id (admin path)
UPDATE api_keys
SET revoked_at = NOW()
WHERE id = $1::uuid
  AND revoked_at IS NULL;
