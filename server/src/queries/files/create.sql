-- Create file metadata entry
INSERT INTO files (
    filename,
    file_path,
    mime_type,
    size_bytes,
    checksum,
    status_id,
    tags,
    metadata
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb)
RETURNING *;
