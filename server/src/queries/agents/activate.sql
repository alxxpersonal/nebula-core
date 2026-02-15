-- Activate agent and apply final scope/trust grants
UPDATE agents
SET
  status_id = $1,
  scopes = COALESCE($2::uuid[], scopes),
  requires_approval = COALESCE($3, requires_approval)
WHERE id = $4::uuid
RETURNING *;
