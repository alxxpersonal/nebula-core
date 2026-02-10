-- Create new log entry
WITH inserted AS (
    INSERT INTO logs (
        log_type_id,
        timestamp,
        value,
        status_id,
        tags,
        metadata
    )
    VALUES ($1, $2, $3::jsonb, $4, $5, $6::jsonb)
    RETURNING *
)
SELECT
    i.id,
    lt.name AS log_type,
    i.timestamp,
    i.value,
    s.name AS status,
    i.tags,
    i.metadata,
    i.created_at,
    i.updated_at
FROM inserted i
JOIN log_types lt ON i.log_type_id = lt.id
JOIN statuses s ON i.status_id = s.id;
