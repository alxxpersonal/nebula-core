-- Detect cycles for a relationship type
WITH RECURSIVE rel_path AS (
    SELECT
        r.source_id,
        r.target_id,
        1 AS depth
    FROM relationships r
    WHERE r.source_type = $1
      AND r.target_type = $2
      AND r.type_id = $3
      AND r.source_id = $4
    UNION ALL
    SELECT
        r.source_id,
        r.target_id,
        p.depth + 1
    FROM relationships r
    JOIN rel_path p ON r.source_id = p.target_id
    WHERE r.source_type = $1
      AND r.target_type = $2
      AND r.type_id = $3
      AND p.depth < $5
)
SELECT 1
FROM rel_path
WHERE target_id = $6
LIMIT 1;
