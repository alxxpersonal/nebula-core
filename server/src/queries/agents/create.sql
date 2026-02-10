-- Create new agent
INSERT INTO agents (name, description, scopes, capabilities, status_id, requires_approval)
VALUES ($1, $2, $3, $4::text[], $5, $6)
RETURNING *;
