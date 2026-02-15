-- Mark enrollment session rejected
UPDATE agent_enrollment_sessions
SET
  status = 'rejected',
  rejected_reason = $2,
  approved_by = $3::uuid,
  approved_at = NOW()
WHERE approval_request_id = $1::uuid
  AND status = 'pending_approval'
RETURNING *;
