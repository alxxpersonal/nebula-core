-- Create API key for entity with returning
INSERT INTO api_keys (entity_id, key_hash, key_prefix, name)
VALUES ($1, $2, $3, $4)
RETURNING id, key_prefix, name, created_at
