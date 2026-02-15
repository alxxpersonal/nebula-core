-- Approve approval request
UPDATE approval_requests
SET 
  status = 'approved',
  reviewed_by = $2,
  reviewed_at = NOW(),
  review_details = COALESCE($3::jsonb, '{}'::jsonb),
  review_notes = COALESCE($4, review_notes)
WHERE id = $1
  AND status = 'pending'
RETURNING *;
