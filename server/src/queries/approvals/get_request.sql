-- Get approval request by id
SELECT 
  ar.*,
  a.name as agent_name,
  e.name as reviewer_name
FROM approval_requests ar
LEFT JOIN agents a ON ar.requested_by = a.id
LEFT JOIN entities e ON ar.reviewed_by = e.id
WHERE ar.id = $1;
