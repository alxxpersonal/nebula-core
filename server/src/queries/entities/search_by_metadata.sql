-- Search entities by JSONB metadata containment
SELECT 
    e.id,
    e.name,
    et.name AS type,
    s.name AS status,
    e.privacy_scope_ids,
    e.tags,
    e.metadata,
    e.created_at
FROM entities e
JOIN entity_types et ON e.type_id = et.id
JOIN statuses s ON e.status_id = s.id
WHERE 
    e.metadata @> $1::jsonb
    AND s.category = 'active'
LIMIT $2;
