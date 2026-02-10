-- List agents by status category
SELECT 
    a.id,
    a.name,
    a.description,
    a.scopes,
    a.capabilities,
    s.name AS status,
    a.requires_approval,
    a.created_at
FROM agents a
JOIN statuses s ON a.status_id = s.id
WHERE s.category = $1
ORDER BY a.name;
