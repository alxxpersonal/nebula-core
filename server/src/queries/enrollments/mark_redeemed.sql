-- Mark enrollment session redeemed (one-time key issued)
UPDATE agent_enrollment_sessions
SET
  status = 'redeemed',
  redeemed_at = NOW()
WHERE id = $1::uuid
  AND status = 'approved'
RETURNING *;
