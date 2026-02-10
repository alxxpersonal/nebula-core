-- Get relationships for an item with direction filter
SELECT 
    r.id,
    r.source_type,
    r.source_id,
    r.target_type,
    r.target_id,
    rt.name AS relationship_type,
    s.name AS status,
    r.properties,
    r.created_at
FROM relationships r
JOIN relationship_types rt ON r.type_id = rt.id
JOIN statuses s ON r.status_id = s.id
WHERE 
    CASE 
        WHEN $3 = 'outgoing' THEN r.source_type = $1 AND r.source_id = $2
        WHEN $3 = 'incoming' THEN r.target_type = $1 AND r.target_id = $2
        ELSE (r.source_type = $1 AND r.source_id = $2)
             OR (r.target_type = $1 AND r.target_id = $2)
    END
    AND ($4::text IS NULL OR rt.name = $4)
    AND s.category = 'active'
ORDER BY r.created_at DESC;
