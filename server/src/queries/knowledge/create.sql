-- Create new knowledge item
INSERT INTO knowledge_items (
    title,
    url,
    source_type,
    content,
    privacy_scope_ids,
    status_id,
    tags,
    metadata
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb)
RETURNING 
    id, title, url, source_type, content,
    privacy_scope_ids, status_id, tags, metadata, created_at;
