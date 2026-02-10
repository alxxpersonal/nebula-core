-- Create API key for entity (user login)
INSERT INTO api_keys (entity_id, key_hash, key_prefix, name)
VALUES ($1, $2, $3, $4)
