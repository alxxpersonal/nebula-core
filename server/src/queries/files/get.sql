-- Retrieve file by ID
SELECT
    f.id,
    f.filename,
    f.file_path,
    f.mime_type,
    f.size_bytes,
    f.checksum,
    s.name AS status,
    f.tags,
    f.metadata,
    f.created_at,
    f.updated_at
FROM files f
JOIN statuses s ON f.status_id = s.id
WHERE f.id = $1;
