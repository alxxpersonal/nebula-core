-- Create approval request
INSERT INTO approval_requests (
  request_type,
  requested_by,
  change_details,
  job_id
)
VALUES ($1, $2, $3, $4)
RETURNING *;
