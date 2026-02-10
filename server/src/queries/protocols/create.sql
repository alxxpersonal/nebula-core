-- Create protocol
INSERT INTO protocols (
    name,
    title,
    version,
    content,
    protocol_type,
    applies_to,
    status_id,
    tags,
    metadata,
    vault_file_path
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10)
RETURNING
    id,
    name,
    title,
    version,
    content,
    protocol_type,
    applies_to,
    status_id,
    tags,
    metadata,
    vault_file_path,
    created_at,
    updated_at;
