-- Reject approval request
UPDATE approval_requests
SET 
  status = 'rejected',
  reviewed_by = $2,
  reviewed_at = NOW(),
  review_notes = $3
WHERE id = $1
  AND status = 'pending'
RETURNING *;
