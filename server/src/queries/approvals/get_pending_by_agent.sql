-- List pending approval requests for agent
SELECT 
  ar.*,
  a.name as agent_name
FROM approval_requests ar
LEFT JOIN agents a ON ar.requested_by = a.id
WHERE ar.status = 'pending'
  AND ar.requested_by = $1
ORDER BY ar.created_at ASC;
