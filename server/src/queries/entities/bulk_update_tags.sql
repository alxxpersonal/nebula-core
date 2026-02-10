-- Bulk update entity tags
WITH updated AS (
    UPDATE entities
    SET tags = CASE
        WHEN $2::text = 'set' THEN $3::text[]
        WHEN $2::text = 'add' THEN (
            SELECT ARRAY(
                SELECT DISTINCT unnest(COALESCE(tags, '{}'::text[]) || $3::text[])
            )
        )
        WHEN $2::text = 'remove' THEN (
            SELECT ARRAY(
                SELECT unnest(COALESCE(tags, '{}'::text[]))
                EXCEPT SELECT unnest($3::text[])
            )
        )
        ELSE COALESCE(tags, '{}'::text[])
    END,
    updated_at = now()
    WHERE id = ANY($1::uuid[])
    RETURNING id
)
SELECT id FROM updated;
