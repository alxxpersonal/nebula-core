-- Update relationship properties or status
UPDATE relationships
SET 
    properties = COALESCE($2::jsonb, properties),
    status_id = COALESCE($3::uuid, status_id)
WHERE id = $1::uuid
RETURNING 
    id, source_type, source_id, target_type, target_id,
    type_id, status_id, properties, updated_at;
