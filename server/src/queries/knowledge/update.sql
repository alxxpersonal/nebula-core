-- Update knowledge item fields
WITH updated AS (
    UPDATE knowledge_items
    SET
        title = COALESCE($2, title),
        url = COALESCE($3, url),
        source_type = COALESCE($4, source_type),
        content = COALESCE($5, content),
        status_id = COALESCE($6, status_id),
        tags = COALESCE($7, tags),
        privacy_scope_ids = COALESCE($8, privacy_scope_ids),
        metadata = COALESCE($9::jsonb, metadata)
    WHERE id = $1
    RETURNING *
)
SELECT
    u.id,
    u.title,
    u.url,
    u.source_type,
    u.content,
    u.privacy_scope_ids,
    s.name AS status,
    u.tags,
    u.metadata,
    u.vault_file_path,
    u.created_at,
    u.updated_at
FROM updated u
JOIN statuses s ON u.status_id = s.id;
