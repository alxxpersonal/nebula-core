-- Update job status with optional completion timestamp
UPDATE jobs
SET 
    status_id = $2,
    status_reason = $3,
    status_changed_at = NOW(),
    completed_at = COALESCE($4::timestamptz, completed_at)
WHERE id = $1
RETURNING 
    id, title, status_id, status_reason, 
    status_changed_at, completed_at, updated_at;
