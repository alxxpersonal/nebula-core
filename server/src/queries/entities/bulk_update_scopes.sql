-- Bulk update entity privacy scopes
WITH updated AS (
    UPDATE entities
    SET privacy_scope_ids = CASE
        WHEN $2::text = 'set' THEN $3::uuid[]
        WHEN $2::text = 'add' THEN (
            SELECT ARRAY(
                SELECT DISTINCT unnest(COALESCE(privacy_scope_ids, '{}'::uuid[]) || $3::uuid[])
            )
        )
        WHEN $2::text = 'remove' THEN (
            SELECT ARRAY(
                SELECT unnest(COALESCE(privacy_scope_ids, '{}'::uuid[]))
                EXCEPT SELECT unnest($3::uuid[])
            )
        )
        ELSE COALESCE(privacy_scope_ids, '{}'::uuid[])
    END,
    updated_at = now()
    WHERE id = ANY($1::uuid[])
    RETURNING id
)
SELECT id FROM updated;
