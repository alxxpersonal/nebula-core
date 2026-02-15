-- Create enrollment session linked to an approval request
INSERT INTO agent_enrollment_sessions (
  agent_id,
  approval_request_id,
  status,
  enrollment_token_hash,
  requested_scope_ids,
  requested_requires_approval,
  expires_at
)
VALUES ($1::uuid, $2::uuid, 'pending_approval', $3, $4::uuid[], $5, $6)
RETURNING *;
