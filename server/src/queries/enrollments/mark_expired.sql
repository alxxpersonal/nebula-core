-- Expire pending or approved enrollment sessions once TTL elapsed
UPDATE agent_enrollment_sessions
SET status = 'expired'
WHERE id = $1::uuid
  AND status IN ('pending_approval', 'approved')
RETURNING *;
