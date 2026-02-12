-- Candidate knowledge items for semantic ranking.
SELECT
    k.id,
    k.title,
    k.source_type,
    k.content,
    k.tags,
    k.metadata,
    k.privacy_scope_ids,
    k.created_at,
    k.updated_at
FROM knowledge_items k
JOIN statuses s ON s.id = k.status_id
WHERE
    s.category = 'active'
    AND ($1::uuid[] IS NULL OR k.privacy_scope_ids && $1)
ORDER BY k.updated_at DESC
LIMIT $2;
