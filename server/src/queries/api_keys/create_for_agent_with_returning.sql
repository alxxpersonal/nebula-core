-- Create API key for agent with returning fields
INSERT INTO api_keys (agent_id, key_hash, key_prefix, name)
VALUES ($1::uuid, $2, $3, $4)
RETURNING id, key_prefix, name, created_at;
