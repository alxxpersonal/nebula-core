-- Create new API key for agent
INSERT INTO api_keys (agent_id, key_hash, key_prefix, name)
VALUES ($1::uuid, $2, $3, $4)
