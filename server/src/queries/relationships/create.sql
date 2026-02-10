-- Create new relationship (polymorphic)
INSERT INTO relationships (
    source_type,
    source_id,
    target_type,
    target_id,
    type_id,
    status_id,
    properties
)
VALUES ($1, $2::uuid, $3, $4::uuid, $5, $6, $7::jsonb)
RETURNING 
    id, source_type, source_id, target_type, target_id,
    type_id, status_id, properties, created_at;
