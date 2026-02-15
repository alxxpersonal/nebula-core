-- Get enrollment session by registration id
SELECT
  aes.*,
  a.name AS agent_name,
  ar.review_notes,
  ar.request_type
FROM agent_enrollment_sessions aes
JOIN agents a ON a.id = aes.agent_id
JOIN approval_requests ar ON ar.id = aes.approval_request_id
WHERE aes.id = $1::uuid;
