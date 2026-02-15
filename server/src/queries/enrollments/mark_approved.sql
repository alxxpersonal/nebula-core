-- Mark enrollment session approved with final reviewer grants
UPDATE agent_enrollment_sessions
SET
  status = 'approved',
  granted_scope_ids = $2::uuid[],
  granted_requires_approval = $3,
  approved_by = $4::uuid,
  approved_at = NOW(),
  rejected_reason = NULL
WHERE approval_request_id = $1::uuid
  AND status = 'pending_approval'
RETURNING *;
