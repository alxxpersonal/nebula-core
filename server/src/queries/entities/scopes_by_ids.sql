-- Fetch entity privacy scopes for a list of ids
SELECT
    id,
    privacy_scope_ids
FROM entities
WHERE id = ANY($1::uuid[]);
