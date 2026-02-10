-- List approval requests requested by agent
SELECT 
  ar.*,
  e.name as reviewer_name
FROM approval_requests ar
LEFT JOIN entities e ON ar.reviewed_by = e.id
WHERE ar.requested_by = $1
ORDER BY ar.created_at DESC
LIMIT $2;
