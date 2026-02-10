-- Get active agent by id
SELECT a.*, s.name AS status_name, s.category AS status_category
FROM agents a
JOIN statuses s ON a.status_id = s.id
WHERE a.id = $1 AND s.category = 'active';
