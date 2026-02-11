-- Update protocol
UPDATE protocols
SET
    title = COALESCE($2, title),
    version = COALESCE($3, version),
    content = COALESCE($4, content),
    protocol_type = COALESCE($5, protocol_type),
    applies_to = COALESCE($6, applies_to),
    status_id = COALESCE($7, status_id),
    tags = COALESCE($8, tags),
    trusted = COALESCE($9, trusted),
    metadata = COALESCE($10::jsonb, metadata),
    vault_file_path = COALESCE($11, vault_file_path)
WHERE name = $1
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
    trusted,
    metadata,
    vault_file_path,
    created_at,
    updated_at;
