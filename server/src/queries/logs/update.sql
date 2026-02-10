-- Update log entry
WITH updated AS (
    UPDATE logs
    SET
        log_type_id = COALESCE($2, log_type_id),
        timestamp = COALESCE($3, timestamp),
        value = COALESCE($4::jsonb, value),
        status_id = COALESCE($5, status_id),
        tags = COALESCE($6, tags),
        metadata = COALESCE($7::jsonb, metadata)
    WHERE id = $1
    RETURNING *
)
SELECT
    u.id,
    lt.name AS log_type,
    u.timestamp,
    u.value,
    s.name AS status,
    u.tags,
    u.metadata,
    u.created_at,
    u.updated_at
FROM updated u
JOIN log_types lt ON u.log_type_id = lt.id
JOIN statuses s ON u.status_id = s.id;
