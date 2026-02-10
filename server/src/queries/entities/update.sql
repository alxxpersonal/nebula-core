-- Update entity metadata, tags, or status
UPDATE entities
SET 
    metadata = COALESCE($2::jsonb, metadata),
    tags = COALESCE($3::text[], tags),
    status_id = COALESCE($4::uuid, status_id),
    status_reason = COALESCE($5::text, status_reason),
    status_changed_at = CASE WHEN $4::uuid IS NOT NULL THEN NOW() ELSE status_changed_at END
WHERE id = $1::uuid
RETURNING 
    id, name, type_id, status_id, privacy_scope_ids, 
    tags, metadata, status_reason, updated_at;
