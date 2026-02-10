-- Retrieve knowledge item by id
SELECT 
    k.id,
    k.title,
    k.url,
    k.source_type,
    k.content,
    k.privacy_scope_ids,
    s.name AS status,
    k.tags,
    k.metadata,
    k.vault_file_path,
    k.created_at,
    k.updated_at
FROM knowledge_items k
JOIN statuses s ON k.status_id = s.id
WHERE 
    k.id = $1::uuid
    AND ($2::uuid[] IS NULL OR k.privacy_scope_ids && $2);
