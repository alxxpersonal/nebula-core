-- Update file metadata entry
WITH updated AS (
    UPDATE files
    SET
        filename = COALESCE($2, filename),
        file_path = COALESCE($3, file_path),
        mime_type = COALESCE($4, mime_type),
        size_bytes = COALESCE($5, size_bytes),
        checksum = COALESCE($6, checksum),
        status_id = COALESCE($7, status_id),
        tags = COALESCE($8, tags),
        metadata = COALESCE($9::jsonb, metadata)
    WHERE id = $1
    RETURNING *
)
SELECT
    u.id,
    u.filename,
    u.file_path,
    u.mime_type,
    u.size_bytes,
    u.checksum,
    s.name AS status,
    u.tags,
    u.metadata,
    u.created_at,
    u.updated_at
FROM updated u
JOIN statuses s ON u.status_id = s.id;
